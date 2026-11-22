import copy
import json
import subprocess
import sys
import unittest
from datetime import datetime, timedelta, timezone
from pathlib import Path

import glamsterdam_soak_audit as audit


START = datetime(2026, 11, 22, tzinfo=timezone.utc)


def at(seconds):
    return (START + timedelta(seconds=seconds)).isoformat()


def root(number):
    return "0x" + format(number, "064x")


def point(index):
    memory = {"memory_max": audit.MEMORY, "process_hwm": 1000000000,
              "events": {"oom": 0, "oom_kill": 0}}
    return {"ready": True, "sync": {"is_syncing": False, "is_optimistic": False,
            "el_offline": False, "head_slot": str(500 + index)},
            "memory": {"el": copy.deepcopy(memory), "cl": copy.deepcopy(memory)},
            "finalized_number": 100 + index, "finalized_hash": root(100 + index),
            "finalized_root": root(200 + index), "reference_lag": 0}


def operating(count=3):
    samples = [{**point(i), "elapsed_seconds": i * 30, "at": at(i * 30)}
               for i in range(1, count + 1)]
    state = {"status": "running", "passed": False, "profile": {
        "duration_seconds": audit.DURATION, "sampling_seconds": 30,
        "source": audit.SOURCE, "binary_sha256": audit.EXECUTABLES["el"],
        "cl_source": audit.CL_SOURCE,
        "cl_binary_sha256": audit.EXECUTABLES["cl"], "memory_each_bytes": audit.MEMORY,
        "cpus_each": 2}, "samples": count, "initial": point(0),
        "accepted_started_at": at(0), "elapsed_seconds": count * 30, "last": point(count)}
    return state, samples


def headers(last, fork_at=None):
    chain, parent = [], root(0)
    for slot in range(last + 1):
        body = 4 if fork_at is not None and slot >= fork_at else 3
        message = {"slot": str(slot), "proposer_index": "1", "parent_root": parent,
                   "state_root": root(2), "body_root": root(body)}
        parent = audit.beacon_header_root(message)
        chain.append((parent, message))
    return chain


def window(chain, first, stop, anchor):
    def record(origin, index):
        value, message = chain[index]
        return {"url": origin + "/eth/v1/beacon/headers/" + value, "http_status": 200,
                "response": json.dumps({"data": {"root": value, "canonical": True,
                                                   "header": {"message": message}}})}
    trace = [{"url": "http://main/eth/v1/config/spec"}]
    trace += [record("http://main", index) for index in range(anchor - 1, first - 2, -1)]
    trace += [record("http://second", anchor), record("http://third", anchor)]
    report = {"beacon_window": {"first": first, "stop": stop, "slots": stop - first,
              "finalized_root": chain[anchor][0], "finalized_slot": anchor,
              "parent_hash_links": anchor - first + 1, "missing_slots": []}}
    return trace, report


class SamplingTests(unittest.TestCase):
    def test_live_prefix_cannot_be_called_48_hour_completion(self):
        state, samples = operating()
        self.assertEqual(audit.check_samples(state, samples, now=START.timestamp() + 90)["samples"], 3)
        state.update(passed=True, finished_at=at(100))
        with self.assertRaisesRegex(audit.EvidenceError, "48-hour"):
            audit.check_samples(state, samples, complete=True)

    def test_cl_source_declaration_cannot_disagree_with_frozen_binary(self):
        for source in (None, "other-source"):
            state, samples = operating()
            state["profile"]["cl_source"] = source
            with self.assertRaisesRegex(audit.EvidenceError, "runtime source changed"):
                audit.check_samples(state, samples)

    def test_complete_real_duration_and_all_samples(self):
        state, samples = operating(5760)
        state.update(passed=True, finished_at=at(audit.DURATION + 1))
        result = audit.check_samples(state, samples, complete=True)
        self.assertEqual(result["observed_seconds"], audit.DURATION)

    def test_suspended_sampler_cannot_fill_elapsed_duration(self):
        state, samples = operating()
        samples[-1].update(elapsed_seconds=172800, at=at(172800))
        state.update(elapsed_seconds=172800, passed=True, finished_at=at(172801))
        with self.assertRaisesRegex(audit.EvidenceError, "delayed sample"):
            audit.check_samples(state, samples, complete=True)

    def test_truncation_and_stale_heartbeat_reject(self):
        state, samples = operating()
        with self.assertRaisesRegex(audit.EvidenceError, "sample count"):
            audit.check_samples(state, samples[:-1])
        with self.assertRaisesRegex(audit.EvidenceError, "stale"):
            audit.check_samples(state, samples, now=START.timestamp() + 200)

    def test_initial_optimism_missing_cl_memory_oom_and_regression_reject(self):
        for mutation in ("initial", "memory", "oom", "regression"):
            with self.subTest(mutation=mutation):
                state, samples = operating()
                if mutation == "initial":
                    state["initial"]["sync"]["is_optimistic"] = True
                elif mutation == "memory":
                    del samples[1]["memory"]["cl"]
                elif mutation == "oom":
                    samples[1]["memory"]["cl"]["events"]["oom_kill"] = 1
                else:
                    samples[1]["finalized_number"] = 1
                with self.assertRaises(audit.EvidenceError):
                    audit.check_samples(state, samples)

    def test_optimized_python_still_rejects_incomplete_duration(self):
        command = [sys.executable, "-O", "-c",
                   "import glamsterdam_soak_audit as a; a.require(False, 'reject')"]
        result = subprocess.run(command, cwd=Path(__file__).parent, capture_output=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn(b"EvidenceError", result.stderr)


class FinalityTests(unittest.TestCase):
    def test_initial_signed_warmup_is_pending_not_completion(self):
        audit.check_coverage({"status": "waiting", "windows": []}, [], 500, 500, 510)
        with self.assertRaisesRegex(audit.EvidenceError, "no signed finality"):
            audit.check_coverage({"status": "waiting", "windows": []}, [], 500, 500, 510, True)

    def test_ssz_header_root_matches_unchanged_reference_vector(self):
        # Generated independently with the pinned consensus spec; not derived
        # from the implementation under test at test runtime.
        message = {"slot": "1", "proposer_index": "2", "parent_root": "0x" + "11" * 32,
                   "state_root": "0x" + "22" * 32, "body_root": "0x" + "33" * 32}
        self.assertEqual(audit.beacon_header_root(message),
                         "0xca97916da2119fd20a6e873e4c8d77d4f92297cf3b82d017d277a9a46d10de61")

    def test_complete_contiguous_parent_path_and_anchor_overlap(self):
        chain = headers(10)
        trace, report = window(chain, 2, 5, 6)
        anchor = audit.linked_window(trace, report)
        later, later_report = window(chain, 5, 9, 10)
        self.assertEqual(audit.linked_window(later, later_report, anchor), chain[10][0])

    def test_matching_slot_boundary_does_not_excuse_conflicting_finality(self):
        original, fork = headers(10), headers(10, fork_at=5)
        old, report = window(original, 2, 5, 6)
        anchor = audit.linked_window(old, report)
        later, later_report = window(fork, 5, 9, 10)
        # Both paths include identical slot4, and the interval bounds touch.
        # The later path nevertheless conflicts with the already finalized6.
        self.assertEqual(original[4][0], fork[4][0])
        audit.linked_window(later, later_report)
        with self.assertRaisesRegex(audit.EvidenceError, "previous finalized anchor"):
            audit.linked_window(later, later_report, anchor)

    def test_modified_header_bytes_and_missing_parent_reject(self):
        chain = headers(10)
        trace, report = window(chain, 2, 5, 6)
        altered = copy.deepcopy(trace)
        body = json.loads(altered[1]["response"])
        body["data"]["header"]["message"]["body_root"] = root(123)
        altered[1]["response"] = json.dumps(body)
        with self.assertRaisesRegex(audit.EvidenceError, "root or canonical"):
            audit.linked_window(altered, report)
        del trace[2]
        report["beacon_window"]["parent_hash_links"] -= 1
        with self.assertRaisesRegex(audit.EvidenceError, "disconnected"):
            audit.linked_window(trace, report)


class RuntimeTests(unittest.TestCase):
    def setUp(self):
        self.infos, self.baseline = {}, {}
        for role in ("el", "cl"):
            self.baseline[role] = {"image": role, "started": at(0),
                                   "binary_sha256": audit.EXECUTABLES[role],
                                   "source": {"el": audit.SOURCE, "cl": audit.CL_SOURCE}[role]}
            self.infos[role] = {"Image": role, "RestartCount": 0,
                "Config": {"Labels": {"eth2030.qualification": audit.LABEL}},
                "HostConfig": {"Memory": audit.MEMORY, "NanoCpus": 2000000000},
                "State": {"StartedAt": at(0), "Running": True, "ExitCode": 0, "OOMKilled": False}}

    def test_running_owner_then_clean_terminal_owner(self):
        audit.check_runtime(self.infos, self.baseline)
        for value in self.infos.values():
            value["State"]["Running"] = False
        audit.check_runtime(self.infos, self.baseline, complete=True)

    def test_baseline_source_declaration_is_required_for_both_clients(self):
        for role in ("el", "cl"):
            with self.subTest(role=role):
                original = self.baseline[role].pop("source")
                with self.assertRaisesRegex(audit.EvidenceError, "source declaration"):
                    audit.check_runtime(self.infos, self.baseline)
                self.baseline[role]["source"] = original

    def test_passed_time_cannot_hide_failed_cleanup(self):
        with self.assertRaisesRegex(audit.EvidenceError, "cleanup failed"):
            audit.check_runtime(self.infos, self.baseline, complete=True)
        for value in self.infos.values():
            value["State"].update(Running=False, ExitCode=137)
        with self.assertRaisesRegex(audit.EvidenceError, "cleanup failed"):
            audit.check_runtime(self.infos, self.baseline, complete=True)

    def test_identity_change_or_unowned_container_rejects(self):
        self.infos["el"]["State"]["StartedAt"] = at(1)
        with self.assertRaisesRegex(audit.EvidenceError, "identity changed"):
            audit.check_runtime(self.infos, self.baseline)
        self.infos["el"]["State"]["StartedAt"] = at(0)
        self.infos["cl"]["Config"]["Labels"].clear()
        with self.assertRaisesRegex(audit.EvidenceError, "ownership"):
            audit.check_runtime(self.infos, self.baseline)


class WireTests(unittest.TestCase):
    def exchanges(self, state, samples):
        records = []
        for index, sample in enumerate([state["initial"], *samples]):
            end = index * 30 - 0.01
            header = {"number": hex(sample["finalized_number"]), "hash": sample["finalized_hash"],
                      "stateRoot": sample["finalized_root"]}
            records += [{"at": at(end - 0.3), "url": "http://cl/eth/v1/node/syncing", "method": None,
                         "params": None, "response": {"data": sample["sync"]}}]
            for offset, origin, params in [(0.2, "http://el", ["finalized", False]),
                                           (0.1, "http://geth", [header["number"], False]),
                                           (0, "http://geth", ["finalized", False])]:
                records.append({"at": at(end - offset), "url": origin,
                                "method": "eth_getBlockByNumber", "params": params,
                                "response": {"result": copy.deepcopy(header)}})
        return records

    def test_actual_matching_exchange_per_sample(self):
        state, samples = operating()
        records = self.exchanges(state, samples)
        self.assertEqual(len(audit.check_wire(records, state, samples, "http://geth")), 4)

    def test_metadata_passed_cannot_hide_wrong_geth_bytes(self):
        state, samples = operating()
        records = self.exchanges(state, samples)
        records[6]["response"]["result"]["stateRoot"] = root(999)
        with self.assertRaisesRegex(audit.EvidenceError, "actual EL/Geth"):
            audit.check_wire(records, state, samples, "http://geth")

    def test_missing_wire_group_cannot_be_replaced_by_previous_equal_hash(self):
        state, samples = operating()
        records = self.exchanges(state, samples)
        del records[4:8]
        with self.assertRaisesRegex(audit.EvidenceError, "unique actual wire"):
            audit.check_wire(records, state, samples, "http://geth")


if __name__ == "__main__":
    unittest.main()
