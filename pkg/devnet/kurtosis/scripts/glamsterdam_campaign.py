#!/usr/bin/env python3
"""Capture public RPC/Beacon inputs and running executables, never credentials."""
import argparse
import json
import subprocess
import time
import urllib.error
import urllib.request
from datetime import datetime, timezone
from pathlib import Path

ENCLAVE = "99429150b4714b2ea7746343b97175e7"
GENESIS = "0xfdf84ec05edbc77e24c84bc6882b6f0b634c070b03ea52b284695e04002c6c24"
OPENER = urllib.request.build_opener(urllib.request.ProxyHandler({}))


def now():
    return datetime.now(timezone.utc).isoformat()


def docker(*args):
    return subprocess.check_output(["docker", *args], text=True, timeout=60)


def inventory():
    ids = docker("ps", "-aq", "--filter", "label=kurtosis_enclave_uuid=" + ENCLAVE).split()
    if not ids:
        raise RuntimeError("the declared enclave has no containers")
    nodes = {}
    for item in json.loads(docker("inspect", *ids)):
        labels = item["Config"].get("Labels", {})
        assert labels.get("kurtosis_enclave_uuid") == ENCLAVE
        service = labels.get("kurtosis_service_name", "")
        if not service.startswith(("el-", "cl-", "vc-")) and service != "buildoor":
            continue
        nodes[service] = {
            "container": item["Id"], "image_name": item["Config"]["Image"],
            "image_id": item["Image"], "command": item["Config"]["Cmd"],
            "entrypoint": item["Config"]["Entrypoint"],
            "memory_limit": item["HostConfig"]["Memory"],
            "nano_cpus": item["HostConfig"]["NanoCpus"],
            "state": item["State"], "ports": item["NetworkSettings"]["Ports"],
        }
    return nodes


def endpoint(node, port):
    return "http://127.0.0.1:" + node["ports"][port][0]["HostPort"]


def exchange(url, method=None, params=None):
    request = None if method is None else {"jsonrpc": "2.0", "id": 1, "method": method, "params": params}
    data = None if request is None else json.dumps(request).encode()
    req = urllib.request.Request(url, data, {} if data is None else {"Content-Type": "application/json"})
    record = {"started": now(), "url": url, "request": request}
    started = time.monotonic()
    try:
        try:
            response = OPENER.open(req, timeout=10)
        except urllib.error.HTTPError as error:
            response = error
        with response:
            record.update(http_status=response.code, response=response.read().decode())
    except (OSError, TimeoutError) as error:
        record["transport_error"] = {"class": type(error).__name__, "message": str(error)}
    record["elapsed_ms"] = (time.monotonic() - started) * 1000
    return record


def decoded(record):
    if record.get("http_status") != 200:
        raise RuntimeError("unsuccessful captured exchange: " + json.dumps(record))
    value = json.loads(record["response"])
    if "error" in value:
        raise RuntimeError("RPC error: " + json.dumps(value["error"]))
    if record["request"] is not None and (value.get("id") != 1 or value.get("jsonrpc") != "2.0"):
        raise RuntimeError("RPC response identity mismatch")
    return value


def assert_private_network(nodes):
    for service, node in nodes.items():
        if service.startswith("el-"):
            url = endpoint(node, "8545/tcp")
            chain = decoded(exchange(url, "eth_chainId", []))["result"]
            genesis = decoded(exchange(url, "eth_getBlockByNumber", ["0x0", False]))["result"]
            if int(chain, 16) != 1337 or genesis["hash"] != GENESIS:
                raise RuntimeError("not the declared private chain: " + service)


def executable_hashes(node):
    # Hash the running inode, including replaced files, instead of the image path.
    script = '''for f in /proc/[0-9]*/exe; do
p=$(readlink "$f" 2>/dev/null) || continue
case "$p" in
 /usr/local/bin/eth2030*|/usr/local/bin/geth-real*|/usr/local/bin/lighthouse*|/usr/local/bin/engine-capture*)
  printf '%s %s ' "$f" "$p"; sha256sum "$f" || exit 1;;
esac
done'''
    return docker("exec", "--user", "root", node["container"], "sh", "-c", script)


def snapshot(output):
    output.mkdir(parents=True, exist_ok=False)
    nodes = inventory()
    records = []
    for service, node in sorted(nodes.items()):
        if not service.startswith(("el-", "cl-")):
            continue
        node["running_executable_hashes"] = executable_hashes(node)
        if service.startswith("el-"):
            url = endpoint(node, "8545/tcp")
            for method, params in [("web3_clientVersion", []), ("eth_chainId", []),
                                   ("eth_getBlockByNumber", ["0x0", False]),
                                   ("eth_getBlockByNumber", ["finalized", False]),
                                   ("eth_getBlockByNumber", ["latest", False]), ("eth_syncing", [])]:
                records.append({"service": service, **exchange(url, method, params)})
        else:
            url = endpoint(node, "4000/tcp")
            for path in ["/eth/v1/node/version", "/eth/v1/node/syncing", "/eth/v1/node/identity",
                         "/eth/v1/node/peers", "/eth/v1/config/spec", "/eth/v1/beacon/genesis",
                         "/eth/v1/beacon/states/head/finality_checkpoints"]:
                records.append({"service": service, **exchange(url + path)})
    (output / "inventory.json").write_text(json.dumps({"at": now(), "enclave": ENCLAVE, "services": nodes}, indent=2) + "\n")
    with (output / "exchanges.jsonl").open("x") as stream:
        for record in records:
            stream.write(json.dumps(record) + "\n")
    # Retain failures before validation; HTTP success is not tuple qualification.
    assert_private_network(nodes)
    for record in records:
        decoded(record)
    print(json.dumps({"snapshot": str(output), "services": len(nodes), "exchanges": len(records)}))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()
    snapshot(args.output)


if __name__ == "__main__":
    main()
