import unittest

from qualify_fast_confirmation import profile, restarted_endpoint, archive_signed_block
from unittest.mock import patch
import json
import gzip
import tempfile
import urllib.error
from pathlib import Path


class FastConfirmationProfileTest(unittest.TestCase):
    def test_crash_restart_uses_actual_docker_mapping(self):
        current = [{"NetworkSettings": {"Ports": {"4000/tcp": [{"HostIp": "127.0.0.1", "HostPort": "34567"}]}}}]
        with patch("qualify_fast_confirmation.run", return_value=json.dumps(current)) as inspect:
            self.assertEqual(restarted_endpoint("owned-container"), "http://127.0.0.1:34567")
            inspect.assert_called_once_with("docker", "inspect", "owned-container")

    def test_signed_window_retains_block_state_and_envelope(self):
        values = [{"data": {"message": {"state_root": "0xstate"}}}, {"data": {"root": "0xblock"}}, {"data": {"slot": "7"}}, {"data": {"message": {"beacon_block_root": "0xblock"}}}]
        with tempfile.TemporaryDirectory() as directory, patch("qualify_fast_confirmation.get", side_effect=values):
            root = Path(directory)
            archive_signed_block(root, "http://cl", 7)
            self.assertEqual(len(list(root.iterdir())), 3)
            self.assertEqual(json.loads(gzip.decompress((root / "state-7.json.gz").read_bytes())), {"slot": "7"})
            self.assertEqual(json.loads(gzip.decompress((root / "block-7.json.gz").read_bytes())), values[0])

    def test_unavailable_envelope_does_not_record_a_partial_archive(self):
        values = [{"data": {"message": {"state_root": "0xstate"}}}, {"data": {"root": "0xblock"}}, {"data": {"slot": "7"}}, urllib.error.HTTPError("http://cl/envelope", 404, "pending", {}, None)]
        with tempfile.TemporaryDirectory() as directory, patch("qualify_fast_confirmation.get", side_effect=values):
            root = Path(directory)
            with self.assertRaises(urllib.error.HTTPError):
                archive_signed_block(root, "http://cl", 7)
            self.assertEqual(list(root.iterdir()), [])

    def test_real_cl_authority_and_native_consumers_are_enabled(self):
        result = profile("native:qualified", "lighthouse:qualified")
        participant = result["participants"][0]
        self.assertEqual(participant["count"], 2)
        self.assertIn("--enable-fast-confirmation", participant["cl_extra_params"])
        self.assertIn("--http.api=admin,eth,net,web3,zkvm", participant["el_extra_params"])
        self.assertIn("--ws.api=eth,net,web3,zkvm", participant["el_extra_params"])
        self.assertIn("--rpc.allow-insecure", participant["el_extra_params"])
        self.assertGreaterEqual(participant["el_max_mem"], 8192)
        self.assertEqual(result["network_params"]["seconds_per_slot"], 6)
        self.assertEqual(result["network_params"]["gloas_fork_epoch"], 0)
        self.assertEqual(result["network_params"]["num_validator_keys_per_node"], 128)
        self.assertIn("0x0000bff46984e3725691fa540a8c7589300d8282", result["network_params"]["additional_preloaded_contracts"])
        self.assertEqual(result["additional_services"], [])


if __name__ == "__main__":
    unittest.main()
