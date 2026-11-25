import copy
import importlib.util
from pathlib import Path
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("frame_focil", Path(__file__).with_name("verify-heze-focil.py"))
verifier = importlib.util.module_from_spec(spec)
spec.loader.exec_module(verifier)


class FrameControlTests(unittest.TestCase):
    def fixture(self):
        receipt = {"type": "0x6", "status": "0x1", "payer": "payer", "gasUsed": "0x64", "blockNumber": "0x2",
                   "frameResults": [{"status": "0x1"}, {"status": "0x1"}, {"status": "0x1", "stateGasUsed": "0x17e80"}]}
        expected = 100 * (1 + 1000000000)
        answers = [receipt, {"baseFeePerGas": "0x1", "stateRoot": "root"}, hex(10), hex(10),
                   hex(10**15), hex(10**15 - expected), "0x0", "0x1"]
        return receipt, answers

    def test_readback_checks_real_sponsor_charge_and_historical_storage(self):
        _, answers = self.fixture()
        with patch.object(verifier, "rpc", side_effect=answers), patch.object(verifier, "command", side_effect=["sender", "payer"]):
            result = verifier.frame_readback("execution", "tx")
        self.assertEqual(result["payer_charge"], "100000000100")
        self.assertEqual(result["storage"], ["0x0", "0x1"])

    def test_success_status_does_not_hide_a_failed_frame(self):
        receipt, _ = self.fixture()
        receipt["frameResults"][2]["status"] = "0x0"
        with patch.object(verifier, "rpc", return_value=receipt):
            with self.assertRaisesRegex(ValueError, "three frames"):
                verifier.frame_readback("execution", "tx")

    def test_paid_receipt_does_not_hide_wrong_balance_or_state(self):
        for index, value, message in ((5, "0x0", "charge"), (7, "0x0", "historical")):
            _, answers = self.fixture()
            answers = copy.deepcopy(answers)
            answers[index] = value
            with patch.object(verifier, "rpc", side_effect=answers), patch.object(verifier, "command", side_effect=["sender", "payer"]):
                with self.assertRaisesRegex(ValueError, message):
                    verifier.frame_readback("execution", "tx")


if __name__ == "__main__":
    unittest.main()
