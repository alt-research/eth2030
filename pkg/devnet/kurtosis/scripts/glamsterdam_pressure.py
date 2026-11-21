#!/usr/bin/env python3
"""Bounded private-chain cancellation/invalid-payload campaign with valid controls."""

import argparse
import base64
import copy
import hashlib
import hmac
import http.client
import json
import subprocess
import threading
import time
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path
from urllib.parse import urlsplit
import urllib.request

import glamsterdam_campaign as c

ACCOUNT = "0x7e5f4552091a69125d5dfcb7b8c2659029395bdf"
TARGET = "0x0000000000000000000000000000000000001234"


def call_params(block_hash, gas, code):
    return [
        {"to": TARGET, "gas": hex(gas)},
        {"blockHash": block_hash, "requireCanonical": True},
        {TARGET: {"code": code}},
    ]


def invalid_payload(original):
    altered = copy.deepcopy(original)
    assert altered["method"] == "engine_newPayloadV5"
    assert altered["params"][0]["blockHash"] != "0x" + "ff" * 32
    altered["params"][0]["blockHash"] = "0x" + "ff" * 32
    return altered


def jwt(secret):
    def encoded(value):
        return base64.urlsafe_b64encode(value).rstrip(b"=")

    message = (
        encoded(b'{"alg":"HS256","typ":"JWT"}')
        + b"."
        + encoded(json.dumps({"iat": int(time.time())}).encode())
    )
    return (
        message + b"." + encoded(hmac.new(secret, message, hashlib.sha256).digest())
    ).decode()


def run(output, duration):
    from websockets.sync.client import connect

    output.mkdir(parents=True, exist_ok=False)
    nodes = c.inventory()
    c.assert_private_network(nodes)
    el = {name: node for name, node in nodes.items() if name.startswith("el-")}
    native = {name: node for name, node in el.items() if not name.startswith("el-1-")}
    identity = c.stable_identity(nodes)
    anchors = {
        name: c.decoded(
            c.exchange(
                c.endpoint(node, "8545/tcp"),
                "eth_getBlockByNumber",
                ["finalized", False],
            )
        )["result"]
        for name, node in el.items()
    }
    assert len({header["hash"] for header in anchors.values()}) == 1
    controls, secrets, payloads = {}, {}, {}
    for name, node in native.items():
        base, block = c.endpoint(node, "8545/tcp"), anchors[name]["hash"]
        low = c.exchange(base, "eth_call", call_params(block, 22000, "0x5b600056"))
        value = json.loads(low["response"])
        assert "out of gas" in value["error"]["message"].lower(), low
        long = c.exchange(base, "eth_call", call_params(block, 0xFFFFFF, "0x5b600056"))
        value = json.loads(long["response"])
        assert (
            "out of gas" in value["error"]["message"].lower()
            and long["elapsed_ms"] > 25
        ), long
        controls[name] = {"short_evm_out_of_gas": low, "long_evm_out_of_gas": long}
        # Credentials are used in memory only and are never evidence fields.
        secrets[name] = bytes.fromhex(
            c.docker(
                "exec", "--user", "root", node["container"], "cat", "/jwt/jwtsecret"
            )
            .strip()
            .removeprefix("0x")
        )
        tail = c.docker(
            "exec",
            "--user",
            "root",
            node["container"],
            "tail",
            "-c",
            "2097152",
            "/data/engine-exchanges.jsonl",
        )
        for line in reversed(tail.splitlines()[1:]):
            record = json.loads(line)
            request = json.loads(base64.b64decode(record["request_body"]))
            reply = json.loads(base64.b64decode(record["response_body"]))
            if (
                request["method"] == "engine_newPayloadV5"
                and reply.get("result", {}).get("status") == "VALID"
            ):
                payloads[name] = invalid_payload(request)
                break
        assert name in payloads
    (output / "preflight.json").write_text(json.dumps(controls, indent=2) + "\n")
    profile = {
        "declared_at": c.now(),
        "duration_seconds": duration,
        "identities": identity,
        "scope": "Main three EL/CL pairs with two fixed validator-free observer pairs on the same host. Original 384 validators and builder settings unchanged. No throughput extrapolation.",
        "load": "Per native EL: two HTTP and two WebSocket sequential EVM-loop callers, disconnect after 25 ms, then pause 75 ms; one invalid-hash Engine payload per second. Valid proof/header/finite EVM controls on all three ELs every two seconds.",
        "rpc_p95_ms": 500,
        "rpc_p99_ms": 2000,
        "engine_p95_ms": 1000,
        "engine_p99_ms": 3000,
        "missing_slot_fraction_max": 0.1,
        "valid_control_interval_seconds": 2,
        "required_control_rounds": duration // 2,
        "resource_interval_seconds": 3,
        "anchors": anchors,
    }
    (output / "profile.json").write_text(json.dumps(profile, indent=2) + "\n")
    stop = threading.Event()
    lock = threading.Lock()
    stream = (output / "observations.jsonl").open("x")
    samples = {name: [] for name in el}
    failures, counts = [], {}

    def emit(record):
        with lock:
            stream.write(json.dumps({"at": c.now(), **record}) + "\n")
            stream.flush()
            label = record["kind"] + ":" + record.get("outcome", "completed")
            counts[label] = counts.get(label, 0) + 1

    def cancelled(name, node, transport):
        url = c.endpoint(node, "8545/tcp" if transport == "http" else "8546/tcp")
        request = {
            "jsonrpc": "2.0",
            "id": 1,
            "method": "eth_call",
            "params": call_params(anchors[name]["hash"], 0xFFFFFF, "0x5b600056"),
        }
        while not stop.is_set():
            begin, connection = time.monotonic(), None
            row = {
                "kind": transport + "-disconnect",
                "service": name,
                "request": request,
            }
            try:
                if transport == "http":
                    address = urlsplit(url)
                    connection = http.client.HTTPConnection(
                        address.hostname, address.port, timeout=2
                    )
                    connection.request(
                        "POST",
                        "/",
                        json.dumps(request),
                        {"Content-Type": "application/json"},
                    )
                    time.sleep(0.025)
                    row["outcome"] = "socket_closed_after_send"
                else:
                    connection = connect(
                        url.replace("http:", "ws:"),
                        proxy=None,
                        compression=None,
                        open_timeout=2,
                        close_timeout=0,
                    )
                    connection.send(json.dumps(request))
                    try:
                        row["response"] = connection.recv(timeout=0.025)
                        row["outcome"] = "completed_before_disconnect"
                    except TimeoutError:
                        row["outcome"] = "socket_closed_while_pending"
            except Exception as error:
                row.update(
                    outcome="transport_error",
                    error={"class": type(error).__name__, "message": str(error)},
                )
            finally:
                if connection is not None:
                    if transport == "http":
                        connection.close()
                    else:
                        connection.close_socket()
                row["elapsed_ms"] = (time.monotonic() - begin) * 1000
                emit(row)
            stop.wait(0.075)

    def invalid(name, node):
        while not stop.is_set():
            request = urllib.request.Request(
                c.endpoint(node, "8551/tcp"),
                json.dumps(payloads[name]).encode(),
                {
                    "Content-Type": "application/json",
                    "Authorization": "Bearer " + jwt(secrets[name]),
                },
            )
            row = {
                "kind": "invalid-payload",
                "service": name,
                "request": payloads[name],
            }
            try:
                with c.OPENER.open(request, timeout=3) as response:
                    row.update(
                        http_status=response.status,
                        response=json.loads(response.read()),
                    )
                assert (
                    row["http_status"] == 200
                    and row["response"]["result"]["status"] == "INVALID"
                ), row
            except Exception as error:
                row["error"] = {"class": type(error).__name__, "message": str(error)}
                with lock:
                    failures.append(row)
            emit(row)
            stop.wait(1)

    def resources():
        while not stop.is_set():
            sample_started = time.monotonic()
            try:
                emit(
                    {
                        "kind": "resources",
                        "containers": [
                            json.loads(line)
                            for line in c.docker(
                                "stats",
                                "--no-stream",
                                "--format",
                                "{{json .}}",
                                *[node["container"] for node in nodes.values()]
                            ).splitlines()
                        ],
                    }
                )
            except Exception as error:
                row = {
                    "kind": "resources",
                    "error": {"class": type(error).__name__, "message": str(error)},
                }
                with lock:
                    failures.append(row)
                emit(row)
            stop.wait(max(0, 3 - (time.monotonic() - sample_started)))

    begin = time.monotonic()
    control_rounds = 0
    try:
        with ThreadPoolExecutor(max_workers=16) as executor:
            jobs = [
                executor.submit(cancelled, name, node, transport)
                for name, node in native.items()
                for transport in ("http", "http", "ws", "ws")
            ]
            jobs += [
                executor.submit(invalid, name, node) for name, node in native.items()
            ]
            jobs.append(executor.submit(resources))
            try:
                while time.monotonic() - begin < duration:
                    for name, node in el.items():
                        header = anchors[name]
                        for kind, method, params in [
                            ("header", "eth_getBlockByHash", [header["hash"], False]),
                            (
                                "proof",
                                "eth_getProof",
                                [
                                    ACCOUNT,
                                    [],
                                    {
                                        "blockHash": header["hash"],
                                        "requireCanonical": True,
                                    },
                                ],
                            ),
                            (
                                "finite-call",
                                "eth_call",
                                call_params(
                                    header["hash"], 100000, "0x602a60005260206000f3"
                                ),
                            ),
                        ]:
                            rec = c.exchange(
                                c.endpoint(node, "8545/tcp"), method, params
                            )
                            samples[name].append(rec["elapsed_ms"])
                            row = {
                                "kind": kind,
                                "service": name,
                                "expected_hash": header["hash"],
                                "header": header,
                                **rec,
                            }
                            try:
                                response = c.decoded(rec)
                                if kind == "finite-call":
                                    assert int(response["result"], 16) == 42
                                elif kind == "header":
                                    assert response["result"]["hash"] == header["hash"]
                                row["response"] = response
                            except Exception as error:
                                failures.append(
                                    {"service": name, "kind": kind, "error": str(error)}
                                )
                            emit(row)
                    control_rounds += 1
                    stop.wait(
                        max(
                            0,
                            min(begin + duration, begin + control_rounds * 2)
                            - time.monotonic(),
                        )
                    )
            finally:
                stop.set()
            for job in jobs:
                job.result()
    finally:
        stop.set()
        stream.close()
    ended = time.time()
    report = {
        "window_ended_at": c.now(),
        "counts": counts,
        "control_rounds": control_rounds,
        "required_control_rounds": duration // 2,
        "failures": failures,
        "identity_unchanged": identity == c.stable_identity(c.inventory()),
        "rpc": {
            name: {
                "count": len(v),
                "p95_ms": c.percentile(v, 0.95),
                "p99_ms": c.percentile(v, 0.99),
            }
            for name, v in samples.items()
        },
    }
    from datetime import datetime

    started = datetime.fromisoformat(profile["declared_at"]).timestamp()
    # Worker termination time is included rather than silently discarding tail work.
    report["engine"] = c.capture_engine_window(output, el, started, ended)
    report["latency_passed"] = (
        not failures
        and control_rounds >= duration // 2
        and report["identity_unchanged"]
        and not any(key.endswith(":transport_error") for key in counts)
        and counts.get("ws-disconnect:socket_closed_while_pending", 0) > 100
        and counts.get("http-disconnect:socket_closed_after_send", 0) > 100
        and all(
            v["count"] >= 30 and v["p95_ms"] <= 500 and v["p99_ms"] <= 2000
            for v in report["rpc"].values()
        )
        and all(
            v["count"] > 20
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


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--seconds", type=int, default=180)
    args = parser.parse_args()
    if args.seconds < 180:
        parser.error("the pressure window must cover at least 180 seconds")
    run(args.output, args.seconds)
