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

    def test_blob_restart_requires_original_receipt_storage_and_balances(self):
        original = {"transaction": "tx", "receipt": {"blockNumber": "0x2"}, "balances": [10, 5]}
        def rpc(_url, method, *args):
            if method == "eth_getTransactionReceipt":
                return original["receipt"]
            if method == "eth_getStorageAt":
                return "0x0" if args[-1] == "0x1" else "0x1"
            return "0xa" if args[-1] == "0x1" else "0x5"
        with patch.object(history.control, "rpc", side_effect=rpc):
            self.assertEqual(history.blob_readback("rpc", original)["storage"], [0, 1])
        for changed_method in ("eth_getTransactionReceipt", "eth_getStorageAt", "eth_getBalance"):
            def wrong(url, method, *args):
                if method == changed_method:
                    return {"blockNumber": "0x3"} if method == "eth_getTransactionReceipt" else "0xff"
                return rpc(url, method, *args)
            with patch.object(history.control, "rpc", side_effect=wrong):
                with self.assertRaises(RuntimeError):
                    history.blob_readback("rpc", original)


if __name__ == "__main__":
    unittest.main()
