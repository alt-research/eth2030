import json
import base64
import tempfile
import tarfile
import io
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import unittest
from pathlib import Path
from unittest.mock import patch

import glamsterdam_campaign as campaign
import glamsterdam_process_lifecycle as lifecycle
import glamsterdam_pressure as pressure
import glamsterdam_artifacts as artifacts


class CampaignTests(unittest.TestCase):
    def test_runtime_identity_uses_original_container_uid_when_root_cannot_read_inode(
        self,
    ):
        node = {"container": "builder-owned-process"}
        expected = "/proc/7/exe /usr/local/bin/buildoor actual-inode-hash\n"
        with patch.object(campaign, "docker", side_effect=["", expected]) as run:
            self.assertEqual(campaign.executable_hashes(node), expected)
        self.assertEqual(
            run.call_args_list[0].args[:4],
            ("exec", "--user", "root", node["container"]),
        )
        self.assertEqual(run.call_args_list[1].args[:2], ("exec", node["container"]))

    def test_runtime_identity_rejects_an_unreadable_process_in_both_accounts(self):
        with patch.object(campaign, "docker", return_value=""):
            with self.assertRaisesRegex(
                RuntimeError, "executable identity unavailable"
            ):
                campaign.executable_hashes({"container": "no-readable-inode"})

    def test_artifact_requires_complete_original_files_and_never_replaces_prior_evidence(
        self,
    ):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            original = b"actual signed bytes\x00\xff"
            (root / "capture.bin").write_bytes(original)
            output = root / "evidence.tar.gz"
            result = artifacts.bundle(root, ["capture.bin"], output)
            self.assertTrue(result["passed"])
            self.assertEqual(result["uncompressed_bytes"], len(original))
            with self.assertRaises(FileExistsError):
                artifacts.bundle(root, ["capture.bin"], output)
            for names in [
                ["missing.json"],
                ["../capture.bin"],
                ["capture.bin", "capture.bin"],
            ]:
                with self.subTest(names=names), self.assertRaises(ValueError):
                    artifacts.bundle(root, names, root / "unqualified.tar.gz")

    def test_artifact_rejects_a_changed_committed_capture_hash(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            body = b"original signed bytes"
            (root / "capture.bin").write_bytes(body)
            original = root / "original.tar.gz"
            artifacts.bundle(root, ["capture.bin"], original)
            with tarfile.open(original, "r:*") as source:
                manifest = json.load(source.extractfile("MANIFEST.json"))
            manifest["files"][0]["sha256"] = "00" * 32
            altered = root / "altered.tar.gz"
            with tarfile.open(altered, "w:gz") as archive:
                for name, value in [
                    ("MANIFEST.json", json.dumps(manifest).encode()),
                    ("capture.bin", body),
                ]:
                    header = tarfile.TarInfo(name)
                    header.size = len(value)
                    archive.addfile(header, io.BytesIO(value))
            with self.assertRaisesRegex(ValueError, "captured bytes changed"):
                artifacts.verify(altered)

    def test_builder_fleet_is_restored_before_lifecycle_can_exit_a_live_key(self):
        state = {"builder_keys": {"target_count": 1}, "lifecycle_enabled": False}
        original = {"builder_keys": {"target_count": 2}, "lifecycle_enabled": True}
        premature_exit = []

        class Handler(BaseHTTPRequestHandler):
            def log_message(self, *args):
                pass

            def do_GET(self):
                self.send_response(200)
                self.end_headers()
                self.wfile.write(json.dumps(state).encode())

            def do_POST(self):
                body = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
                state["builder_keys"]["target_count"] = body["target"]
                self.do_GET()

            def do_PATCH(self):
                body = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
                if (
                    body["lifecycle_enabled"]
                    and state["builder_keys"]["target_count"] != 2
                ):
                    premature_exit.append(True)
                state["lifecycle_enabled"] = body["lifecycle_enabled"]
                self.do_GET()

        server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
        thread = threading.Thread(target=server.serve_forever)
        thread.start()
        try:
            rows = []
            campaign.restore_builder_runtime(
                "http://127.0.0.1:" + str(server.server_port), original, rows.append
            )
            self.assertEqual(state, original)
            self.assertFalse(premature_exit)
            self.assertEqual(
                [row["method"] for row in rows], ["GET", "POST", "PATCH", "GET"]
            )
        finally:
            server.shutdown()
            thread.join()
            server.server_close()

    def test_zstd_artifact_preserves_original_signed_bytes(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            body = b"signed SSZ bytes\x00\xff" * 1024
            (root / "capture.ssz").write_bytes(body)
            result = artifacts.bundle(root, ["capture.ssz"], root / "evidence.tar.zst")
            self.assertTrue(result["passed"])
            self.assertEqual(result["uncompressed_bytes"], len(body))

    def test_engine_window_detects_unfinished_request_before_sealing_traffic(self):
        def record(sequence, stamp):
            return {
                "session": "real-session",
                "sequence": sequence,
                "started": stamp,
                "elapsed_ns": 1000,
                "complete": True,
                "http_status": 200,
                "request_body": base64.b64encode(
                    b'{"method":"engine_newPayloadV5"}'
                ).decode(),
                "response_body": base64.b64encode(
                    b'{"result":{"status":"VALID"}}'
                ).decode(),
            }

        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "engine.jsonl"
            first, last = record(1, "1970-01-01T00:00:01Z"), record(
                3, "1970-01-01T00:00:03Z"
            )
            path.write_text(json.dumps(first) + "\n" + json.dumps(last) + "\n")
            result = campaign.engine_window(path, 0, 2)
            self.assertEqual(result["sequence_gaps"], {"real-session": [2]})
            path.write_text(json.dumps(first) + "\n")
            self.assertEqual(
                campaign.engine_window(path, 0, 2)["unsealed_sessions"],
                ["real-session"],
            )

    def test_pressure_mutates_only_hash_and_keeps_original_request(self):
        original = {
            "method": "engine_newPayloadV5",
            "params": [
                {"blockHash": "0x" + "11" * 32, "stateRoot": "0x22"},
                [],
                "0x33",
                [],
            ],
        }
        before = json.dumps(original, sort_keys=True)
        altered = pressure.invalid_payload(original)
        self.assertEqual(json.dumps(original, sort_keys=True), before)
        self.assertNotEqual(
            altered["params"][0]["blockHash"], original["params"][0]["blockHash"]
        )
        altered["params"][0]["blockHash"] = original["params"][0]["blockHash"]
        self.assertEqual(altered, original)
        params = pressure.call_params("0x44", 22000, "0x5b600056")
        self.assertEqual(params[1], {"blockHash": "0x44", "requireCanonical": True})
        self.assertEqual(int(params[0]["gas"], 16), 22000)

    def test_private_arguments_are_excluded_from_publishable_inventory(self):
        command = [
            "run",
            "--wallet-privkey",
            "private-key-value",
            "--builder-privkey=another-secret",
            "--cl-client=http://beacon",
        ]
        public = campaign.public_command(command)
        self.assertNotIn("private-key-value", str(public))
        self.assertNotIn("another-secret", str(public))
        self.assertIn("--cl-client=http://beacon", public)
        shell = campaign.public_command(
            ['tool --builder-mnemonic="private words here" --api-port=8080']
        )
        self.assertNotIn("private words", str(shell))
        self.assertIn("--api-port=8080", shell[0])

    def test_percentiles_include_tail_and_runtime_identity_tracks_restart(self):
        self.assertEqual(campaign.percentile(list(range(1, 101)), 0.99), 99)
        node = {
            "container": "a",
            "image_id": "image",
            "command": [],
            "entrypoint": [],
            "memory_limit": 100,
            "nano_cpus": 2,
            "state": {"StartedAt": "original"},
        }
        before = campaign.stable_identity({"node": node})
        node["state"]["StartedAt"] = "restarted"
        self.assertNotEqual(before, campaign.stable_identity({"node": node}))

    def test_only_complete_wal_commit_can_change_expected_head(self):
        def put(value):
            return (
                b"\x01"
                + (2).to_bytes(4, "big")
                + b"hb"
                + len(value).to_bytes(4, "big")
                + value
            )

        old, new = b"a" * 32, b"b" * 32
        initial = "0x" + old.hex()
        prepared = put(new)
        self.assertEqual(lifecycle.committed_head(prepared, initial), initial)
        self.assertEqual(
            lifecycle.committed_head(prepared + b"\x03", initial), "0x" + new.hex()
        )
        self.assertEqual(
            lifecycle.committed_head(prepared + b"\x03\x01", initial), "0x" + new.hex()
        )

    def test_rpc_errors_and_wrong_identity_cannot_pass_as_a_sample(self):
        record = {"http_status": 200, "request": {"id": 1}}
        for response in [
            {"jsonrpc": "2.0", "id": 1, "error": {"code": -32000}},
            {"jsonrpc": "2.0", "id": 2, "result": "0x1"},
            {"id": 1, "result": "0x1"},
        ]:
            with self.subTest(response=response), self.assertRaises(RuntimeError):
                campaign.decoded({**record, "response": json.dumps(response)})

    def test_private_network_requires_both_chain_and_genesis(self):
        node = {"ports": {"8545/tcp": [{"HostPort": "1"}]}}
        for chain, genesis, accepted in [
            ("0x539", campaign.GENESIS, True),
            ("0x1", campaign.GENESIS, False),
            ("0x539", "0x" + "00" * 32, False),
        ]:
            replies = [{"result": chain}, {"result": {"hash": genesis}}]
            with patch.object(campaign, "exchange"), patch.object(
                campaign, "decoded", side_effect=replies
            ):
                if accepted:
                    campaign.assert_private_network({"el-1-test": node})
                else:
                    with self.assertRaises(RuntimeError):
                        campaign.assert_private_network({"el-1-test": node})

    def test_snapshot_never_overwrites_prior_attempt(self):
        with tempfile.TemporaryDirectory() as root:
            output = Path(root) / "attempt"
            output.mkdir()
            evidence = output / "inventory.json"
            evidence.write_text("original failed attempt")
            with self.assertRaises(FileExistsError):
                campaign.snapshot(output)
            self.assertEqual(evidence.read_text(), "original failed attempt")


if __name__ == "__main__":
    unittest.main()
