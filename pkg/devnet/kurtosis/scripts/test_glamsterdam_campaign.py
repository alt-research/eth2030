import json
import base64
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

import glamsterdam_campaign as campaign
import glamsterdam_process_lifecycle as lifecycle
import glamsterdam_pressure as pressure
import glamsterdam_artifacts as artifacts


class CampaignTests(unittest.TestCase):
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
