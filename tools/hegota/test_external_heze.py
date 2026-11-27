import importlib.util
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("external_heze", Path(__file__).with_name("run_external_heze.py"))
external = importlib.util.module_from_spec(spec)
spec.loader.exec_module(external)


class ExternalHezeTests(unittest.TestCase):
    def test_peer_diagnostics_survive_one_unavailable_service(self):
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory)
            with patch.object(external.base, "run", side_effect=[
                    subprocess.TimeoutExpired("kurtosis", 20), "actual CL fork-choice diagnostic"]) as run:
                errors = external.retain_service_logs("owned", ["el-1", "cl-1"], output)
            self.assertEqual(list(errors), ["cl-1"])
            self.assertIn("TimeoutExpired", errors["cl-1"])
            self.assertEqual((output / "el-1.log").read_text(), "actual CL fork-choice diagnostic\n")
            self.assertEqual(run.call_count, 2)
            for call in run.call_args_list:
                self.assertEqual(call.args[:5], ("kurtosis", "service", "logs", "--all", "owned"))
                self.assertEqual(call.kwargs["timeout"], 20)

    def test_private_profile_preserves_node_defaults_and_engine_authentication(self):
        profile = external.private_profile()
        self.assertEqual(profile.count('"--rpc.allow-insecure"'), 2)
        self.assertNotIn("--authrpc.jwtsecret", profile)
        self.assertNotIn("--engine.auth", profile)
        self.assertIn("el_image: eth2030:frame-control", profile)
        original = (external.SCRIPTS.parent / "configs/frame-heze-control.yaml").read_text()
        self.assertNotIn("--rpc.allow-insecure", original)

    def test_unsatisfied_root_requires_independent_execution_validity(self):
        node = {"root": "beacon", "execution_status": {"Valid": "execution"}}
        snapshot = {"data": {"nodes": [{"V29": node}], "unsatisfied_inclusion_lists": ["beacon"]}}
        self.assertEqual(external.unsatisfied_valid_root(snapshot, "execution"), "beacon")
        for state in [{"Invalid": "execution"}, {"Optimistic": "execution"}, {"Valid": "other"}]:
            candidate = json.loads(json.dumps(snapshot))
            candidate["data"]["nodes"][0]["V29"]["execution_status"] = state
            self.assertIsNone(external.unsatisfied_valid_root(candidate, "execution"))
        snapshot["data"]["unsatisfied_inclusion_lists"] = []
        self.assertIsNone(external.unsatisfied_valid_root(snapshot, "execution"))
        self.assertEqual(external.valid_root(snapshot, "execution", True), "beacon")

    def test_only_actual_native_valid_false_with_the_fixture_counts(self):
        record = {"method": "engine_newPayloadV6", "block_hash": "payload", "inclusion_list": ["0x01"], "result": {"status": "VALID", "inclusionListSatisfied": False}}
        self.assertEqual(external.native_omissions([record], "0x01"), [record])
        self.assertEqual(external.native_omissions([record], "0x01", "payload"), [record])
        self.assertEqual(external.native_omissions([record], "0x01", "other"), [])
        for status in ("INVALID", "SYNCING", "ACCEPTED"):
            changed = {**record, "result": {"status": status, "inclusionListSatisfied": False}}
            self.assertEqual(external.native_omissions([changed], "0x01"), [])
        self.assertEqual(external.native_omissions([{**record, "result": None}], "0x01"), [])
        self.assertEqual(external.native_omissions([record], "0x02"), [])
        satisfied = {**record, "result": {"status": "VALID", "inclusionListSatisfied": True}}
        self.assertEqual(external.native_verdicts([satisfied], "0x01", True, "payload"), [satisfied])
        self.assertEqual(external.native_omissions([satisfied], "0x01"), [])

    def observations(self):
        events, observations = [[], []], []
        for origin in (0, 1):
            for slot in range(1, 17):
                start = slot * 1000000000
                events[origin].append({"test_il_injected": True, "transactions": ["0xab"], "unix_ns": start, "end_ns": start})
                signed = {"message": {"slot": str(slot), "dependent_root": "root", "validator_index": str(128 * origin), "transactions": ["0xab"]}, "signature": "signature"}
                observations.extend([(origin, start + 1000000, signed), (1-origin, start + 2000000, signed)])
        return observations, events

    def test_bidirectional_measurements_require_exact_signed_peer_messages(self):
        observations, events = self.observations()
        result = external.network_measurements(observations, events, ["0xab"], 0, 1)
        self.assertEqual(result["samples"], 32)
        self.assertEqual(result["producer_directions"], [16, 16])
        self.assertEqual(result["latency_ms"]["p99"], 2)
        self.assertEqual(result["signed_ssz_bytes"], 157)
        self.assertFalse(result["public_network_tail_qualified"])
        delayed = [(peer, at + 9000000, signed) for peer, at, signed in observations]
        repeated = external.network_measurements(observations + delayed, events, ["0xab"], 0, 1)
        self.assertEqual(repeated["latency_ms"], result["latency_ms"])
        with self.assertRaisesRegex(ValueError, "did not reach"):
            external.network_measurements(observations[:-1], events, ["0xab"], 0, 1)
        changed = json.loads(json.dumps(observations))
        changed[-1][2]["signature"] = "different"
        with self.assertRaisesRegex(ValueError, "differ"):
            external.network_measurements(changed, events, ["0xab"], 0, 1)

    def test_missing_sources_or_insufficient_samples_cannot_pass(self):
        observations, events = self.observations()
        for records in ([[], []], [events[0], []]):
            with self.assertRaisesRegex(ValueError, "insufficient"):
                external.network_measurements(observations, records, ["0xab"], 0, 1)
        self.assertEqual(external.percentile([3, 1, 2], .95), 3)


if __name__ == "__main__":
    unittest.main()
