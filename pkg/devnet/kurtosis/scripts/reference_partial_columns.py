#!/usr/bin/env python3
import sys, json, io, copy, argparse
from pathlib import Path
from types import SimpleNamespace
import rlp
from eth_hash.auto import keccak
from eth_consensus_specs.gloas import mainnet as spec
from eth_consensus_specs.fulu import mainnet as fulu
from eth_consensus_specs.utils import bls, kzg

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "qualification/reference"))
from ssz_rest import parsed

parser = argparse.ArgumentParser(
    description="Independently replay signed partial-column captures using the pinned unchanged consensus reference."
)
parser.add_argument("--anchor", type=Path, required=True)
parser.add_argument("--bundle", type=Path, required=True)
parser.add_argument("--genesis", type=Path, required=True)
parser.add_argument("--output", type=Path, required=True)
args = parser.parse_args()
anchor = json.loads(args.anchor.read_text())
out = args.output
assert not out.exists()
raw = args.genesis.read_bytes()
genesis = fulu.BeaconState.deserialize(io.BytesIO(raw), len(raw))
gvr = spec.hash_tree_root(genesis.validators)
assert (
    bls.bls_active
    and bytes(gvr).hex()
    == "0888cde6ee0c51578046793b377d6f6474908f7812b8db4d87a736d73e539d2c"
)
kzg.load_trusted_setup()


def b(x):
    return bytes.fromhex(x[2:])


def h(x):
    return "0x" + bytes(spec.hash_tree_root(x)).hex()


report = {
    "passed": False,
    "spec": "2ea9d96a420927eb1a35d7c9399c8b3691c69f6a",
    "scope": "Unchanged BLS/KZG and exact signed block/envelope/transaction commitments. Historical BeaconState retrieval is separately retained as 404; no unavailable state/root is invented.",
    "nodes": [],
}

bundle = json.loads(args.bundle.read_text())


def verify_node(name, node):
    block = parsed(node["block"]["data"], spec.SignedBeaconBlock)
    assert (
        h(block.message) == anchor["beacon_root"]
        and int(block.message.slot) == anchor["slot"]
    )
    pub = genesis.validators[int(block.message.proposer_index)].pubkey
    proposer_domain = spec.compute_domain(
        spec.DOMAIN_BEACON_PROPOSER, spec.Version(b("0x80000038")), gvr
    )
    assert bls.Verify(
        pub, spec.compute_signing_root(block.message, proposer_domain), block.signature
    )
    env = parsed(node["envelope"]["data"], spec.SignedExecutionPayloadEnvelope)
    assert int(env.message.builder_index) == 2**64 - 1
    builder_domain = spec.compute_domain(
        spec.DOMAIN_BEACON_BUILDER, spec.Version(b("0x80000038")), gvr
    )
    assert bls.Verify(
        pub, spec.compute_signing_root(env.message, builder_domain), env.signature
    )
    bid = block.message.body.signed_execution_payload_bid.message
    assert (
        env.message.beacon_block_root == spec.hash_tree_root(block.message)
        and env.message.parent_beacon_block_root == block.message.parent_root
        and env.message.payload.block_hash == bid.block_hash
        and "0x" + bytes(bid.block_hash).hex() == anchor["completion"]["last_block"]
    )
    commitments = list(bid.blob_kzg_commitments)
    assert len(commitments) == 2 and commitments[0] != commitments[1]
    tx = next(
        bytes(t)
        for t in env.message.payload.transactions
        if "0x" + keccak(bytes(t)).hex() == anchor["completion"]["blob_transaction"]
    )
    fields = rlp.decode(tx[1:], strict=True)
    assert (
        tx[0] == 3
        and len(fields) == 14
        and fields[10]
        == [bytes(spec.kzg_commitment_to_versioned_hash(c)) for c in commitments]
    )
    columns = node["columns"]["data"]
    column_indices = [int(c["index"]) for c in columns]
    assert (
        columns
        and len(column_indices) == len(set(column_indices))
        and all(0 <= i < 128 for i in column_indices)
    )
    cs = []
    indices = []
    cells = []
    proofs = []
    for col in columns:
        assert (
            int(col["slot"]) == anchor["slot"]
            and col["beacon_block_root"] == anchor["beacon_root"]
            and len(col["column"]) == len(col["kzg_proofs"]) == 2
        )
        for c, cell, proof in zip(commitments, col["column"], col["kzg_proofs"]):
            cs.append(bytes(c))
            indices.append(int(col["index"]))
            cells.append(b(cell))
            proofs.append(b(proof))
    assert kzg.verify_cell_kzg_proof_batch(cs, indices, cells, proofs)
    changed = list(cells)
    changed[0] = bytes([changed[0][0] ^ 1]) + changed[0][1:]
    assert not kzg.verify_cell_kzg_proof_batch(cs, indices, changed, proofs)
    changed = list(proofs)
    changed[0] = bytes([changed[0][0] ^ 1]) + changed[0][1:]
    try:
        bad = kzg.verify_cell_kzg_proof_batch(cs, indices, cells, changed)
    except (AssertionError, ValueError, RuntimeError):
        bad = False
    assert not bad
    signature = bytearray(bytes(env.signature))
    signature[0] ^= 1
    assert not bls.Verify(
        pub, spec.compute_signing_root(env.message, builder_domain), bytes(signature)
    )
    return {
        "node": name,
        "proofs_verified": len(proofs),
        "column_indices": column_indices,
        "signed_block_and_envelope": True,
        "two_ordered_rows": True,
        "rejections": ["changed_cell", "changed_proof", "changed_envelope_signature"],
    }


results = [verify_node(name, node) for name, node in bundle["nodes"].items()]
assert len(results) == 3
report.update(
    passed=True,
    nodes=results,
    total_cell_proofs=sum(row["proofs_verified"] for row in results),
    beacon_root=anchor["beacon_root"],
    slot=anchor["slot"],
)
out.write_text(json.dumps(report, indent=2) + "\n")
print(json.dumps(report))
