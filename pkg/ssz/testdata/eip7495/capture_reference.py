"""Cross-check the unchanged consensus reference against captured Lighthouse
Gloas state and proofs. Run in the generated d202 consensus-reference venv with
PYTHONPATH pointing at the pinned remerkleable checkout. Writes gzip to stdout.
"""
import gzip
import hashlib
import inspect
import io
import json
from pathlib import Path
import subprocess
import sys
from typing import ClassVar

from eth_consensus_specs.gloas import mainnet as spec
from ssz.container import ProgressiveContainer
from ssz.proofs import build_proof
from ssz.uint import Uint16
from remerkleable.progressive import to_gindex_progressive

SPEC = "d2021d2719235f0df4726a1d330502cb15aa8ea3"
source = next(p for p in Path(inspect.getfile(spec)).parents if (p / ".git").exists())
assert subprocess.check_output(["git", "-C", str(source), "rev-parse", "HEAD"], text=True).strip() == SPEC
assert not subprocess.check_output(["git", "-C", str(source), "diff", "--", "specs", "pysetup", "tests/core/pyspec/eth_consensus_specs/utils"], text=True)
fixture_root = Path(sys.argv[1])
raw = gzip.decompress((fixture_root / "gloas-384-state.ssz.gz").read_bytes())
expected = json.loads((fixture_root / "gloas-384-proofs.json").read_text())
state = spec.BeaconState.deserialize(io.BytesIO(raw), len(raw))
assert state.encode_bytes() == raw
root = spec.hash_tree_root(state)
assert root.hex() == expected["state_root"].removeprefix("0x")
assert len(state.validators) == expected["validators"] == 384
names = list(type(state).model_fields)
fields = [getattr(state, name) for name in names]
positions = [i for i, bit in enumerate(state.ACTIVE_FIELDS) if bit]
proofs = []
for position in positions:
    gindex = int(to_gindex_progressive(position)[0])
    proofs.append({"position": position, "gindex": gindex, "branch": [bytes(x).hex() for x in build_proof(state, gindex)]})
captured = {"name": "CapturedGloas384", "active_fields": list(map(bool, state.ACTIVE_FIELDS)),
            "sizes": [type(value).fixed_size() if type(value).is_fixed_size() else 0 for value in fields],
            "fields": [value.encode_bytes().hex() for value in fields],
            "field_roots": [bytes(spec.hash_tree_root(value)).hex() for value in fields],
            "root": bytes(root).hex(), "proofs": proofs}
# Normative width-256 shape without the extra 'more' naming restriction of the
# other reference. Do not patch either reference or waive the 256-bit case.
annotations = {"ACTIVE_FIELDS": ClassVar[tuple[int, ...]], **{f"f{i}": Uint16 for i in range(256)}}
T = type("Normative256WithoutMore", (ProgressiveContainer,), {"ACTIVE_FIELDS": (1,) * 256, "__annotations__": annotations})
value = T(**{f"f{i}": i + 1 for i in range(256)})
result = {"reference": "ethereum/consensus-specs@" + SPEC,
          "source_fixture_sha256": hashlib.sha256(raw).hexdigest(),
          "capture": captured, "original_nested_proofs": {name: {"gindex": proof["gindex"], "leaf": proof["leaf"].removeprefix("0x"), "branch": [x.removeprefix("0x") for x in proof["branch"]]} for name, proof in expected["proofs"].items()},
          "secondary_256_without_more": {"root": bytes(spec.hash_tree_root(value)).hex(), "ssz": value.encode_bytes().hex()}}
sys.stdout.buffer.write(gzip.compress(json.dumps(result, separators=(",", ":")).encode(), mtime=0))
