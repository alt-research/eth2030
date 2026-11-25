import hashlib
import json
import unittest
from acceptance import COMMANDS, validate_acceptance, validate_reference, validate_skips, validate_inventory, validate_run, validate_baseline
from check import ROOT


class AcceptanceEvidence(unittest.TestCase):
    def setUp(self):
        load = lambda p: json.loads((ROOT / p).read_text())
        self.register = load('specs/hegota/tasks.json')
        self.matrix = load('specs/hegota/acceptance.json')
        self.runtime = {'source_fingerprint_sha256': 'a' * 64}
        self.evidence = {'acceptance': {'source_fingerprint_sha256': 'a' * 64, 'passed_tests': {}},
                         'full_suite': {'exit_code': 0, 'named_test_or_subtest_events': {'pass': 100, 'fail': 0, 'skip': 0},
                                        'package_events': {'pass': 1, 'fail': 0, 'skip': 0},
                                        'started_packages': 1, 'completed_packages': 1}, 'artifacts': {}}
        for task in self.matrix['tasks']:
            for c in task['criteria']:
                for path in c['artifacts']:
                    self.evidence['artifacts'][path] = hashlib.sha256((ROOT/path).read_bytes()).hexdigest()
                self.evidence['acceptance']['passed_tests'].update({t: 'pass' for t in c['tests']})
        sources = {p: h for p, h in self.evidence['artifacts'].items() if p.endswith(('.go', '.py'))}
        self.runtime['source_file_sha256'] = sources
        fp = hashlib.sha256(json.dumps(sources, sort_keys=True).encode()).hexdigest()
        self.runtime['source_fingerprint_sha256'] = fp
        self.evidence['source_fingerprint_sha256'] = fp
        self.evidence['acceptance']['source_fingerprint_sha256'] = fp

    def validate(self):
        return validate_acceptance(ROOT, self.register, self.matrix, self.evidence, self.runtime, COMMANDS)

    def test_consistent_evidence_is_accepted(self):
        self.assertTrue(self.validate()['complete'])

    def test_verified_labels_do_not_replace_execution(self):
        self.evidence['acceptance']['passed_tests'].clear()
        with self.assertRaisesRegex(ValueError, 'no passing execution'):
            self.validate()

    def test_missing_criterion_is_rejected(self):
        self.matrix['tasks'][0]['criteria'].pop()
        with self.assertRaisesRegex(ValueError, 'missing or duplicate'):
            self.validate()

    def test_removed_height_regression_is_rejected(self):
        c = next(c for t in self.matrix['tasks'] for c in t['criteria'] if c['id'] == 'canonical_head_height')
        c['tests'] = c['tests'][1:]
        with self.assertRaisesRegex(ValueError, 'head-height regression omitted'):
            self.validate()

    def test_canonical_timestamp_tests_cannot_be_omitted(self):
        c = next(c for t in self.matrix['tasks'] for c in t['criteria'] if c['id'] == 'canonical_head_height')
        c['tests'] = [t for t in c['tests'] if 'ForkTime' not in t]
        with self.assertRaisesRegex(ValueError, 'timestamp activation coverage omitted'):
            self.validate()

    def test_duplicate_skips_cannot_hide_missing_inventory_entries(self):
        import copy
        run = copy.deepcopy(self.evidence['full_suite'])
        run['named_test_or_subtest_events']['skip'] = 2
        rows = [{'package': 'example', 'test': 'TestSkipped', 'reason': ['fixture absent']}]*2
        with self.assertRaisesRegex(ValueError, 'duplicate or unexplained skip'):
            validate_skips({'full_suite': rows, 'race_suite': []}, {'full_suite': run})

    def test_missing_semantic_case_is_rejected(self):
        c = next(c for t in self.matrix['tasks'] for c in t['criteria'] if c['id'] == 'approval')
        c['tests'].pop()
        with self.assertRaisesRegex(ValueError, 'semantic acceptance coverage omitted'):
            self.validate()

    def test_native_gas_cases_cannot_be_replaced_with_a_label(self):
        c = next(c for t in self.matrix['tasks'] for c in t['criteria'] if c['id'] == 'gas_and_settlement')
        c['tests'] = []
        with self.assertRaisesRegex(ValueError, 'gas/settlement coverage omitted'):
            self.validate()

    def test_unknown_or_canceled_package_results_are_rejected(self):
        import copy
        run = copy.deepcopy(self.evidence['full_suite'])
        run['package_events']['canceled'] = 1
        run['started_packages'] = run['completed_packages'] = 2
        with self.assertRaisesRegex(ValueError, 'unknown or missing execution counter'):
            validate_run(run)

    def test_boolean_completion_is_not_a_package_count(self):
        import copy
        run = copy.deepcopy(self.evidence['full_suite'])
        run['completed_packages'] = True
        with self.assertRaisesRegex(ValueError, 'run is incomplete'):
            validate_run(run)

    def test_stale_or_missing_fingerprint_is_rejected(self):
        path = next(iter(self.evidence['artifacts']))
        self.evidence['artifacts'][path] = 'b' * 64
        with self.assertRaisesRegex(ValueError, 'artifact fingerprint'):
            self.validate()

    def test_stale_source_run_is_rejected(self):
        self.evidence['acceptance']['source_fingerprint_sha256'] = 'b' * 64
        with self.assertRaisesRegex(ValueError, 'stale source fingerprint'):
            self.validate()

    def test_skipped_test_is_not_a_pass(self):
        name = next(iter(self.evidence['acceptance']['passed_tests']))
        self.evidence['acceptance']['passed_tests'][name] = 'skip'
        with self.assertRaisesRegex(ValueError, 'no passing execution'):
            self.validate()

    def test_incomplete_run_is_rejected(self):
        self.evidence['full_suite']['completed_packages'] = 0
        with self.assertRaisesRegex(ValueError, 'run is incomplete'):
            self.validate()

    def test_empty_or_failed_runs_are_rejected(self):
        import copy
        for mutate in (
            lambda r: r.update(started_packages=0, completed_packages=0),
            lambda r: r.update(exit_code=1),
            lambda r: r['package_events'].update(fail=1),
            lambda r: r['named_test_or_subtest_events'].update(fail=1),
            lambda r: r['named_test_or_subtest_events'].update(**{'pass': 0}),
        ):
            run = copy.deepcopy(self.evidence['full_suite'])
            mutate(run)
            with self.assertRaisesRegex(ValueError, 'run (is empty|failed)'):
                validate_run(run)

    def test_failed_race_cannot_hide_behind_zero_skips(self):
        import copy
        race = copy.deepcopy(self.evidence['full_suite'])
        race['exit_code'] = 1
        with self.assertRaisesRegex(ValueError, 'run failed'):
            validate_skips({'full_suite': [], 'race_suite': []}, {'full_suite': self.evidence['full_suite'], 'race_suite': race})

    def test_summary_fingerprint_cannot_differ_from_the_test_run(self):
        self.evidence['source_fingerprint_sha256'] = 'b'*64
        with self.assertRaisesRegex(ValueError, 'stale source fingerprint'):
            self.validate()

    def test_source_manifest_cannot_drop_an_executable_artifact(self):
        sources = self.runtime['source_file_sha256']
        sources.pop(next(iter(sources)))
        fp = hashlib.sha256(json.dumps(sources, sort_keys=True).encode()).hexdigest()
        self.runtime['source_fingerprint_sha256'] = self.evidence['source_fingerprint_sha256'] = self.evidence['acceptance']['source_fingerprint_sha256'] = fp
        with self.assertRaisesRegex(ValueError, 'omitted from source manifest'):
            self.validate()

    def test_passing_claims_cannot_exceed_execution_counts(self):
        self.evidence['full_suite']['named_test_or_subtest_events']['pass'] = 1
        with self.assertRaisesRegex(ValueError, 'exceeds executed'):
            self.validate()

    def test_prerequisite_id_cannot_be_replaced_to_preserve_the_count(self):
        inventory = json.loads((ROOT/'specs/hegota/conformance.json').read_text())
        inventory['requirements'][0]['requirement_id'] = 'PRE-9999'
        with self.assertRaisesRegex(ValueError, 'inventory omission'):
            validate_inventory(ROOT, inventory, self.evidence)

    def test_verified_task_cannot_hide_open_criterion(self):
        self.matrix['tasks'][0]['criteria'][0].update(status='review', remaining_work='new pinned source required')
        with self.assertRaisesRegex(ValueError, 'unresolved criterion'):
            self.validate()

    def test_review_status_can_be_reported_truthfully(self):
        self.register['tasks'][0]['status'] = 'review'
        self.matrix['tasks'][0]['criteria'][0].update(status='review', remaining_work='new pinned source required')
        self.assertFalse(self.validate()['complete'])

    def test_profile_cannot_claim_another_reference(self):
        load = lambda p: json.loads((ROOT / p).read_text())
        profiles = load('specs/hegota/profiles.json')
        profiles['profiles'][0]['reference_version'] = 'v1.17.1'
        with self.assertRaisesRegex(ValueError, 'reference identity differs'):
            validate_reference(ROOT, load('specs/hegota/source-lock.json'), profiles, load('evidence/hegota-a/baseline.json'))

    def test_reference_revision_must_be_locked(self):
        load = lambda p: json.loads((ROOT / p).read_text())
        lock = load('specs/hegota/source-lock.json')
        lock['active_reference']['revision'] = '0' * 40
        with self.assertRaisesRegex(ValueError, 'revision is not locked'):
            validate_reference(ROOT, lock, load('specs/hegota/profiles.json'), load('evidence/hegota-a/baseline.json'))

    def test_reference_wrapper_cannot_reuse_the_direct_backend_label(self):
        load = lambda p: json.loads((ROOT/p).read_text())
        report = load('evidence/hegota-a/baseline.json')
        report['reference_backend'] = 'wrapper/'+report['reference_backend']
        with self.assertRaisesRegex(ValueError, 'reference identity differs'):
            validate_reference(ROOT, load('specs/hegota/source-lock.json'), load('specs/hegota/profiles.json'), report)

    def test_skip_counts_must_match_the_inventory(self):
        with self.assertRaisesRegex(ValueError, 'skip inventory/count mismatch'):
            validate_skips({'full_suite': [], 'race_suite': []},
                           {'full_suite': dict(self.evidence['full_suite'], named_test_or_subtest_events={'pass': 100, 'fail': 0, 'skip': 1})})


class BaselineEvidence(unittest.TestCase):
    def setUp(self):
        self.baseline = json.loads((ROOT/'evidence/hegota-a/baseline.json').read_text())

    def test_all_locked_cases_are_present_and_agree(self):
        validate_baseline(ROOT, self.baseline)

    def test_summary_cannot_hide_missing_case_results(self):
        self.baseline['results'] = []
        with self.assertRaisesRegex(ValueError, 'case results missing'):
            validate_baseline(ROOT, self.baseline)

    def test_duplicate_case_cannot_replace_missing_fork(self):
        self.baseline['results'][1] = self.baseline['results'][0]
        with self.assertRaisesRegex(ValueError, 'case coverage differs'):
            validate_baseline(ROOT, self.baseline)

    def test_unknown_case_is_rejected(self):
        self.baseline['results'][0]['fork'] = 'UnknownFork'
        with self.assertRaisesRegex(ValueError, 'case coverage differs'):
            validate_baseline(ROOT, self.baseline)

    def test_matching_but_wrong_roots_or_logs_do_not_pass(self):
        import copy
        for field in ('root', 'logs'):
            bad = copy.deepcopy(self.baseline)
            bad['results'][0]['native_'+field] = bad['results'][0]['reference_'+field] = '0x'+'11'*32
            with self.assertRaisesRegex(ValueError, 'pinned oracle'):
                validate_baseline(ROOT, bad)

    def test_gas_must_be_an_actual_positive_differential_match(self):
        import copy
        for native, reference in [(1, 2), (0, 0), (True, True), ('41012', '41012')]:
            bad = copy.deepcopy(self.baseline)
            bad['results'][0].update(native_gas=native, reference_gas=reference)
            with self.assertRaisesRegex(ValueError, 'gas differs or is invalid'):
                validate_baseline(ROOT, bad)

    def test_case_failure_cannot_hide_behind_green_totals(self):
        self.baseline['results'][0]['passed'] = False
        with self.assertRaisesRegex(ValueError, 'case did not pass'):
            validate_baseline(ROOT, self.baseline)

    def test_case_errors_cannot_hide_behind_a_pass_flag(self):
        self.baseline['results'][0]['error'] = 'executor failed'
        with self.assertRaisesRegex(ValueError, 'case did not pass'):
            validate_baseline(ROOT, self.baseline)

    def test_wrapper_cannot_be_relabeled_as_native(self):
        self.baseline['native_backend'] = 'wrapper/geth'
        with self.assertRaisesRegex(ValueError, 'not the native executor'):
            validate_baseline(ROOT, self.baseline)


if __name__ == '__main__':
    unittest.main()
