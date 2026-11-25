#!/usr/bin/env python3
import sys, json, hashlib, time, io, re, subprocess, argparse
from pathlib import Path
from Crypto.Protocol.KDF import HKDF
from Crypto.Hash import SHA256

curve_order = int(
    "73eda753299d7d483339d80809a1d80553bda402fffe5bfeffffffff00000001", 16
)
from eth_consensus_specs.gloas import mainnet as spec
from eth_consensus_specs.utils import bls


def child_bytes(entry, index):
    salt = index.to_bytes(4, "big")
    parts = []
    for ikm in [entry, bytes(x ^ 255 for x in entry)]:
        lamport = HKDF(ikm, 255 * 32, salt, SHA256, context=b"")
        parts.extend(
            hashlib.sha256(lamport[n * 32 : (n + 1) * 32]).digest() for n in range(255)
        )
    ikm = hashlib.sha256(b"".join(parts)).digest()
    salt = b"BLS-SIG-KEYGEN-SALT-"
    scalar = 0
    while scalar == 0:
        salt = hashlib.sha256(salt).digest()
        scalar = (
            int.from_bytes(
                HKDF(ikm + b"\x00", 48, salt, SHA256, context=b"\x00\x30"), "big"
            )
            % curve_order
        )
    return scalar.to_bytes(32, "big")


def old_key(builder_container, key_index, public_key):
    v = json.loads(
        subprocess.check_output(["docker", "inspect", builder_container], text=True)
    )[0]
    command = " ".join(v["Config"]["Cmd"])
    match = re.search(
        r"--builder-privkey(?:=|\s+)(?:0x)?([0-9a-fA-F]{64})(?![0-9a-fA-F])", command
    )
    assert match is not None
    # Frozen producer source and public-key checks bind this exact EIP-2333
    # child to the actual retired key. No secret is written to an artifact.
    sk = int.from_bytes(child_bytes(bytes.fromhex(match.group(1)), key_index), "big")
    expected = bytes.fromhex(public_key.removeprefix("0x"))
    assert bytes(bls.SkToPk(sk)) == expected
    return sk, expected


parser = argparse.ArgumentParser(
    description="Sign one unchanged actual envelope with its retired builder key for a real-client rejection control. Private material stays in memory."
)
parser.add_argument("--builder-container", required=True)
parser.add_argument("--capture", type=Path)
parser.add_argument("--retired-key-index", type=int, default=1)
parser.add_argument(
    "--retired-pubkey",
    default="0xb49b274d74d615523a5591a22a37b4e9cdfb1fbaab95df7e8aae393d5823493011965f67213574ffcd7138c9689aa96e",
)
args = parser.parse_args()
# Independent published EIP-2333 child derivation vectors, before private inputs.
for master, index, child in [
    (
        6083874454709270928345386274498605044986640685124978867557563392430687146096,
        0,
        20397789859736650942317412262472558107875392172444076792671091975210932703118,
    ),
    (
        29757020647961307431480504535336562678282505419141012933316116377660817309383,
        3141592653,
        25457201688850691947727629385191704516744796114925897962676248250929345014287,
    ),
    (
        27580842291869792442942448775674722299803720648445448686099262467207037398656,
        4294967295,
        29358610794459428860402234341874281240803786294062035874021252734817515685787,
    ),
    (
        19022158461524446591288038168518313374041767046816487870552872741050760015818,
        42,
        31372231650479070279774297061823572166496564838472787488249775572789064611981,
    ),
]:
    assert (
        int.from_bytes(child_bytes(master.to_bytes(32, "big"), index), "big") == child
    )
sk, expected = old_key(
    args.builder_container, args.retired_key_index, args.retired_pubkey
)
if args.capture is None:
    print(
        json.dumps(
            {
                "eip2333_vectors": 4,
                "derived_public_key_matches_actual_retired_builder": True,
            }
        )
    )
    raise SystemExit(0)
r = args.capture
deadline = time.monotonic() + 600
while not (r / "probe-input.ssz").exists():
    if time.monotonic() > deadline:
        raise RuntimeError("original producer probe did not arrive")
    time.sleep(0.005)
raw = (r / "probe-input.ssz").read_bytes()
start = int.from_bytes(raw[:4], "little")
stop = int.from_bytes(raw[4:8], "little")
wire = raw[start:stop]
env = spec.SignedExecutionPayloadEnvelope.deserialize(io.BytesIO(wire), len(wire))
domain = spec.compute_domain(
    spec.DOMAIN_BEACON_BUILDER,
    spec.Version(bytes.fromhex("80000038")),
    spec.Root(
        bytes.fromhex(
            "0888cde6ee0c51578046793b377d6f6474908f7812b8db4d87a736d73e539d2c"
        )
    ),
)
signing = spec.compute_signing_root(env.message, domain)
signature = bytes(bls.Sign(sk, signing))
assert bls.Verify(expected, signing, signature)
state = json.loads((r / "reused-builder.json").read_text())
new_pub = bytes.fromhex(state["pubkey"][2:])
assert (
    new_pub != expected
    and bls.Verify(new_pub, signing, env.signature)
    and not bls.Verify(new_pub, signing, signature)
)
temp = r / "probe-signature.tmp"
temp.write_bytes(signature)
temp.replace(r / "probe-signature.bin")
(r / "independent-signatures.json").write_text(
    json.dumps(
        {
            "retired_pubkey": "0x" + expected.hex(),
            "current_pubkey": "0x" + new_pub.hex(),
            "builder_index": int(env.message.builder_index),
            "old_signature_valid_under_retired_key": True,
            "old_signature_rejected_under_current_key": True,
            "original_signature_valid_under_current_key": True,
            "signing_root": "0x" + bytes(signing).hex(),
        },
        indent=2,
    )
    + "\n"
)
print("original and retired signatures independently verified", flush=True)
