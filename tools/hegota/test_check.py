import copy
import hashlib
import json
import tempfile
import unittest
from pathlib import Path
from check import ROOT, compare, validate_admission, validate_lock

class RegisterValidation(unittest.TestCase):
    def setUp(self):
        self.admission = json.loads((ROOT/'specs/hegota/admission.json').read_text())
        self.lock = json.loads((ROOT/'specs/hegota/source-lock.json').read_text())
        self.expected = {r['eip'] for r in self.admission['records']}
        self.tasks = {t for r in self.admission['records'] for t in r['tasks']}

    def changed_source(self, admission, lock, change_bytes=True):
        record = admission['records'][0]
        record['spec_commit'] = 'b'*40
        record['spec_url'] = record['spec_url'].replace(self.admission['records'][0]['spec_commit'], record['spec_commit'])
        if change_bytes:
            record['spec_sha256'] = 'a'*64
        source = next(f for f in lock['files'] if f.get('eip') == record['eip'])
        source.update(commit=record['spec_commit'], url=record['spec_url'], sha256=record['spec_sha256'])
        return record

    def record_change(self, record):
        h = copy.deepcopy(record['history'][-1])
        h.update(previous_local_sha256=self.admission['records'][0]['spec_sha256'],
                 new_spec_sha256=record['spec_sha256'], date='2026-11-26')
        record['history'].append(h)

    def test_duplicate_eip_is_rejected(self):
        self.admission['records'][1] = self.admission['records'][0]
        with self.assertRaisesRegex(ValueError, 'scope'):
            validate_admission(self.admission, self.expected, self.tasks)

    def test_missing_task_and_history_rejected(self):
        self.admission['records'][0]['tasks'] = []
        with self.assertRaisesRegex(ValueError, 'task'):
            validate_admission(self.admission, self.expected, self.tasks)

    def test_unrecorded_upstream_change_rejected(self):
        new, lock = copy.deepcopy(self.admission), copy.deepcopy(self.lock)
        self.changed_source(new, lock)
        with self.assertRaisesRegex(ValueError, 'impact decision'):
            compare(self.lock, lock, self.admission, new)

    def test_recorded_change_reports_impacted_tasks_without_adopting(self):
        new, lock = copy.deepcopy(self.admission), copy.deepcopy(self.lock)
        record = self.changed_source(new, lock)
        self.record_change(record)
        result = compare(self.lock, lock, self.admission, new)
        self.assertFalse(result['adopted'])
        self.assertEqual(result['eip_changes'][0]['affected_tasks'], record['tasks'])

    def test_lock_change_without_admission_change_rejected(self):
        lock = copy.deepcopy(self.lock)
        source = next(f for f in lock['files'] if f.get('eip') == self.admission['records'][0]['eip'])
        source['sha256'] = 'a'*64
        with self.assertRaisesRegex(ValueError, 'immutable source digest|source lock/admission mismatch'):
            compare(self.lock, lock, self.admission, self.admission)

    def test_immutable_non_eip_digest_change_rejected(self):
        lock = copy.deepcopy(self.lock)
        source = next(f for f in lock['files'] if 'eip' not in f)
        source['sha256'] = 'a'*64
        with self.assertRaisesRegex(ValueError, 'immutable source digest'):
            compare(self.lock, lock, self.admission, self.admission)

    def test_same_bytes_new_revision_requires_history(self):
        new, lock = copy.deepcopy(self.admission), copy.deepcopy(self.lock)
        record = self.changed_source(new, lock, change_bytes=False)
        with self.assertRaisesRegex(ValueError, 'impact decision'):
            compare(self.lock, lock, self.admission, new)
        self.record_change(record)
        self.assertEqual(len(compare(self.lock, lock, self.admission, new)['eip_changes']), 1)

    def test_rewriting_prior_history_rejected(self):
        new = copy.deepcopy(self.admission)
        new['records'][0]['history'][0]['rationale'] = 'rewritten'
        with self.assertRaisesRegex(ValueError, 'history was rewritten'):
            compare(self.lock, self.lock, self.admission, new)

    def test_repository_update_requires_impact_decision(self):
        lock = copy.deepcopy(self.lock)
        source = lock['repositories'][0]
        old = source['sha']; source['sha'] = 'b'*40
        with self.assertRaisesRegex(ValueError, 'repository update lacks'):
            compare(self.lock, lock, self.admission, self.admission)
        lock['source_updates'] = [dict(repo=source['repo'], ref=source['ref'], previous_sha=old,
            new_sha=source['sha'], date='2026-11-26', affected_tasks=['HGT-001'], rationale='reviewed revision update')]
        self.assertEqual(compare(self.lock, lock, self.admission, self.admission)['repository_changes'][0]['affected_tasks'], ['HGT-001'])

    def test_duplicate_repository_pin_rejected(self):
        lock = copy.deepcopy(self.lock)
        lock['repositories'][1] = lock['repositories'][0]
        with self.assertRaisesRegex(ValueError, 'duplicate source repository'):
            validate_lock(lock, ROOT)

    def test_new_context_source_requires_impact_record(self):
        lock = copy.deepcopy(self.lock)
        source = dict(url='https://raw.githubusercontent.com/ethereum/execution-apis/'+'a'*40+'/src/engine/new.md', sha256='a'*64)
        lock['files'].append(source)
        with self.assertRaisesRegex(ValueError, 'file update lacks'):
            compare(self.lock, lock, self.admission, self.admission)
        lock['file_updates'] = [dict(url=source['url'], previous_sha256=None, new_sha256=source['sha256'],
            date='2026-11-26', affected_tasks=['HGT-001'], rationale='reviewed additional source')]
        self.assertEqual(compare(self.lock, lock, self.admission, self.admission)['file_changes'][0]['affected_tasks'], ['HGT-001'])

    def test_tampered_normative_file_rejected(self):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d); p = root/'source.md'; p.write_text('spec')
            (root/'pkg').mkdir(); (root/'pkg/go.sum').write_text('deps')
            lock = {'version':1, 'repositories':[{'repo':f'repo-{i}','ref':'main','sha':'a'*40} for i in range(14)],
                    'files':[{'url':'https://example.org/'+'a'*40+'/source.md',
                              'sha256':hashlib.sha256(b'spec').hexdigest(), 'local_path':'source.md'}],
                    'go_sum_sha256':hashlib.sha256(b'deps').hexdigest()}
            self.assertEqual(validate_lock(lock, root), 1); p.write_text('tampered')
            with self.assertRaisesRegex(ValueError, 'source changed'):
                validate_lock(lock, root)

if __name__ == '__main__':
    unittest.main()
