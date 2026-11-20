import json
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

import glamsterdam_campaign as campaign


class CampaignTests(unittest.TestCase):
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
