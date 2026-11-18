#!/usr/bin/env python3
"""Rebuild the fixed ETH2030 v1 runtime with the recorded compiler/settings."""
import json
import argparse
from pathlib import Path
import subprocess

HERE = Path(__file__).resolve().parent
VERSION = "0.8.38+commit.1835d247.Linux.g++"
parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("--check", action="store_true", help="Compare artifacts without overwriting them")
options = parser.parse_args()

def emit(name, value):
    path = HERE / name
    if options.check:
        if not path.exists() or path.read_text() != value:
            raise RuntimeError("artifact differs: " + name)
    else:
        path.write_text(value)
version = subprocess.check_output(["solc", "--version"], text=True)
if "Version: " + VERSION not in version:
    raise RuntimeError("paymaster v1 requires solc " + VERSION)
request = {"language": "Yul", "sources": {"Paymaster.yul": {"content": (HERE / "Paymaster.yul").read_text()}},
           "settings": {"optimizer": {"enabled": True, "runs": 200}, "evmVersion": "osaka",
                        "outputSelection": {"*": {"*": ["evm.bytecode.object", "evm.deployedBytecode.object"]}}}}
result = subprocess.run(["solc", "--standard-json"], input=json.dumps(request),
                        capture_output=True, text=True, check=True)
output = json.loads(result.stdout)
errors = [item["formattedMessage"] for item in output.get("errors", []) if item["severity"] == "error"]
if errors:
    raise RuntimeError("\n".join(errors))
evm = output["contracts"]["Paymaster.yul"]["ETH2030PaymasterV1"]["evm"]
emit("init.hex", evm["bytecode"]["object"] + "\n")
emit("runtime.hex", evm["deployedBytecode"]["object"] + "\n")
emit("build.json", json.dumps({"version": "eth2030-frame-paymaster-v1", "compiler": VERSION,
    "eipRevision": "dc5d5fa688b7fe87b95a907fcc128b4fde2f4d2f", "settings": request["settings"],
    "witness": "protocol-validated SECP256K1 signature at index 1, empty msg, resolved signer equal to owner"}, indent=2) + "\n")
print("runtime bytes:", len(evm["deployedBytecode"]["object"]) // 2)
