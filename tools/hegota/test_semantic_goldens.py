import copy
import json
import unittest
from semantic_goldens import ROOT, check


class SemanticFixtures(unittest.TestCase):
    def setUp(self):
        self.data = json.loads((ROOT / 'specs/hegota/semantic-goldens.json').read_text())

    def vector(self, name):
        return next(v for v in self.data['trace_vectors'] if v['id'] == name)

    def test_all_cases_have_executable_consumers(self):
        self.assertEqual(check(self.data)['unexecuted_semantic_trace_seeds'], 0)

    def test_text_seed_cannot_replace_a_fixture(self):
        self.vector('approval-zero')['input'] = 'APPROVE scope=0'
        with self.assertRaisesRegex(AssertionError, 'concrete fixture'):
            check(self.data)

    def test_duplicate_case_cannot_hide_missing_coverage(self):
        self.data['trace_vectors'][1] = copy.deepcopy(self.data['trace_vectors'][0])
        with self.assertRaisesRegex(AssertionError, 'missing/duplicate'):
            check(self.data)

    def test_single_use_expected_nonce_and_cost_are_checked(self):
        v = self.vector('keyed-nonce-single-use')
        for field, bad in [('stored', [0, 0]), ('state_charge', 0), ('legacy_nonce', 8), ('replay_valid', True)]:
            with self.subTest(field=field):
                original = copy.deepcopy(v['expected'])
                v['expected'][field] = bad
                with self.assertRaises(AssertionError):
                    check(self.data)
                v['expected'] = original

    def test_insufficient_state_budget_cannot_consume_keys(self):
        self.vector('keyed-nonce-single-use')['input']['state_budget'] -= 1
        with self.assertRaises(AssertionError):
            check(self.data)

    def test_approval_revert_cannot_keep_nonce_consumption(self):
        self.vector('keyed-nonce-single-use')['input']['approval_reverts'] = True
        with self.assertRaises(AssertionError):
            check(self.data)

    def test_slot_edges_and_orphaned_roots_are_checked(self):
        v = self.vector('recent-root-slot-window')
        for field, bad in [('canonical_valid', [True, True, True, False]), ('canonical_valid', [False, True, True, True]), ('after_reorg_valid', [False, True, False, False])]:
            original = copy.deepcopy(v['expected'])
            v['expected'][field] = bad
            with self.assertRaises(AssertionError):
                check(self.data)
            v['expected'] = original


if __name__ == '__main__':
    unittest.main()
