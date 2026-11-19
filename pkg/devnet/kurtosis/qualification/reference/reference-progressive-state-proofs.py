import json, io, gzip, hashlib
from pathlib import Path
from eth_consensus_specs.gloas import mainnet as spec
from ssz.proofs import build_proof
import argparse

parser = argparse.ArgumentParser(
    description="Verify frozen public devnet inputs with the unchanged selected consensus reference."
)
parser.add_argument("--artifact-dir", required=True, type=Path)
root = parser.parse_args().artifact_dir
assert not (root / "attempt4-reference-progressive-state-proofs.json").exists()
raw = json.loads((root / "attempt4-progressive-state-consumers.json").read_text())
header = raw["selected_beacon_header"]
expected = header["header"]["message"]["state_root"]
result = {
    "spec_revision": "2ea9d96a420927eb1a35d7c9399c8b3691c69f6a",
    "slot": int(header["header"]["message"]["slot"]),
    "state_root": expected,
    "beacon_root": header["root"],
    "nodes": [],
    "passed": False,
}


def hx(x):
    return "0x" + bytes(x).hex()


def save():
    (root / "attempt4-reference-progressive-state-proofs.json").write_text(
        json.dumps(result, indent=2) + "\n"
    )


for cid, node in raw["nodes"].items():
    assert node["status"] == 200
    data = (root / node["ssz_file"]).read_bytes()
    state = spec.BeaconState.deserialize(io.BytesIO(data), len(data))
    assert hx(spec.hash_tree_root(state)) == expected
    assert len(state.validators) == 384
    entry = {
        "container": cid,
        "ssz_sha256": hashlib.sha256(data).hexdigest(),
        "validators": len(state.validators),
        "builders": len(state.builders),
        "proofs": {},
    }
    cases = [
        (
            "current_sync_committee",
            2945,
            spec.hash_tree_root(state.current_sync_committee),
        ),
        ("next_sync_committee", 2946, spec.hash_tree_root(state.next_sync_committee)),
        ("finalized_root", 735, state.finalized_checkpoint.root),
    ]
    for name, index, leaf in cases:
        branch = build_proof(state, index)
        depth = index.bit_length() - 1
        assert spec.is_valid_merkle_branch(
            leaf,
            branch,
            depth,
            index - (1 << depth),
            spec.Root(bytes.fromhex(expected[2:])),
        )
        wrong = list(branch)
        wrong[0] = spec.Root(bytes([1]) * 32)
        assert not spec.is_valid_merkle_branch(
            leaf,
            wrong,
            depth,
            index - (1 << depth),
            spec.Root(bytes.fromhex(expected[2:])),
        )
        entry["proofs"][name] = {
            "gindex": index,
            "leaf": hx(leaf),
            "branch": [hx(x) for x in branch],
            "mutated_sibling_rejected": True,
        }
    result["nodes"].append(entry)
    save()
assert len({entry["ssz_sha256"] for entry in result["nodes"]}) == 1
fixture = (root / raw["nodes"]["87799773fc57"]["ssz_file"]).read_bytes()
(root / "actual-gloas-384-state.ssz.gz").write_bytes(gzip.compress(fixture, mtime=0))
result["passed"] = True
result["scope"] = (
    "Actual common state from three mainnet-preset CLs, 384 validators and the live builder registry; reference-generated current/next/finalized branches. Bootstrap/finality authentication is separately retained."
)
save()
print(
    json.dumps(
        {
            "passed": True,
            "slot": result["slot"],
            "state_root": expected,
            "compressed_bytes": (root / "actual-gloas-384-state.ssz.gz").stat().st_size,
            "nodes": len(result["nodes"]),
        }
    )
)
