#!/usr/bin/env python3
"""Exercise real isolated EL processes and OS materialization faults.

The supplied public private-network genesis is copied into a new owned directory
with Amsterdam active at genesis. No running validator or original DB is used.
This is an EL lifecycle/storage campaign; CL FULL/EMPTY/PENDING is separate.
"""
import argparse
import base64
import copy
import hashlib
import hmac
import json
import os
import resource
import signal
import socket
import subprocess
import time
import urllib.request
from pathlib import Path

ZERO = "0x" + "00" * 32
ADDRESS = "0x" + "00" * 19 + "01"
OPENER = urllib.request.build_opener(urllib.request.ProxyHandler({}))


def port():
    with socket.socket() as server:
        server.bind(("127.0.0.1", 0))
        return server.getsockname()[1]


def token(secret):
    def part(value):
        return base64.urlsafe_b64encode(value).rstrip(b"=")
    body = part(b'{"alg":"HS256","typ":"JWT"}') + b"." + part(json.dumps({"iat": int(time.time())}).encode())
    return (body + b"." + part(hmac.new(secret, body, hashlib.sha256).digest())).decode()


class Node:
    def __init__(self, binary, directory, genesis, trace):
        self.binary, self.directory, self.genesis, self.trace = binary, directory, genesis, trace
        self.rpc_port, self.engine_port, self.p2p_port = port(), port(), port()
        self.secret = bytes.fromhex("42" * 32)  # Disposable isolated development auth.
        directory.mkdir()
        self.jwt = directory / "jwt.hex"
        self.jwt.write_text(self.secret.hex())
        self.process = None

    def call(self, method, params, allow_error=False):
        request = {"jsonrpc": "2.0", "id": 1, "method": method, "params": params}
        engine = method.startswith("engine_")
        headers = {"Content-Type": "application/json"}
        if engine:
            headers["Authorization"] = "Bearer " + token(self.secret)
        started = time.monotonic()
        req = urllib.request.Request("http://127.0.0.1:" + str(self.engine_port if engine else self.rpc_port), json.dumps(request).encode(), headers)
        try:
            with OPENER.open(req, timeout=15) as response:
                raw = response.read().decode()
            self.trace.write(json.dumps({"node": self.directory.name, "request": request, "response": raw, "elapsed_ms": (time.monotonic()-started)*1000}) + "\n")
            self.trace.flush()
        except Exception as error:
            self.trace.write(json.dumps({"node": self.directory.name, "request": request, "transport_failure": str(error)}) + "\n")
            self.trace.flush()
            raise
        reply = json.loads(raw)
        assert reply["id"] == 1 and reply["jsonrpc"] == "2.0"
        if not allow_error:
            assert "error" not in reply, reply
            return reply["result"]
        return reply

    def start(self):
        args = [str(self.binary), "--datadir="+str(self.directory/"data"), "--override.genesis="+str(self.genesis),
                "--networkid=1337", "--http.addr=127.0.0.1", "--http.port="+str(self.rpc_port),
                "--authrpc.addr=127.0.0.1", "--authrpc.port="+str(self.engine_port),
                "--authrpc.jwtsecret="+str(self.jwt), "--port="+str(self.p2p_port),
                "--discovery.port=0", "--maxpeers=0", "--bootnodes=", "--rpc.rate_limit=0"]
        self.log = (self.directory/"node.log").open("ab")
        self.process = subprocess.Popen(args, stdout=self.log, stderr=subprocess.STDOUT, start_new_session=True)
        deadline = time.monotonic()+60
        while time.monotonic()<deadline:
            if self.process.poll() is not None:
                raise RuntimeError("native process exited; see " + str(self.directory/"node.log"))
            try:
                return self.call("eth_getBlockByNumber", ["latest", False])
            except OSError:
                time.sleep(.2)
        raise RuntimeError("isolated node not ready within 60 seconds")

    def stop(self, crash=False):
        if self.process is not None and self.process.poll() is None:
            os.killpg(self.process.pid, signal.SIGKILL if crash else signal.SIGTERM)
            self.process.wait(timeout=30)
        if hasattr(self, "log"):
            self.log.close()

    def choose(self, head, safe=ZERO, final=ZERO, attrs=None, allow_error=False):
        return self.call("engine_forkchoiceUpdatedV4", [{"headBlockHash": head, "safeBlockHash": safe, "finalizedBlockHash": final}, attrs], allow_error)


def committed_head(wal, initial):
    """Decode actual durable WAL commits; discard only an incomplete suffix."""
    position, pending, head = 0, {}, initial
    while position < len(wal):
        operation = wal[position]
        position += 1
        if operation == 3:
            if b"hb" in pending:
                head = "0x" + pending[b"hb"].hex()
            pending = {}
            continue
        if operation not in [1, 2] or position + 4 > len(wal):
            break
        key_size = int.from_bytes(wal[position:position+4], "big")
        position += 4
        if position + key_size + 4 > len(wal):
            break
        key = wal[position:position+key_size]
        position += key_size
        value_size = int.from_bytes(wal[position:position+4], "big")
        position += 4
        if position + value_size > len(wal):
            break
        pending[key] = wal[position:position+value_size]
        position += value_size
    return head


def run(binary, genesis_path, output, fault_kind="materialization"):
    output.mkdir(parents=True, exist_ok=False)
    genesis = json.loads(genesis_path.read_text())
    assert int(str(genesis["config"]["chainId"]), 0) == 1337
    genesis["config"]["amsterdamTime"] = 0
    original_time = int(str(genesis["timestamp"]), 0)
    owned_genesis = output/"genesis.json"
    owned_genesis.write_text(json.dumps(genesis))
    report = {"passed": False, "fault_kind": fault_kind, "scope": "actual independent EL processes, not complete CL lifecycle", "cases": []}
    trace = (output/"exchanges.jsonl").open("x")
    proposer = Node(binary, output/"proposer", owned_genesis, trace)
    verifier = Node(binary, output/"verifier", owned_genesis, trace)
    def save():
        (output/"result.json").write_text(json.dumps(report, indent=2)+"\n")
    def case(name):
        report["cases"].append(name)
        save()
    def build(parent, slot, discriminator):
        attrs = {"timestamp": hex(original_time+slot*3), "slotNumber": hex(slot), "withdrawals": [],
                 "prevRandao": "0x"+format(discriminator,"02x")*32,
                 "suggestedFeeRecipient": ADDRESS, "parentBeaconBlockRoot": ZERO, "targetGasLimit": hex(150000000)}
        selected = proposer.choose(parent, attrs=attrs)
        assert selected["payloadStatus"]["status"] == "VALID" and selected["payloadId"]
        return proposer.call("engine_getPayloadV6", [selected["payloadId"]])
    def submit(node, built, allow_error=False):
        return node.call("engine_newPayloadV5", [built["executionPayload"], [], ZERO, built["executionRequests"]], allow_error)
    try:
        first = proposer.start()
        assert verifier.start()["hash"] == first["hash"]
        genesis_hash = first["hash"]
        one = build(genesis_hash, 1, 1)
        h1 = one["executionPayload"]["blockHash"]
        # Keep the unavailable nominated hash distinct from h1. If h1 had
        # already been nominated, publishing it when it arrives is correct.
        assert verifier.choose("0x" + "ff" * 32)["payloadStatus"]["status"] == "SYNCING"
        assert verifier.call("eth_getBlockByNumber", ["latest", False])["hash"] == genesis_hash
        case("missing payload nomination never publishes unknown head")
        bad = copy.deepcopy(one)
        bad["executionPayload"]["slotNumber"] = "0x2"
        assert submit(verifier, bad)["status"] == "INVALID"
        assert submit(verifier, one)["status"] == "VALID"
        assert submit(verifier, one)["status"] == "VALID"
        assert verifier.call("eth_getBlockByNumber", ["latest", False])["hash"] == genesis_hash
        assert verifier.choose(h1)["payloadStatus"]["status"] == "VALID"
        case("invalid slot rejected; valid duplicate stable; import awaits forkchoice")
        assert submit(proposer, one)["status"] == "VALID"
        assert proposer.choose(h1)["payloadStatus"]["status"] == "VALID"
        left = build(h1, 2, 2)
        right = build(h1, 2, 3)
        h2, other = [x["executionPayload"]["blockHash"] for x in [left,right]]
        assert h2 != other
        assert submit(verifier, left)["status"] == "VALID"
        assert submit(verifier, right)["status"] == "VALID"
        for head in [h2, other, h2]:
            assert verifier.choose(head, safe=h1)["payloadStatus"]["status"] == "VALID"
            assert verifier.call("eth_getBlockByNumber", ["latest", False])["hash"] == head
        case("same-height canonical reorg both ways preserves exact heads")
        assert verifier.choose(h2, safe=h2, final=h2)["payloadStatus"]["status"] == "VALID"
        rejected = verifier.choose(other, allow_error=True)
        assert "error" in rejected or rejected.get("result",{}).get("payloadStatus",{}).get("status") != "VALID"
        assert verifier.call("eth_getBlockByNumber", ["finalized", False])["hash"] == h2
        case("finality blocks competing branch and survives duplicate forkchoice")
        # A real OS EISDIR fault after the durable WAL commit, before head
        # materialization. No database/test hook or validation bypass is used.
        assert submit(proposer, left)["status"] == "VALID"
        assert proposer.choose(h2)["payloadStatus"]["status"] == "VALID"
        three = build(h2, 3, 4)
        h3 = three["executionPayload"]["blockHash"]
        assert submit(verifier, three)["status"] == "VALID"
        fault = verifier.directory/"data"/"chaindata"/"data"/(b"hb".hex()+".tmp")
        wal = verifier.directory/"data"/"chaindata"/"wal"
        before_size = wal.stat().st_size
        if fault_kind == "materialization":
            fault.mkdir()
        else:
            # Allow exactly one more WAL byte. The real write then returns a
            # short count/EFBIG; no FileDB hook or crafted WAL substitutes it.
            soft, hard = resource.prlimit(verifier.process.pid, resource.RLIMIT_FSIZE)
            resource.prlimit(verifier.process.pid, resource.RLIMIT_FSIZE, (before_size + 1, hard))
        rejected = verifier.choose(h3, safe=h2, final=h2, allow_error=True)
        assert "error" in rejected, rejected
        fenced = verifier.choose(h3, safe=h2, final=h2, allow_error=True)
        assert "error" in fenced, fenced
        report["wal_bytes_before_crash"] = wal.stat().st_size
        assert report["wal_bytes_before_crash"] > 0
        actual_wal = wal.read_bytes()
        (output/"fault-wal.bin").write_bytes(actual_wal)
        expected_head = committed_head(actual_wal, h2)
        report["expected_head_from_actual_wal"] = expected_head
        if fault_kind == "short-write":
            assert len(actual_wal) == before_size + 1
            assert expected_head == h2
        else:
            assert expected_head == h3
        case("actual OS write failure returns resource error and fences retry")
        verifier.stop(crash=True)
        if fault_kind == "materialization":
            fault.rmdir()
        recovered = verifier.start()
        assert recovered["hash"] == expected_head
        assert verifier.call("eth_getBlockByNumber", ["finalized", False])["hash"] == h2
        assert verifier.choose(h3, safe=h2, final=h2)["payloadStatus"]["status"] == "VALID"
        assert submit(verifier, three)["status"] == "VALID"
        case("SIGKILL and WAL recovery restore committed head with original finality")
        verifier.stop()
        assert verifier.start()["hash"] == h3
        case("normal indexed restart preserves recovered exact head")
        report.update(passed=True, genesis_hash=genesis_hash, final_head=h3, finalized=h2)
        save()
    except Exception as error:
        report["failure"] = {"class": type(error).__name__, "message": str(error)}
        save()
        raise
    finally:
        proposer.stop()
        verifier.stop()
        trace.close()
    print(json.dumps(report))


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", required=True, type=Path)
    parser.add_argument("--genesis", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--fault", choices=["materialization", "short-write"], default="materialization")
    args = parser.parse_args()
    run(args.binary, args.genesis, args.output, args.fault)
