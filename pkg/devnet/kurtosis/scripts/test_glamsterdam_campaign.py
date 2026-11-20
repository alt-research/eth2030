import json
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

import glamsterdam_campaign as campaign
import glamsterdam_process_lifecycle as lifecycle


class CampaignTests(unittest.TestCase):
    def test_percentiles_include_tail_and_runtime_identity_tracks_restart(self):
        self.assertEqual(campaign.percentile(list(range(1, 101)), .99), 99)
        node = {"container": "a", "image_id": "image", "command": [], "entrypoint": [],
                "memory_limit": 100, "nano_cpus": 2, "state": {"StartedAt": "original"}}
        before = campaign.stable_identity({"node": node})
        node["state"]["StartedAt"] = "restarted"
        self.assertNotEqual(before, campaign.stable_identity({"node": node}))

    def test_only_complete_wal_commit_can_change_expected_head(self):
        def put(value):
            return b"\x01" + (2).to_bytes(4,"big") + b"hb" + len(value).to_bytes(4,"big") + value
        old, new = b"a"*32, b"b"*32
        initial = "0x"+old.hex()
        prepared = put(new)
        self.assertEqual(lifecycle.committed_head(prepared, initial), initial)
        self.assertEqual(lifecycle.committed_head(prepared+b"\x03", initial), "0x"+new.hex())
        self.assertEqual(lifecycle.committed_head(prepared+b"\x03\x01", initial), "0x"+new.hex())

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
            with patch.object(campaign, "exchange"), patch.object(campaign, "decoded", side_effect=replies):
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
