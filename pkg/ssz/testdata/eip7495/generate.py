"""Generate EIP-7495/7916 bytes, roots and complete branches with an unchanged
ethereum/remerkleable checkout at 2f0baeef0082d4278acaef7d822deb7009d7db7e.

Usage: PYTHONPATH=/path/to/remerkleable python generate.py > vectors.json.gz
No Go implementation is imported. The checked-in corpus is deterministic.
"""
import gzip
import hashlib
import json
from pathlib import Path
import subprocess
import sys

import remerkleable
from remerkleable.basic import uint8, uint16, uint64
from remerkleable.byte_arrays import ByteList, Bytes32
from remerkleable.progressive import ProgressiveContainer, ProgressiveList, ProgressiveByteList

REVISION = "2f0baeef0082d4278acaef7d822deb7009d7db7e"
source = Path(remerkleable.__file__).resolve().parents[1]
assert subprocess.check_output(["git", "-C", str(source), "rev-parse", "HEAD"], text=True).strip() == REVISION
assert not subprocess.check_output(["git", "-C", str(source), "diff", "--", "remerkleable"], text=True)


def branch(view, index):
    node = view.get_backing()
    siblings = []
    for bit in bin(int(index))[3:]:
        siblings.append((node.get_right() if bit == "0" else node.get_left()).merkle_root())
        node = node.get_left() if bit == "0" else node.get_right()
    siblings.reverse()
    value, g = node.merkle_root(), int(index)
    for sibling in siblings:
        value = hashlib.sha256(value + sibling if g & 1 == 0 else sibling + value).digest()
        g >>= 1
    assert g == 1 and value == view.hash_tree_root()
    return [root.hex() for root in siblings]


def container(name, layout, types, values):
    names = [f"f{i}" for i in range(len(types))]
    # This reference adds a non-normative field-name requirement at width 256.
    # Names do not affect SSZ. Use its accepted spelling without patching code;
    # Go independently tests width 256 with no name requirement.
    if len(layout) == 256:
        names[-1] = "more"
    cls = type(name, (ProgressiveContainer(active_fields=layout),), {"__annotations__": dict(zip(names, types))})
    view = cls(**dict(zip(names, values)))
    fields = [getattr(view, field) for field in names]
    positions = [i for i, bit in enumerate(layout) if bit]
    return {
        "name": name, "active_fields": list(map(bool, layout)),
        "sizes": [typ.type_byte_length() if typ.is_fixed_byte_length() else 0 for typ in types],
        "fields": [field.encode_bytes().hex() for field in fields],
        "field_roots": [field.hash_tree_root().hex() for field in fields],
        "ssz": view.encode_bytes().hex(), "root": view.hash_tree_root().hex(),
        "proofs": [{"position": position, "gindex": int(cls.key_to_static_gindex(field)),
                    "branch": branch(view, cls.key_to_static_gindex(field))}
                   for position, field in zip(positions, names)],
    }


containers = [
    container("Square", [1, 0, 1], [uint16, uint8], [42, 255]),
    container("Circle", [0, 1, 1], [uint16, uint8], [7, 255]),
    container("SquareZero", [1, 0, 1], [uint16, uint8], [0, 0]),
    container("CircleZero", [0, 1, 1], [uint16, uint8], [0, 0]),
    container("MixedVariable", [1, 0, 1, 1], [uint16, ByteList[40], ProgressiveByteList], [42, b"abc", b"\x00\xff"]),
    container("EmptyVariable", [0, 1, 1, 1], [ByteList[40], uint64, ProgressiveByteList], [b"", 0, b""]),
]
class Inner(ProgressiveContainer(active_fields=[1, 0, 1])):
    number: uint16
    data: ProgressiveByteList
containers.append(container("Nested", [0, 1, 0, 1], [Inner, ByteList[32]], [Inner(number=65535, data=b"nested"), b"end"]))
for width in [1, 2, 3, 5, 6, 21, 22, 85, 86, 255, 256]:
    for sparse in [False, True]:
        layout = [1 if not sparse or i % 3 == 0 or i == width - 1 else 0 for i in range(width)]
        count = sum(layout)
        containers.append(container(f"Width{width}{'Sparse' if sparse else 'Dense'}", layout, [uint16] * count, list(range(1, count + 1))))

lists = []
for count in [1, 2, 3, 5, 6, 21, 22, 85, 86, 256]:
    view = ProgressiveList[Bytes32]([Bytes32((i + 1).to_bytes(32, "little")) for i in range(count)])
    lists.append({"count": count, "field_roots": [value.hash_tree_root().hex() for value in view],
                  "root": view.hash_tree_root().hex(),
                  "proofs": [{"position": i, "gindex": int(type(view).chunk_to_gindex(i)),
                              "branch": branch(view, type(view).chunk_to_gindex(i))} for i in range(count)]})

invalid = []
for name, layout, count in [("empty", [], 0), ("trailing", [1, 0], 1), ("mismatch", [1, 1], 1), ("too_wide", [1] * 257, 257)]:
    try:
        type("Invalid_" + name, (ProgressiveContainer(active_fields=layout),),
             {"__annotations__": {f"f{i}": uint16 for i in range(count)}})
    except Exception as error:
        invalid.append({"name": name, "active_fields": list(map(bool, layout)), "sizes": [2] * count, "reference_rejected": True, "reason": str(error)})
    else:
        raise AssertionError("reference accepted illegal type " + name)

output = {"reference": "ethereum/remerkleable@" + REVISION,
          "reference_unmodified": True,
          "reference_limitations": ["At width 256 remerkleable requires a field named more, beyond the normative EIP rule. Fixture field names use that accepted spelling; SSZ bytes/roots/indices are unaffected. Go accepts normative layouts without names."],
          "containers": containers, "lists": lists, "invalid_schemas": invalid}
sys.stdout.buffer.write(gzip.compress(json.dumps(output, separators=(",", ":")).encode(), mtime=0))
