import copy
import importlib.util
from pathlib import Path
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("history_control", Path(__file__).with_name("run-frame-history-control.py"))
history = importlib.util.module_from_spec(spec)
spec.loader.exec_module(history)


class FrameHistoryTests(unittest.TestCase):
    def fixture(self):
        return {"version": "heze", "execution_optimistic": False, "finalized": True,
                "data": {"message": {"beacon_block_root": "beacon", "payload": {"block_hash": "execution", "transactions": ["0x06"]}}, "signature": "signature"}}

    def test_historical_read_requires_original_transaction_and_both_roots(self):
        with patch.object(history.verifier, "command", return_value="tx"):
            result = history.check_historical_envelope(self.fixture(), "execution", "tx", ["commitment"], "beacon")
        self.assertEqual(result["raw_transaction"], "0x06")
        for path, value in ((["execution_optimistic"], True), (["finalized"], False),
                            (["data", "message", "beacon_block_root"], "other"),
                            (["data", "message", "payload", "block_hash"], "other")):
            fixture = copy.deepcopy(self.fixture())
            cursor = fixture
            for key in path[:-1]:
                cursor = cursor[key]
            cursor[path[-1]] = value
            with patch.object(history.verifier, "command", return_value="tx"):
                with self.assertRaises(ValueError):
                    history.check_historical_envelope(fixture, "execution", "tx", [], "beacon")

    def test_trusted_checkpoint_alone_does_not_prove_frame_history(self):
        with patch.object(history.verifier, "command", return_value="different-tx"):
            with self.assertRaisesRegex(ValueError, "original Frame"):
                history.check_historical_envelope(self.fixture(), "execution", "tx", [], "beacon")


if __name__ == "__main__":
    unittest.main()
