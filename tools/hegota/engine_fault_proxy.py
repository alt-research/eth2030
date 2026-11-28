#!/usr/bin/env python3
"""Private-devnet fault injector; never changes an execution result.

Only builder IL inputs or IL-generation test data may be changed. Actual
newPayload/forkchoice responses always come from the native execution node.
JWT headers are forwarded and are never retained in the evidence journal.
"""
import argparse
import http.client
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import os
from pathlib import Path
import signal
import subprocess
import threading
import time


MAX_IL_BYTES = 8192


def transaction_bytes(items):
    if not isinstance(items, list):
        raise ValueError("test IL must be an array")
    decoded = []
    for item in items:
        if not isinstance(item, str) or not item.startswith("0x"):
            raise ValueError("test IL must contain DATA")
        value = bytes.fromhex(item[2:])
        if not value or len(item[2:]) % 2:
            raise ValueError("empty or malformed test IL transaction")
        decoded.append(value)
    if sum(map(len, decoded)) > MAX_IL_BYTES:
        raise ValueError("test IL exceeds EIP-7805 byte bound")
    return decoded


def build_request(request, control):
    """Omit only an exact fixture from building, retaining validation views."""
    changed = False
    if request.get("method") == "engine_forkchoiceUpdatedV5" and control.get("mode") == "omit":
        params = request.get("params", [])
        attrs = params[1] if len(params) > 1 else None
        if attrs:
            for name in ("inclusionListTransactions", "inclusion_list_transactions"):
                listed = attrs.get(name)
                if isinstance(listed, list) and control.get("transaction") in listed:
                    attrs[name] = []
                    changed = True
                elif isinstance(listed, str) and listed.startswith("0x"):
                    # Lighthouse's alternate JSON representation is an SSZ list.
                    encoded = bytes.fromhex(listed[2:])
                    wanted = bytes.fromhex(control.get("transaction", "0x")[2:])
                    if wanted and wanted in decode_ssz_transactions(encoded):
                        attrs[name] = "0x"
                        changed = True
    return request, changed


def decode_ssz_transactions(data):
    if not data:
        return []
    if len(data) < 4:
        raise ValueError("truncated SSZ list")
    first = int.from_bytes(data[:4], "little")
    if first < 4 or first % 4 or first > len(data):
        raise ValueError("invalid SSZ list offsets")
    offsets = [int.from_bytes(data[i:i+4], "little") for i in range(0, first, 4)] + [len(data)]
    if any(a < first or a > b or b > len(data) for a, b in zip(offsets, offsets[1:])):
        raise ValueError("invalid SSZ list offsets")
    return [data[a:b] for a, b in zip(offsets, offsets[1:])]


class Proxy(ThreadingHTTPServer):
    daemon_threads = True

    def __init__(self, address, upstream, control, journal):
        super().__init__(address, Handler)
        self.upstream = upstream
        self.control = Path(control)
        self.journal = Path(journal)
        self.journal_lock = threading.Lock()

    def record(self, event):
        with self.journal_lock:
            with self.journal.open("a") as stream:
                stream.write(json.dumps(event, separators=(",", ":")) + "\n")


class Handler(BaseHTTPRequestHandler):
    def log_message(self, *_):
        pass

    def do_POST(self):
        start = time.monotonic_ns()
        unix = time.time_ns()
        try:
            body = self.rfile.read(int(self.headers.get("Content-Length", "0")))
            request = json.loads(body)
            control = json.loads(self.server.control.read_text()) if self.server.control.exists() else {}
            method = request.get("method", "")
            request, changed = build_request(request, control)
            sent = json.dumps(request).encode()
            headers = {k: v for k, v in self.headers.items() if k.lower() not in ("host", "content-length", "connection")}
            headers["Content-Length"] = str(len(sent))
            connection = http.client.HTTPConnection(*self.server.upstream, timeout=30)
            try:
                connection.request("POST", self.path, body=sent, headers=headers)
                response = connection.getresponse()
                status = response.status
                result = json.loads(response.read())
            finally:
                connection.close()
            injected = False
            if method == "engine_getInclusionListV1" and control.get("mode") in ("inject", "omit") and "transactions" in control and "error" not in result:
                if not isinstance(result.get("result"), list):
                    raise ValueError("native IL response is not an array")
                items = control["transactions"]
                transaction_bytes(items)
                result["result"] = items
                injected = True
            event = {"start_ns": start, "end_ns": time.monotonic_ns(), "unix_ns": unix, "method": method,
                     "build_view_changed": changed, "test_il_injected": injected, "http_status": status}
            if method == "engine_getInclusionListV1" and "result" in result:
                event["transactions"] = result["result"]
            if method in ("engine_newPayloadV6", "engine_forkchoiceUpdatedV5"):
                event["result"] = result.get("result")
                if method == "engine_newPayloadV6":
                    event["block_hash"] = request["params"][0].get("blockHash", request["params"][0].get("block_hash"))
                    event["inclusion_list"] = request["params"][4]
            self.server.record(event)
            encoded = json.dumps(result).encode()
            self.send_response(status)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(encoded)))
            self.end_headers()
            self.wfile.write(encoded)
        except Exception as error:
            self.server.record({"start_ns": start, "error": type(error).__name__ + ": " + str(error)})
            self.send_error(502, "private-devnet proxy failure")


def native_arguments(arguments):
    args = list(arguments)
    port = 8551
    for i, value in enumerate(args):
        if value.startswith("--authrpc.port="):
            port = int(value.split("=", 1)[1])
            args[i] = "--authrpc.port=18551"
            break
        if value == "--authrpc.port":
            port = int(args[i+1])
            args[i+1] = "18551"
            break
    else:
        args.append("--authrpc.port=18551")
    if port == 18551:
        raise ValueError("proxy and native Engine ports overlap")
    return port, args


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--control", default="/tmp/hegota-e-control.json")
    parser.add_argument("--journal", default="/tmp/hegota-e-engine.jsonl")
    args, native = parser.parse_known_args()
    if native and native[0] == "--":
        native = native[1:]
    if any(value in ("init", "version", "--version", "help") for value in native):
        os.execv("/usr/local/bin/eth2030", ["eth2030", *native])
    port, arguments = native_arguments(native)
    server = Proxy(("0.0.0.0", port), ("127.0.0.1", 18551), args.control, args.journal)
    child = subprocess.Popen(["/usr/local/bin/eth2030", *arguments], start_new_session=True)
    def stop(signum, _frame):
        os.killpg(child.pid, signum)
    signal.signal(signal.SIGTERM, stop)
    signal.signal(signal.SIGINT, stop)
    threading.Thread(target=server.serve_forever, daemon=True).start()
    try:
        raise SystemExit(child.wait())
    finally:
        server.shutdown()
        server.server_close()


if __name__ == "__main__":
    main()
