#!/usr/bin/env python3
"""Verify every signed category in a captured private Gloas FCR window."""
import argparse
import copy
import gzip
import inspect
import json
import subprocess
import sys
import urllib.error
from pathlib import Path


def verify(args):
    from eth_consensus_specs.gloas import mainnet as spec
    from eth_consensus_specs.utils import bls
    sys.path.insert(0, str(Path(__file__).parents[1] / "qualification/reference"))
    from ssz_rest import parsed
    from qualify_fast_confirmation import get

    source = Path(inspect.getfile(spec)).resolve()
    source_root = next(parent for parent in source.parents if (parent / ".git").exists())
    revision = subprocess.check_output(["git", "-C", str(source_root), "rev-parse", "HEAD"], text=True).strip()
    assert revision == "d2021d2719235f0df4726a1d330502cb15aa8ea3"
    assert not subprocess.check_output(["git", "-C", str(source_root), "diff", "--", "specs", "pysetup", "tests/core/pyspec/eth_consensus_specs/utils"], text=True)
    assert bls.bls_active
    root = args.capture
    profile = json.loads((root / "profile.json").read_text())
    result = json.loads((root / "result.json").read_text())
    assert result["passed"], "the actual node window must pass independently first"
    output = args.output
    output.mkdir(parents=True, exist_ok=False)
    base = profile["cl"]
    report = {"passed": False, "source_revision": revision, "bls_active": True, "first_slot": 1, "through_slot": result["final_slot"], "counts": {}, "mutation_controls": {}}
    counts = report["counts"]
    controls = report["mutation_controls"]

    def count(name):
        counts[name] = counts.get(name, 0) + 1

    def retained(name, value):
        data = json.dumps(value, separators=(",", ":")).encode()
        (output / (name + ".json.gz")).write_bytes(gzip.compress(data, mtime=0))

    def reject(name, check):
        if name in controls:
            return
        try:
            accepted = check()
        except (AssertionError, ValueError):
            accepted = False
        assert not accepted, "unchanged reference accepted a changed " + name
        controls[name] = True

    def changed_signature(value):
        data = bytearray(bytes(value)); data[-1] ^= 1
        return spec.BLSSignature(bytes(data))

    try:
        genesis = get(base + "/eth/v1/beacon/genesis")["data"]
        assert genesis == json.loads((root / "genesis.json").read_text())
        raw = get(base + "/eth/v2/debug/beacon/states/0")["data"]
        retained("genesis-state", raw)
        state0 = parsed(raw, spec.BeaconState)
        assert "0x" + bytes(state0.genesis_validators_root).hex() == genesis["genesis_validators_root"]
        header0 = get(base + "/eth/v1/beacon/headers/0")["data"]
        assert "0x" + bytes(spec.hash_tree_root(state0)).hex() == header0["header"]["message"]["state_root"]
        roster = [bytes(validator.pubkey) for validator in state0.validators]
        previous = header0["root"]
        for slot in range(1, result["final_slot"] + 1):
            try:
                block_response = get(base + "/eth/v2/beacon/blocks/" + str(slot))
            except urllib.error.HTTPError as error:
                if error.code == 404:
                    count("missed_slots")
                    continue
                raise
            raw_block = block_response["data"]
            block = parsed(raw_block, spec.SignedBeaconBlock)
            retained(f"block-{slot}", raw_block)
            block_root = "0x" + bytes(spec.hash_tree_root(block.message)).hex()
            assert "0x" + bytes(block.message.parent_root).hex() == previous
            state_root = "0x" + bytes(block.message.state_root).hex()
            raw_state = get(base + "/eth/v2/debug/beacon/states/" + state_root)["data"]
            retained(f"state-{slot}", raw_state)
            state = parsed(raw_state, spec.BeaconState)
            assert int(state.slot) == slot
            assert "0x" + bytes(spec.hash_tree_root(state)).hex() == state_root
            assert bytes(state.genesis_validators_root) == bytes(state0.genesis_validators_root)
            assert [bytes(validator.pubkey) for validator in state.validators] == roster
            proposer = roster[int(block.message.proposer_index)]
            epoch = spec.compute_epoch_at_slot(block.message.slot)
            signing = spec.compute_signing_root(block.message, spec.get_domain(state, spec.DOMAIN_BEACON_PROPOSER, epoch))
            assert bls.Verify(proposer, signing, block.signature)
            reject("proposer", lambda: bls.Verify(proposer, signing, changed_signature(block.signature)))
            count("proposer_signatures"); count("state_root_bindings")
            randao = block.message.body.randao_reveal
            signing = spec.compute_signing_root(epoch, spec.get_domain(state, spec.DOMAIN_RANDAO, epoch))
            assert bls.Verify(proposer, signing, randao)
            reject("randao", lambda: bls.Verify(proposer, signing, changed_signature(randao)))
            count("randao_signatures")
            for attestation in block.message.body.attestations:
                indexed = spec.get_indexed_attestation(state, attestation)
                assert spec.is_valid_indexed_attestation(state, indexed)
                altered = copy.deepcopy(indexed); altered.signature = changed_signature(indexed.signature)
                reject("attestation", lambda: spec.is_valid_indexed_attestation(state, altered))
                count("attestation_aggregates")
            for attestation in block.message.body.payload_attestations:
                indexed = spec.get_indexed_payload_attestation(state, attestation)
                assert spec.is_valid_indexed_payload_attestation(state, indexed)
                altered = copy.deepcopy(indexed); altered.signature = changed_signature(indexed.signature)
                reject("payload_attestation", lambda: spec.is_valid_indexed_payload_attestation(state, altered))
                count("payload_attestation_aggregates")
            sync = block.message.body.sync_aggregate
            spec.process_sync_aggregate(state.copy(), sync)
            if any(sync.sync_committee_bits):
                altered = copy.deepcopy(sync); altered.sync_committee_signature = changed_signature(sync.sync_committee_signature)
                def bad_sync():
                    spec.process_sync_aggregate(state.copy(), altered)
                    return True
                reject("sync_aggregate", bad_sync)
                count("sync_aggregates")
            else:
                count("empty_sync_aggregates")
            bid = block.message.body.signed_execution_payload_bid
            if bid.message.builder_index == spec.BUILDER_INDEX_SELF_BUILD:
                count("self_build_bids_without_separate_signature")
            else:
                assert spec.verify_execution_payload_bid_signature(state, bid)
                count("builder_bid_signatures")
            envelope_response = get(base + "/eth/v1/beacon/execution_payload_envelopes/" + block_root)
            retained(f"envelope-{slot}", envelope_response)
            envelope = parsed(envelope_response["data"], spec.SignedExecutionPayloadEnvelope)
            assert spec.verify_execution_payload_envelope_signature(state, envelope)
            altered = copy.deepcopy(envelope); altered.signature = changed_signature(envelope.signature)
            reject("execution_envelope", lambda: spec.verify_execution_payload_envelope_signature(state, altered))
            count("execution_envelope_signatures"); count("blocks")
            previous = block_root
        assert counts["blocks"] >= 80
        for category in ["proposer", "randao", "attestation", "payload_attestation", "sync_aggregate", "execution_envelope"]:
            assert controls.get(category), "missing real category/control: " + category
        report["passed"] = True
        report["scope"] = "Every available block in the declared private window: proposer, RANDAO, included attestation/PTC aggregates, sync aggregates and execution envelopes, with authenticated state-root/roster bindings. Self-build bids are not counted as separate BLS signatures. This is not a new 48h, scale or complete Ethereum STF acceptance."
    finally:
        (output / "result.json").write_text(json.dumps(report, indent=2) + "\n")
    print(json.dumps(report, indent=2))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--capture", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    verify(parser.parse_args())


if __name__ == "__main__":
    main()
