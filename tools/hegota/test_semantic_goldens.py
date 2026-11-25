import copy
import json
import unittest
import subprocess
import sys
from semantic_goldens import ROOT, check


class SemanticFixtures(unittest.TestCase):
    def setUp(self):
        self.data = json.loads((ROOT / 'specs/hegota/semantic-goldens.json').read_text())

    def vector(self, name):
        return next(v for v in self.data['trace_vectors'] if v['id'] == name)

    def test_all_cases_have_executable_consumers(self):
        self.assertEqual(check(self.data)['unexecuted_semantic_trace_seeds'], 0)

    def test_empty_oracle_families_are_rejected(self):
        for family in ('gas_vectors', 'recent_root_vectors', 'boundary_vectors'):
            with self.subTest(family=family):
                bad = copy.deepcopy(self.data)
                bad[family] = []
                with self.assertRaisesRegex(ValueError, 'coverage'):
                    check(bad)

    def test_optimization_cannot_disable_verification(self):
        code = '''import json,sys
sys.path.insert(0, 'tools/hegota')
from semantic_goldens import check
d=json.load(open('specs/hegota/semantic-goldens.json'))
d['gas_vectors'][0]['expected']['intrinsic']=999
try: check(d)
except ValueError: sys.exit(0)
sys.exit(2)
'''
        r = subprocess.run([sys.executable, '-O', '-c', code], cwd=ROOT, capture_output=True, text=True)
        self.assertEqual(r.returncode, 0, r.stdout+r.stderr)

    def test_every_boundary_expected_result_is_consumed(self):
        for v in self.data['boundary_vectors']:
            with self.subTest(case=v['id']):
                original = copy.deepcopy(v['expected'])
                v['expected']['valid'] = not v['expected']['valid']
                with self.assertRaisesRegex(ValueError, 'oracle mismatch'):
                    check(self.data)
                v['expected'] = original

    def test_unknown_native_expected_field_is_rejected(self):
        self.vector('approval-zero')['expected']['steps'][0]['nonse'] = 7
        with self.assertRaisesRegex(ValueError, 'unknown fields'):
            check(self.data)

    def test_boolean_is_not_a_nonce_or_expected_integer(self):
        self.vector('keyed-nonce-single-use')['input']['sequence'] = True
        with self.assertRaisesRegex(ValueError, 'malformed nonce'):
            check(self.data)
        self.data['trace_vectors'] = json.loads((ROOT/'specs/hegota/semantic-goldens.json').read_text())['trace_vectors']
        self.vector('keyed-nonce-single-use')['expected']['approval_applied'] = 1
        with self.assertRaisesRegex(ValueError, 'oracle mismatch'):
            check(self.data)

    def test_native_case_cannot_be_reclassified_as_conditional(self):
        self.vector('approval-zero')['eip'] = 8250
        with self.assertRaisesRegex(ValueError, 'consumer/EIP mismatch'):
            check(self.data)

    def test_case_names_cannot_hide_missing_protocol_coverage(self):
        nonce, root = self.vector('keyed-nonce-single-use'), self.vector('recent-root-slot-window')
        nonce['id'], root['id'] = root['id'], nonce['id']
        with self.assertRaisesRegex(ValueError, 'case kind mismatch'):
            check(self.data)

    def test_missing_declared_rollback_expectation_is_rejected(self):
        del self.vector('verify-rollback')['expected']['state_pool_restored']
        with self.assertRaisesRegex(ValueError, 'missing or unknown fields'):
            check(self.data)

    def test_every_gas_fee_and_refund_oracle_is_consumed(self):
        for v in self.data['gas_vectors']:
            with self.subTest(case=v['id']):
                original = v['expected']['payer_refund']
                v['expected']['payer_refund'] += 1
                with self.assertRaisesRegex(ValueError, 'gas oracle mismatch'):
                    check(self.data)
                v['expected']['payer_refund'] = original

    def test_impossible_execution_cannot_be_a_valid_oracle(self):
        self.data['gas_vectors'][0]['input']['execution_used'] = 50001
        with self.assertRaisesRegex(ValueError, 'exceeds frame budget'):
            check(self.data)

    def test_keyed_approval_preserves_current_account_nonce(self):
        v = next(v for v in self.data['boundary_vectors'] if v['id'] == 'nonce-keyed-preserves-current-legacy')
        v['expected']['legacy_nonce'] = v['input']['legacy_nonce']
        with self.assertRaisesRegex(ValueError, 'conditional oracle mismatch'):
            check(self.data)

    def test_text_seed_cannot_replace_a_fixture(self):
        self.vector('approval-zero')['input'] = 'APPROVE scope=0'
        with self.assertRaisesRegex(ValueError, 'concrete fixture'):
            check(self.data)

    def test_duplicate_case_cannot_hide_missing_coverage(self):
        self.data['trace_vectors'][1] = copy.deepcopy(self.data['trace_vectors'][0])
        with self.assertRaisesRegex(ValueError, 'missing/duplicate'):
            check(self.data)

    def test_single_use_expected_nonce_and_cost_are_checked(self):
        v = self.vector('keyed-nonce-single-use')
        for field, bad in [('stored', [0, 0]), ('state_charge', 0), ('legacy_nonce', 8), ('replay_valid', True)]:
            with self.subTest(field=field):
                original = copy.deepcopy(v['expected'])
                v['expected'][field] = bad
                with self.assertRaises(ValueError):
                    check(self.data)
                v['expected'] = original

    def test_insufficient_state_budget_cannot_consume_keys(self):
        self.vector('keyed-nonce-single-use')['input']['state_budget'] -= 1
        with self.assertRaises(ValueError):
            check(self.data)

    def test_approval_revert_cannot_keep_nonce_consumption(self):
        self.vector('keyed-nonce-single-use')['input']['approval_reverts'] = True
        with self.assertRaises(ValueError):
            check(self.data)

    def test_slot_edges_and_orphaned_roots_are_checked(self):
        v = self.vector('recent-root-slot-window')
        for field, bad in [('canonical_valid', [True, True, True, False]), ('canonical_valid', [False, True, True, True]), ('after_reorg_valid', [False, True, False, False])]:
            original = copy.deepcopy(v['expected'])
            v['expected'][field] = bad
            with self.assertRaises(ValueError):
                check(self.data)
            v['expected'] = original


if __name__ == '__main__':
    unittest.main()
