import unittest

from qualify_fast_confirmation import profile


class FastConfirmationProfileTest(unittest.TestCase):
    def test_real_cl_authority_and_native_consumers_are_enabled(self):
        result = profile("native:qualified", "lighthouse:qualified")
        participant = result["participants"][0]
        self.assertEqual(participant["count"], 2)
        self.assertIn("--enable-fast-confirmation", participant["cl_extra_params"])
        self.assertTrue(any("zkvm" in value for value in participant["el_extra_params"]))
        self.assertEqual(result["network_params"]["seconds_per_slot"], 6)
        self.assertEqual(result["network_params"]["gloas_fork_epoch"], 0)
        self.assertEqual(result["network_params"]["num_validator_keys_per_node"], 128)
        self.assertIn("0x0000bff46984e3725691fa540a8c7589300d8282", result["network_params"]["additional_preloaded_contracts"])
        self.assertEqual(result["additional_services"], [])


if __name__ == "__main__":
    unittest.main()
