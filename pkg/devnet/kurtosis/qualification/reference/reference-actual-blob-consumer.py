import json, io, rlp
from eth_hash.auto import keccak
from pathlib import Path
from urllib.parse import urlsplit
from eth_consensus_specs.gloas import mainnet as spec
from eth_consensus_specs.fulu import mainnet as fulu
from eth_consensus_specs.utils import bls, kzg
from ssz_rest import parsed
import argparse

parser = argparse.ArgumentParser(
    description="Verify frozen public devnet inputs with the unchanged selected consensus reference."
)
parser.add_argument("--artifact-dir", required=True, type=Path)
root = parser.parse_args().artifact_dir
out = root / "attempt4-reference-actual-blob-consumer.json"
assert not out.exists()
report = {
    "spec_revision": "2ea9d96a420927eb1a35d7c9399c8b3691c69f6a",
    "passed": False,
    "bls_active": bls.bls_active,
    "nodes": [],
}


def save():
    out.write_text(json.dumps(report, indent=2) + "\n")


def hx(v):
    return "0x" + bytes(v).hex()


def h(v):
    return hx(spec.hash_tree_root(v))


def b(v):
    return bytes.fromhex(v[2:])


def rejected(fn):
    try:
        return not fn()
    except (AssertionError, ValueError, RuntimeError):
        return True


try:
    assert bls.bls_active
    data = json.loads((root / "attempt4-actual-blob-cl-consumers.json").read_text())
    genesis_bytes = (root / "attempt4-genesis-beacon.ssz").read_bytes()
    genesis = fulu.BeaconState.deserialize(
        io.BytesIO(genesis_bytes), len(genesis_bytes)
    )
    trusted = "0x0888cde6ee0c51578046793b377d6f6474908f7812b8db4d87a736d73e539d2c"
    assert h(genesis.validators) == trusted
    kzg.load_trusted_setup()
    expected_root = "0x8f3f70cdaa709e79a5a00df948d1fb4a42f36b09c0f8f379aeb7fa5f15448921"
    report["beacon_root"] = expected_root
    report["execution_hash"] = data["execution_header"]["hash"]
    # Use an independently verified supermajority finality update and complete parent hash links.
    progress = json.loads(
        (
            root / "attempt4-reference-light-client-proof-recovery-verified.json"
        ).read_text()
    )
    assert progress["passed"] is True
    assert len(progress["cryptographic_rejection_controls"]) == 6
    final_root = progress["reference_progress"]["finalized_root"]
    headers = {}
    for line in (
        root / "attempt4-reference-light-client-advertised-baseline.jsonl"
    ).open():
        rec = json.loads(line)
        path = urlsplit(rec["url"]).path
        if "/beacon/headers/" in path and rec["status"] == 200:
            raw = json.loads(rec["response_bytes"])["data"]["header"]["message"]
            header = parsed(raw, spec.BeaconBlockHeader)
            headers[h(header)] = header
    # The starting verified header also occurs as the first child's parent capture in this trace.
    if final_root not in headers:
        for line in (
            root / "attempt4-reference-light-client-advertised-baseline.jsonl"
        ).open():
            rec = json.loads(line)
            if rec["status"] == 200 and "/light_client/finality_update" in rec["url"]:
                raw = json.loads(rec["response_bytes"])["data"]["finalized_header"][
                    "beacon"
                ]
                header = parsed(raw, spec.BeaconBlockHeader)
                if h(header) == final_root:
                    headers[final_root] = header
    cursor = final_root
    links = 0
    while cursor != expected_root:
        assert cursor in headers, "authenticated parent header unavailable " + cursor
        cursor = hx(headers[cursor].parent_root)
        links += 1
        assert links < 4096
    report["authenticated_finalized_ancestry"] = {
        "finalized_root": final_root,
        "parent_hash_links": links,
        "source": "unchanged reference light-client finality verification and original complete header captures",
    }
    for node, raw in data["beacon_nodes"].items():
        assert raw["beacon_block_status"] == 200
        block = parsed(raw["beacon_block"]["data"], spec.SignedBeaconBlock)
        assert h(block.message) == expected_root
        proposer = int(block.message.proposer_index)
        pubkey = genesis.validators[proposer].pubkey
        domain = spec.compute_domain(
            spec.DOMAIN_BEACON_PROPOSER,
            spec.Version(b("0x80000038")),
            spec.Root(b(trusted)),
        )
        assert bls.Verify(
            pubkey, spec.compute_signing_root(block.message, domain), block.signature
        )
        assert raw["envelope"]["http_status"] == 200
        envelope = parsed(
            raw["envelope"]["body"]["data"], spec.SignedExecutionPayloadEnvelope
        )
        assert int(envelope.message.builder_index) == 2**64 - 1
        assert (
            int(envelope.message.payload.slot_number) == int(block.message.slot) == 3176
        )
        assert hx(envelope.message.beacon_block_root) == expected_root
        assert envelope.message.parent_beacon_block_root == block.message.parent_root
        assert (
            hx(envelope.message.payload.block_hash) == data["execution_header"]["hash"]
        )
        assert (
            envelope.message.payload.block_hash
            == block.message.body.signed_execution_payload_bid.message.block_hash
        )
        builder_domain = spec.compute_domain(
            spec.DOMAIN_BEACON_BUILDER,
            spec.Version(b("0x80000038")),
            spec.Root(b(trusted)),
        )
        assert bls.Verify(
            pubkey,
            spec.compute_signing_root(envelope.message, builder_domain),
            envelope.signature,
        )
        assert raw["columns"]["http_status"] == 200
        columns = raw["columns"]["body"]["data"]
        assert len(columns) == 128
        assert sorted(int(x["index"]) for x in columns) == list(range(128))
        commitments = list(
            block.message.body.signed_execution_payload_bid.message.blob_kzg_commitments
        )
        assert len(commitments) == 1
        cs = []
        indices = []
        cells = []
        proofs = []
        for col in columns:
            assert (
                int(col["slot"]) == 3176 and col["beacon_block_root"] == expected_root
            )
            assert len(col["column"]) == len(col["kzg_proofs"]) == len(commitments)
            for commitment, cell, proof in zip(
                commitments, col["column"], col["kzg_proofs"]
            ):
                cs.append(bytes(commitment))
                indices.append(int(col["index"]))
                cells.append(b(cell))
                proofs.append(b(proof))
        assert kzg.verify_cell_kzg_proof_batch(cs, indices, cells, proofs)
        raw_tx = next(
            bytes(x)
            for x in envelope.message.payload.transactions
            if "0x" + keccak(bytes(x)).hex() == data["blob_transaction"]
        )
        assert raw_tx[0] == 3
        tx_fields = rlp.decode(raw_tx[1:], strict=True)
        assert len(tx_fields) == 14
        assert [
            bytes(spec.kzg_commitment_to_versioned_hash(x)) for x in commitments
        ] == tx_fields[10]
        # Mutants are structurally valid byte arrays, each tested against the real verifier.
        wrong_cells = list(cells)
        changed = bytearray(wrong_cells[0])
        changed[31] ^= 1
        wrong_cells[0] = bytes(changed)
        wrong_indices = list(indices)
        wrong_indices[0] = (wrong_indices[0] + 1) % 128
        wrong_proofs = list(proofs)
        wrong_proofs[0] = wrong_proofs[1]
        controls = []
        for name, args in [
            ("cell", (cs, indices, wrong_cells, proofs)),
            ("index", (cs, wrong_indices, cells, proofs)),
            ("proof", (cs, indices, cells, wrong_proofs)),
        ]:
            assert rejected(lambda: kzg.verify_cell_kzg_proof_batch(*args)), name
            controls.append(name)
        wrong = envelope.copy()
        wrong.message.parent_beacon_block_root = spec.Root(bytes([1]) * 32)
        assert not bls.Verify(
            pubkey,
            spec.compute_signing_root(wrong.message, builder_domain),
            wrong.signature,
        )
        wrong_domain = spec.compute_domain(
            spec.DOMAIN_BEACON_BUILDER, spec.Version(bytes(4)), spec.Root(b(trusted))
        )
        assert not bls.Verify(
            pubkey,
            spec.compute_signing_root(envelope.message, wrong_domain),
            envelope.signature,
        )
        controls.extend(["envelope_message", "envelope_domain"])
        report["nodes"].append(
            {
                "container": node,
                "slot": 3176,
                "proposer": proposer,
                "columns": 128,
                "cell_proofs": len(cells),
                "block_signature": True,
                "envelope_signature": True,
                "versioned_hash_bound_to_el_transaction": True,
                "rejected_mutations": controls,
            }
        )
        save()
    report["passed"] = True
    report["scope_limit"] = (
        "One real blob, all columns; partial row/bitmap/loss/reconnect campaigns remain separate."
    )
    save()
    print(json.dumps(report))
except Exception as error:
    report["failure"] = {"class": type(error).__name__, "message": str(error)}
    save()
    raise
