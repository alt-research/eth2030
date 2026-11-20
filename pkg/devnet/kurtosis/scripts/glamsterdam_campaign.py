#!/usr/bin/env python3
"""Capture public RPC/Beacon inputs and running executables, never credentials."""

import argparse
import base64
import json
import math
import re
import subprocess
import time
import urllib.error
import urllib.request
from datetime import datetime, timezone
from pathlib import Path
from concurrent.futures import ThreadPoolExecutor

ENCLAVE = "99429150b4714b2ea7746343b97175e7"
GENESIS = "0xfdf84ec05edbc77e24c84bc6882b6f0b634c070b03ea52b284695e04002c6c24"
OPENER = urllib.request.build_opener(urllib.request.ProxyHandler({}))
SECRET_FLAGS = [
    "wallet-privkey",
    "builder-privkey",
    "builder-mnemonic",
    "txpool-auth-token",
    "rpc.auth_secret",
    "engine.auth_secret",
]


def public_command(arguments):
    if arguments is None:
        return None
    result, hide_next = [], False
    for argument in arguments:
        if hide_next:
            result.append("[redacted]")
            hide_next = False
            continue
        if argument in ["--" + name for name in SECRET_FLAGS]:
            result.append(argument)
            hide_next = True
            continue
        for name in SECRET_FLAGS:
            argument = re.sub(
                r"(--" + re.escape(name) + r"(?:=|\s+))(?:\"[^\"]*\"|'[^']*'|[^\s;]+)",
                r"\1[redacted]",
                argument,
            )
        result.append(argument)
    return result


def now():
    return datetime.now(timezone.utc).isoformat()


def docker(*args):
    return subprocess.check_output(["docker", *args], text=True, timeout=60)


def inventory():
    # Stopped historical attempts are not part of the running tuple. Inspecting
    # all of them can block live observations behind an unrelated old DB copy.
    # A service disappearing still changes stable_identity and fails the window.
    ids = docker(
        "ps", "-q", "--filter", "label=kurtosis_enclave_uuid=" + ENCLAVE
    ).split()
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
            "container": item["Id"],
            "image_name": item["Config"]["Image"],
            "image_id": item["Image"],
            "command": public_command(item["Config"]["Cmd"]),
            "entrypoint": item["Config"]["Entrypoint"],
            "memory_limit": item["HostConfig"]["Memory"],
            "nano_cpus": item["HostConfig"]["NanoCpus"],
            "state": item["State"],
            "ports": item["NetworkSettings"]["Ports"],
        }
    return nodes


def endpoint(node, port):
    return "http://127.0.0.1:" + node["ports"][port][0]["HostPort"]


def exchange(url, method=None, params=None):
    request = (
        None
        if method is None
        else {"jsonrpc": "2.0", "id": 1, "method": method, "params": params}
    )
    data = None if request is None else json.dumps(request).encode()
    req = urllib.request.Request(
        url, data, {} if data is None else {"Content-Type": "application/json"}
    )
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
        record["transport_error"] = {
            "class": type(error).__name__,
            "message": str(error),
        }
    record["elapsed_ms"] = (time.monotonic() - started) * 1000
    return record


def decoded(record):
    if record.get("http_status") != 200:
        raise RuntimeError("unsuccessful captured exchange: " + json.dumps(record))
    value = json.loads(record["response"])
    if "error" in value:
        raise RuntimeError("RPC error: " + json.dumps(value["error"]))
    if record["request"] is not None and (
        value.get("id") != 1 or value.get("jsonrpc") != "2.0"
    ):
        raise RuntimeError("RPC response identity mismatch")
    return value


def restore_builder_runtime(base, original, record):
    """Restore fleet size before enabling the producer's lifecycle manager."""

    def request(method, path, value=None):
        data = None if value is None else json.dumps(value).encode()
        req = urllib.request.Request(
            base + path,
            data,
            {} if data is None else {"Content-Type": "application/json"},
            method=method,
        )
        try:
            response = OPENER.open(req, timeout=10)
        except urllib.error.HTTPError as error:
            response = error
        with response:
            row = {
                "started": now(),
                "url": base + path,
                "method": method,
                "request": value,
                "http_status": response.status,
                "response": response.read().decode(),
            }
        record(row)
        if row["http_status"] != 200:
            raise RuntimeError("builder runtime restore failed: " + json.dumps(row))
        return json.loads(row["response"])

    current = request("GET", "/api/v1/config")
    target = original["builder_keys"]["target_count"]
    if current["builder_keys"]["target_count"] != target:
        request("POST", "/api/v1/builder-keys/target", {"target": target})
    if current["lifecycle_enabled"] != original["lifecycle_enabled"]:
        request(
            "PATCH",
            "/api/v1/config",
            {"lifecycle_enabled": original["lifecycle_enabled"]},
        )
    if request("GET", "/api/v1/config") != original:
        raise RuntimeError("builder restart changed other declared runtime settings")


def assert_private_network(nodes):
    for service, node in nodes.items():
        if service.startswith("el-"):
            url = endpoint(node, "8545/tcp")
            chain = decoded(exchange(url, "eth_chainId", []))["result"]
            genesis = decoded(exchange(url, "eth_getBlockByNumber", ["0x0", False]))[
                "result"
            ]
            if int(chain, 16) != 1337 or genesis["hash"] != GENESIS:
                raise RuntimeError("not the declared private chain: " + service)


def executable_hashes(node):
    # Hash the running inode, including replaced files, instead of the image path.
    script = """for f in /proc/[0-9]*/exe; do
p=$(readlink "$f" 2>/dev/null) || continue
case "$p" in
 /usr/local/bin/eth2030*|/usr/local/bin/geth-real*|/usr/local/bin/lighthouse*|/usr/local/bin/engine-capture*|/usr/local/bin/buildoor*|/usr/local/bin/node*)
  printf '%s %s ' "$f" "$p"; sha256sum "$f" || exit 1;;
esac
done"""
    return docker("exec", "--user", "root", node["container"], "sh", "-c", script)


def snapshot(output):
    output.mkdir(parents=True, exist_ok=False)
    nodes = inventory()
    records = []
    for service, node in sorted(nodes.items()):
        node["running_executable_hashes"] = executable_hashes(node)
        if not service.startswith(("el-", "cl-")):
            continue
        if service.startswith("el-"):
            url = endpoint(node, "8545/tcp")
            for method, params in [
                ("web3_clientVersion", []),
                ("eth_chainId", []),
                ("eth_getBlockByNumber", ["0x0", False]),
                ("eth_getBlockByNumber", ["finalized", False]),
                ("eth_getBlockByNumber", ["latest", False]),
                ("eth_syncing", []),
            ]:
                records.append({"service": service, **exchange(url, method, params)})
        else:
            url = endpoint(node, "4000/tcp")
            for path in [
                "/eth/v1/node/version",
                "/eth/v1/node/syncing",
                "/eth/v1/node/identity",
                "/eth/v1/node/peers",
                "/eth/v1/config/spec",
                "/eth/v1/beacon/genesis",
                "/eth/v1/beacon/states/head/finality_checkpoints",
            ]:
                records.append({"service": service, **exchange(url + path)})
    (output / "inventory.json").write_text(
        json.dumps({"at": now(), "enclave": ENCLAVE, "services": nodes}, indent=2)
        + "\n"
    )
    with (output / "exchanges.jsonl").open("x") as stream:
        for record in records:
            stream.write(json.dumps(record) + "\n")
    # Retain failures before validation; HTTP success is not tuple qualification.
    assert_private_network(nodes)
    for record in records:
        decoded(record)
    print(
        json.dumps(
            {"snapshot": str(output), "services": len(nodes), "exchanges": len(records)}
        )
    )


def percentile(values, fraction):
    if not values:
        raise ValueError("no latency samples")
    return sorted(values)[max(0, math.ceil(len(values) * fraction) - 1)]


def stable_identity(nodes):
    return {
        name: {
            key: node[key]
            for key in [
                "container",
                "image_id",
                "command",
                "entrypoint",
                "memory_limit",
                "nano_cpus",
            ]
        }
        | {"started_at": node["state"]["StartedAt"]}
        for name, node in nodes.items()
    }


def engine_window(path, begin, end):
    times, failed, inside, completed, after, statuses = [], [], {}, {}, {}, {}
    with path.open() as stream:
        for index, line in enumerate(stream):
            try:
                rec = json.loads(line)
                stamp = datetime.fromisoformat(
                    rec["started"].replace("Z", "+00:00")
                ).timestamp()
                session, sequence = rec["session"], rec["sequence"]
                completed.setdefault(session, set()).add(sequence)
                if stamp >= end:
                    after[session] = min(after.get(session, sequence), sequence)
                if not begin <= stamp < end:
                    continue
                inside.setdefault(session, []).append(sequence)
                times.append(rec["elapsed_ns"] / 1e6)
                reply = json.loads(base64.b64decode(rec["response_body"]))
                request = json.loads(base64.b64decode(rec["request_body"]))
                result = reply.get("result", {})
                if isinstance(result, dict):
                    result = result.get("payloadStatus", result)
                status = (
                    result.get("status", "success")
                    if isinstance(result, dict)
                    else "success"
                )
                label = request["method"] + ":" + status
                statuses[label] = statuses.get(label, 0) + 1
                if (
                    not rec["complete"]
                    or rec["http_status"] != 200
                    or reply.get("error")
                ):
                    failed.append(index + 1)
            except (ValueError, KeyError, TypeError):
                failed.append(index + 1)
    gaps, unsealed = {}, []
    for session, sequences in inside.items():
        boundary = after.get(session)
        if boundary is None:
            unsealed.append(session)
        upper = boundary if boundary is not None else max(sequences)
        missing = sorted(set(range(min(sequences), upper + 1)) - completed[session])
        if missing:
            gaps[session] = missing
    return {
        "count": len(times),
        "failed_lines": failed,
        "sequence_gaps": gaps,
        "unsealed_sessions": unsealed,
        "statuses": statuses,
        "p95_ms": percentile(times, 0.95) if times else None,
        "p99_ms": percentile(times, 0.99) if times else None,
    }


def capture_engine_window(output, nodes, begin, end):
    # Let the normal next-slot traffic seal sequence coverage. Missing or
    # unfinished requests cannot disappear from the end of the measured window.
    time.sleep(max(0, end + 10 - time.time()))
    result = {}
    for name, node in nodes.items():
        path = output / (name + "-engine.jsonl")
        with path.open("wb") as dest:
            subprocess.run(
                [
                    "docker",
                    "exec",
                    "--user",
                    "root",
                    node["container"],
                    "cat",
                    "/data/engine-exchanges.jsonl",
                ],
                stdout=dest,
                check=True,
                timeout=60,
            )
        result[name] = engine_window(path, begin, end)
    return result


def steady(output, duration):
    output.mkdir(parents=True, exist_ok=False)
    nodes = inventory()
    assert_private_network(nodes)
    before = stable_identity(nodes)
    profile = {
        "declared_at": now(),
        "duration_seconds": duration,
        "sample_interval_seconds": 3,
        "identities": before,
        "rpc_p95_ms": 500,
        "rpc_p99_ms": 2000,
        "engine_p95_ms": 1000,
        "engine_p99_ms": 3000,
        "missing_slot_fraction_max": 0.1,
        "scope": "Unchanged running validators/builder; no synthetic workload. Finality and missing-slot acceptance require a separate unchanged-reference verification.",
    }
    (output / "profile.json").write_text(json.dumps(profile, indent=2) + "\n")
    el = {name: node for name, node in nodes.items() if name.startswith("el-")}
    latencies = {name: [] for name in el}
    errors = []
    started = time.monotonic()
    deadline = started + duration
    samples = 0
    with (output / "samples.jsonl").open("x") as stream, ThreadPoolExecutor(
        max_workers=8
    ) as executor:
        while time.monotonic() < deadline:
            calls = {
                name: executor.submit(
                    exchange, endpoint(node, "8545/tcp"), "eth_blockNumber", []
                )
                for name, node in el.items()
            }
            row = {
                "at": now(),
                "rpc": {name: call.result() for name, call in calls.items()},
            }
            row["containers"] = [
                json.loads(line)
                for line in docker(
                    "stats",
                    "--no-stream",
                    "--format",
                    "{{json .}}",
                    *[n["container"] for n in nodes.values()]
                ).splitlines()
            ]
            for name, call in row["rpc"].items():
                latencies[name].append(call["elapsed_ms"])
                try:
                    int(decoded(call)["result"], 16)
                except (RuntimeError, ValueError, KeyError) as error:
                    errors.append(
                        {"service": name, "sample": samples, "error": str(error)}
                    )
            stream.write(json.dumps(row) + "\n")
            stream.flush()
            samples += 1
            time.sleep(
                max(
                    0,
                    min(
                        deadline - time.monotonic(),
                        started + samples * 3 - time.monotonic(),
                    ),
                )
            )
    report = {
        "window_ended_at": now(),
        "samples": samples,
        "rpc_errors": errors,
        "rpc": {
            name: {
                "count": len(v),
                "p95_ms": percentile(v, 0.95),
                "p99_ms": percentile(v, 0.99),
            }
            for name, v in latencies.items()
        },
        "identity_unchanged": before == stable_identity(inventory()),
        "finality_evaluated": False,
        "passed": False,
    }
    # Capture exact exchanges for the declared window, retaining malformed or
    # incomplete records as failures instead of filtering them out of results.
    begin = datetime.fromisoformat(profile["declared_at"]).timestamp()
    end = begin + duration
    report["engine"] = capture_engine_window(output, el, begin, end)
    report["latency_passed"] = (
        not errors
        and report["identity_unchanged"]
        and samples >= duration // 3
        and all(
            v["p95_ms"] <= 500 and v["p99_ms"] <= 2000 for v in report["rpc"].values()
        )
        and all(
            v["count"] > 100
            and not v["failed_lines"]
            and not v["sequence_gaps"]
            and not v["unsealed_sessions"]
            and v["p95_ms"] <= 1000
            and v["p99_ms"] <= 3000
            for v in report["engine"].values()
        )
    )
    (output / "latency-result.json").write_text(json.dumps(report, indent=2) + "\n")
    print(json.dumps(report))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--steady-seconds", type=int)
    args = parser.parse_args()
    if args.steady_seconds is not None:
        if args.steady_seconds < 1200:
            parser.error("the declared steady campaign requires at least 1200 seconds")
        steady(args.output, args.steady_seconds)
    else:
        snapshot(args.output)


if __name__ == "__main__":
    main()
