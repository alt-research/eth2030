import unittest

from qualify_fast_confirmation import profile, restarted_endpoint
from unittest.mock import patch
import json


class FastConfirmationProfileTest(unittest.TestCase):
    def test_crash_restart_uses_actual_docker_mapping(self):
        current = [{"NetworkSettings": {"Ports": {"4000/tcp": [{"HostIp": "127.0.0.1", "HostPort": "34567"}]}}}]
        with patch("qualify_fast_confirmation.run", return_value=json.dumps(current)) as inspect:
            self.assertEqual(restarted_endpoint("owned-container"), "http://127.0.0.1:34567")
            inspect.assert_called_once_with("docker", "inspect", "owned-container")

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
