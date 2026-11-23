import copy
from datetime import datetime, timezone
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from review_diagnostic import awaiting_final_bls, capture, main, read_published_json, review, review_rolling


START = 1800000000


def iso(value):
    return datetime.fromtimestamp(value, timezone.utc).isoformat()


def fixture(seconds=120):
    rows = []
    for elapsed in range(0, seconds + 1, 30):
        rows.append({
            "at": iso(START + elapsed), "ready": True,
            "sync": {"is_syncing": False, "is_optimistic": False,
                     "el_offline": False, "head_slot": str(100 + elapsed)},
            "finalized_number": 1 + elapsed // 30, "reference_lag": 0,
            "memory": {role: {"memory_max": 4096000000,
                              "process_hwm": 1000000,
                              "events": {"oom": 0, "oom_kill": 0}}
                       for role in ("el", "cl")},
        })
    state = {"started_at": iso(START), "status": "observing",
             "samples": len(rows), "last": {k: v for k, v in rows[-1].items() if k != "at"}}
    return state, rows


class ReviewTest(unittest.TestCase):
    def check_failed(self, state, rows, bls=None, now=None):
        result = review(state, rows, bls or {}, now or START + 120)
        self.assertEqual(result["status"], "failed", result)
        self.assertFalse(result["original_fault_resolved"])
        return result

    def test_legacy_success_does_not_hide_lost_readiness(self):
        for key in ("is_syncing", "is_optimistic", "el_offline"):
            with self.subTest(key=key):
                state, rows = fixture()
                state.update(status="bounded_diagnostic_finished_without_stall", finished_at=iso(START + 121))
                rows[-1]["sync"][key] = True
                self.assertIn("lost readiness", self.check_failed(state, rows)["errors"][0])

    def test_finalized_stall_is_not_a_no_stall_pass(self):
        state, rows = fixture(600)
        for row in rows:
            row["finalized_number"] = 1
        self.assertIn("execution stalled", self.check_failed(state, rows, now=START + 600)["errors"][0])

    def test_ledger_gaps_staleness_and_duplicates_fail(self):
        state, rows = fixture(300)
        for candidate in (rows[:1] + rows[5:], rows + [copy.deepcopy(rows[-1])]):
            bad = copy.deepcopy(state)
            bad["samples"] = len(candidate)
            self.check_failed(bad, candidate, now=START + 300)
        state, rows = fixture()
        self.check_failed(state, rows, now=START + 211)
        state, rows = fixture(30)
        rows[-1]["at"] = iso(START + 61)
        state["last"] = {k: v for k, v in rows[-1].items() if k != "at"}
        self.assertIn("excessively delayed", self.check_failed(state, rows, now=START + 61)["errors"][0])

    def test_five_and_a_half_hours_start_at_readiness_not_process_start(self):
        state, rows = fixture(19800)
        rows[0]["ready"] = False
        result = review(state, rows, {}, START + 19800)
        self.assertEqual(result["status"], "observing_before_5_5h")
        self.assertFalse(result["ready_5_5h_covered"])
        state, rows = fixture(19830)
        rows[0]["ready"] = False
        result = review(state, rows, {}, START + 19830)
        self.assertTrue(result["ready_5_5h_covered"])
        self.assertEqual(result["post_threshold_samples"], 1)
        self.assertFalse(result["original_fault_resolved"])

    def test_threshold_capture_remains_due_when_readiness_failed(self):
        state, rows = fixture()
        rows[-1]["ready"] = False
        result = self.check_failed(state, rows, now=START + 20000)
        self.assertTrue(result["threshold_time_reached"])
        self.assertIn("lost readiness", result["errors"][0])

    def test_bls_authentication_is_separate_from_operating_quality(self):
        state, rows = fixture()
        window = {"first": 100, "through": 220, "passed": True,
                  "beacon_window": {"first": 100, "stop": 221,
                                    "signature_participants": 512,
                                    "parent_hash_links": 125,
                                    "missing_slots": [101, 102]}}
        result = review(state, rows, {"windows": [window]}, START + 120)
        self.assertTrue(result["bls_window_reports_cover_observed_slots"])
        self.assertEqual(result["reported_missing_slots"], [101, 102])
        self.assertFalse(result["bls_signatures_verified_by_this_tool"])
        bad = copy.deepcopy(window)
        bad["first"] = 101
        self.check_failed(state, rows, {"windows": [bad]})
        bad = copy.deepcopy(window)
        bad["beacon_window"]["signature_participants"] = 0
        self.check_failed(state, rows, {"windows": [bad]})
        self.check_failed(state, rows, {"status": "verification_failed"})

    def test_incomplete_window_and_early_exit_cannot_be_complete(self):
        state, rows = fixture()
        result = review(state, rows, {"windows": [{"passed": False}]}, START + 120)
        self.assertTrue(result["bls_window_pending"])
        self.assertFalse(result["bls_window_reports_cover_observed_slots"])
        state.update(status="bounded_diagnostic_finished_without_stall", finished_at=iso(START + 120))
        self.assertIn("before the declared", self.check_failed(state, rows)["errors"][0])

    def test_passive_capture_rejects_changed_identity_without_touching_runtime(self):
        calls = []
        def read_only(*args):
            calls.append(args)
            if args[0] == "inspect":
                return json.dumps([{"State": {"StartedAt": "old", "Running": True},
                                    "RestartCount": 1}])
            raise AssertionError("restart must be rejected before another operation")
        with tempfile.TemporaryDirectory() as tmp, patch("review_diagnostic.docker", read_only):
            record = {"el": "owned-el", "cl": "owned-cl", "el_started": "old", "cl_started": "old"}
            receipt, name = capture(Path(tmp), Path(tmp), record)
            self.assertEqual(len(receipt["errors"]), 2)
            self.assertNotIn("stack_sha256", receipt)
            self.assertTrue((Path(tmp) / name / "receipt.json").is_file())
            self.assertTrue(all(c[0] == "inspect" for c in calls))

    def test_pressure_is_retained_without_treating_no_oom_as_no_pressure(self):
        state, rows = fixture()
        for row in rows:
            row["memory"]["el"]["events"]["max"] = 686
        state["last"] = {k: v for k, v in rows[-1].items() if k != "at"}
        result = review(state, rows, {}, START + 120)
        self.assertEqual(result["status"], "observing_before_5_5h")
        self.assertEqual(result["memory_limit_events"]["el"], 686)

    def test_zero_wait_events_cannot_hide_stale_or_mismatched_collector(self):
        profile = {"source": "pinned", "binary_sha256": "binary"}
        rolling = {**profile, "status": "watching", "updated_at": iso(START), "samples": 10, "events": 0}
        with self.assertRaisesRegex(ValueError, "stale"):
            review_rolling(rolling, profile, START + 61, False)
        with self.assertRaisesRegex(ValueError, "stale"):
            review_rolling(rolling, profile, START + 61, True)
        rolling["source"] = "other"
        with self.assertRaisesRegex(ValueError, "identity"):
            review_rolling(rolling, profile, START + 1, False)

    def test_sustained_wait_requires_review_without_claiming_a_root(self):
        profile = {"source": "pinned", "binary_sha256": "binary"}
        rolling = {**profile, "status": "watching", "updated_at": iso(START), "samples": 10, "events": 1}
        result = review_rolling(rolling, profile, START + 1, False)
        self.assertTrue(result["wait_chain_review_required"])
        self.assertIn("request/log correlation", result["scope"])

    def test_partial_publication_retries_but_persistent_corruption_fails(self):
        with patch.object(Path, "read_text", side_effect=["{", '{"status":"verification_failed"}']), patch("review_diagnostic.time.sleep"):
            value = read_published_json(Path("summary.json"))
            self.assertEqual(value["status"], "verification_failed")
        with patch.object(Path, "read_text", side_effect=["{", "{", "{"]), patch("review_diagnostic.time.sleep"):
            with self.assertRaises(json.JSONDecodeError):
                read_published_json(Path("summary.json"))

    def test_finished_runtime_waits_for_last_bls_window_before_final_review(self):
        with tempfile.TemporaryDirectory() as tmp:
            run, out = Path(tmp) / "run", Path(tmp) / "out"
            (run / "observations").mkdir(parents=True)
            (run / "containers.json").write_text(json.dumps({"el": "el", "cl": "cl"}))
            (run / "observations/identities.json").write_text(json.dumps({r: {"started": iso(START)} for r in ("el", "cl")}))
            state = {"finished_at": iso(START)}
            pending = {"errors": [], "reported_bls_windows": 13, "bls_controller_status": "verifying_window"}
            complete = {"errors": [], "reported_bls_windows": 14, "bls_controller_status": "observed_windows_complete_stopped_heads_pending", "bls_window_reports_cover_observed_slots": True}
            args = ["review_diagnostic.py", "--run", str(run), "--output", str(out), "--watch"]
            with patch("sys.argv", args), patch("review_diagnostic.load_review", side_effect=[(pending, state), (complete, state)]) as load, patch("review_diagnostic.time.sleep"), patch("review_diagnostic.time.time", return_value=START + 1):
                self.assertEqual(main(), 0)
            self.assertEqual(load.call_count, 2, "runtime completion must not discard the pending closing window")
            self.assertEqual(json.loads((out / "result.json").read_text())["reported_bls_windows"], 14)

    def test_closing_worker_failure_incomplete_coverage_and_timeout_are_not_success(self):
        state = {"finished_at": iso(START)}
        for status, now in [("verification_failed", START + 1),
                            ("observed_windows_complete_stopped_heads_pending", START + 1),
                            ("verifying_window", START + 3600)]:
            with self.subTest(status=status):
                result = {"errors": [], "bls_controller_status": status}
                self.assertFalse(awaiting_final_bls(result, state, now))
                self.assertEqual(result["status"], "failed")
                self.assertTrue(result["errors"])


if __name__ == "__main__":
    unittest.main()
