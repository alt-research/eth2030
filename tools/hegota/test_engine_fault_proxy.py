import importlib.util
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
from pathlib import Path
from tempfile import TemporaryDirectory
import threading
import unittest
from urllib.request import Request, urlopen

spec = importlib.util.spec_from_file_location("engine_fault_proxy", Path(__file__).with_name("engine_fault_proxy.py"))
proxy = importlib.util.module_from_spec(spec)
spec.loader.exec_module(proxy)


class FaultProxyTests(unittest.TestCase):
    def test_build_fault_does_not_change_validation_view_or_verdict(self):
        raw = "0x0102"
        build = {"method": "engine_forkchoiceUpdatedV5", "params": [{}, {"inclusionListTransactions": [raw]}, None]}
        value, changed = proxy.build_request(build, {"mode": "omit", "transaction": raw})
        self.assertTrue(changed)
        self.assertEqual(value["params"][1]["inclusionListTransactions"], [])
        validation = {"method": "engine_newPayloadV6", "params": [{}, [], "root", [], [raw]]}
        self.assertEqual(proxy.build_request(validation, {"mode": "omit", "transaction": raw}), (validation, False))
        other = {"method": "engine_forkchoiceUpdatedV5", "params": [{}, {"inclusionListTransactions": ["0x03"]}]}
        self.assertFalse(proxy.build_request(other, {"mode": "omit", "transaction": raw})[1])

    def test_lighthouse_ssz_view_is_decoded_at_transaction_boundaries(self):
        encoded = b"\x08\0\0\0\x0a\0\0\0\x01\x02\x03"
        self.assertEqual(proxy.decode_ssz_transactions(encoded), [b"\x01\x02", b"\x03"])
        request = {"method": "engine_forkchoiceUpdatedV5", "params": [{}, {"inclusion_list_transactions": "0x" + encoded.hex()}]}
        self.assertFalse(proxy.build_request(json.loads(json.dumps(request)), {"mode": "omit", "transaction": "0x0203"})[1])
        changed, yes = proxy.build_request(request, {"mode": "omit", "transaction": "0x0102"})
        self.assertTrue(yes)
        self.assertEqual(changed["params"][1]["inclusion_list_transactions"], "0x")
        for malformed in [b"\0", b"\x03\0\0\0", b"\x08\0\0\0\x07\0\0\0"]:
            with self.assertRaises(ValueError):
                proxy.decode_ssz_transactions(malformed)

    def test_injected_transport_samples_preserve_protocol_byte_bound(self):
        self.assertEqual(len(proxy.transaction_bytes(["0x" + "ab" * 8192])[0]), 8192)
        for malformed in [["0x"], ["0x1"], ["0xzz"], ["0x" + "ab" * 8193], "0xab"]:
            with self.assertRaises(ValueError):
                proxy.transaction_bytes(malformed)

    def test_native_ports_support_both_argument_spellings(self):
        for args in [["--authrpc.port=8551"], ["--authrpc.port", "8551"], ["--http"]]:
            port, updated = proxy.native_arguments(args)
            self.assertEqual(port, 8551)
            self.assertIn("18551", " ".join(updated))
        with self.assertRaises(ValueError):
            proxy.native_arguments(["--authrpc.port=18551"])

    def test_real_http_forwarding_preserves_auth_and_native_unsatisfied_status(self):
        observed = []
        native_result = {"status": "VALID", "inclusionListSatisfied": False, "latestValidHash": "0x01", "validationError": None}

        class Native(BaseHTTPRequestHandler):
            def log_message(self, *_):
                pass

            def do_POST(self):
                message = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
                observed.append((message, self.headers.get("Authorization")))
                result = [] if message["method"] == "engine_getInclusionListV1" else native_result
                encoded = json.dumps({"jsonrpc": "2.0", "id": message["id"], "result": result}).encode()
                self.send_response(200)
                self.send_header("Content-Length", str(len(encoded)))
                self.end_headers()
                self.wfile.write(encoded)

        with TemporaryDirectory() as directory:
            config = Path(directory, "control.json")
            journal = Path(directory, "events.jsonl")
            config.write_text(json.dumps({"mode": "omit", "transaction": "0x0102"}))
            native = ThreadingHTTPServer(("127.0.0.1", 0), Native)
            server = proxy.Proxy(("127.0.0.1", 0), native.server_address, config, journal)
            for endpoint in (native, server):
                threading.Thread(target=endpoint.serve_forever, daemon=True).start()
            try:
                message = {"jsonrpc": "2.0", "id": 7, "method": "engine_newPayloadV6", "params": [{"blockHash": "0x01"}, [], "root", [], ["0x0102"]]}
                request = Request("http://127.0.0.1:" + str(server.server_port), data=json.dumps(message).encode(), headers={"Authorization": "Bearer public-test-sentinel"})
                with urlopen(request) as response:
                    self.assertEqual(json.load(response)["result"], native_result)
                self.assertEqual(observed, [(message, "Bearer public-test-sentinel")])
                event = json.loads(journal.read_text())
                self.assertEqual(event["result"], native_result)
                self.assertEqual(event["inclusion_list"], ["0x0102"])
                self.assertNotIn("public-test-sentinel", journal.read_text())
                self.assertLessEqual(event["start_ns"], event["end_ns"])
                config.write_text(json.dumps({"mode": "omit", "transaction": "0x0102", "transactions": ["0x0102"]}))
                request = Request("http://127.0.0.1:" + str(server.server_port), data=json.dumps({"jsonrpc": "2.0", "id": 8, "method": "engine_getInclusionListV1", "params": []}).encode())
                with urlopen(request) as response:
                    self.assertEqual(json.load(response)["result"], ["0x0102"])
                events = [json.loads(line) for line in journal.read_text().splitlines()]
                self.assertTrue(events[-1]["test_il_injected"])
                self.assertEqual(events[0]["result"], native_result)
            finally:
                for endpoint in (server, native):
                    endpoint.shutdown()
                    endpoint.server_close()


if __name__ == "__main__":
    unittest.main()
