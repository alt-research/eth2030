import hashlib
import json
import unittest
from acceptance import COMMANDS, validate_acceptance, validate_reference, validate_skips
from check import ROOT


class AcceptanceEvidence(unittest.TestCase):
    def setUp(self):
        load = lambda p: json.loads((ROOT / p).read_text())
        self.register = load('specs/hegota/tasks.json')
        self.matrix = load('specs/hegota/acceptance.json')
        self.runtime = {'source_fingerprint_sha256': 'a' * 64}
        self.evidence = {'acceptance': {'source_fingerprint_sha256': 'a' * 64, 'passed_tests': {}},
                         'full_suite': {'exit_code': 0, 'named_test_or_subtest_events': {'fail': 0},
                                        'started_packages': 1, 'completed_packages': 1}, 'artifacts': {}}
        for task in self.matrix['tasks']:
            for c in task['criteria']:
                for path in c['artifacts']:
                    self.evidence['artifacts'][path] = hashlib.sha256((ROOT/path).read_bytes()).hexdigest()
                self.evidence['acceptance']['passed_tests'].update({t: 'pass' for t in c['tests']})

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

    def test_missing_semantic_case_is_rejected(self):
        c = next(c for t in self.matrix['tasks'] for c in t['criteria'] if c['id'] == 'approval')
        c['tests'].pop()
        with self.assertRaisesRegex(ValueError, 'semantic acceptance coverage omitted'):
            self.validate()

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

    def test_skip_counts_must_match_the_inventory(self):
        with self.assertRaisesRegex(ValueError, 'skip inventory/count mismatch'):
            validate_skips({'full_suite': [], 'race_suite': []},
                           {'full_suite': {'named_test_or_subtest_events': {'skip': 1}}})


if __name__ == '__main__':
    unittest.main()
