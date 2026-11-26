import copy
import json
import tempfile
import unittest
from pathlib import Path

from check_b import ROOT, fingerprint, read_events, validate_manifest, verify_report

class GroupBCoverageTests(unittest.TestCase):
    def manifest(self):
        return json.loads((ROOT/'specs/hegota/group-b.json').read_text())

    def test_all_twelve_tasks_have_executable_consumers(self):
        tests, artifacts = validate_manifest(ROOT, self.manifest())
        self.assertGreaterEqual(len(tests), 50)
        self.assertIn('pkg/core/vm/frame_8250.go', artifacts)

    def test_missing_task_rejected(self):
        manifest = self.manifest()
        manifest['tasks'].pop()
        with self.assertRaisesRegex(ValueError, 'task omission'):
            validate_manifest(ROOT, manifest)

    def test_missing_criterion_rejected(self):
        manifest = self.manifest()
        manifest['tasks'][0]['criteria'].pop()
        with self.assertRaisesRegex(ValueError, 'criterion omission'):
            validate_manifest(ROOT, manifest)

    def test_public_nonce_activation_rejected(self):
        manifest = self.manifest()
        manifest['profiles']['8250-nonce-only-delta-isolated-v1']['public_activation'] = True
        with self.assertRaisesRegex(ValueError, 'stay isolated'):
            validate_manifest(ROOT, manifest)

    def test_nonexistent_native_test_rejected(self):
        manifest = self.manifest()
        manifest['tasks'][0]['criteria'][0]['tests'] = ['github.com/eth2030/eth2030/core/types::TestInventedAcceptance']
        with self.assertRaisesRegex(ValueError, 'missing native test'):
            validate_manifest(ROOT, manifest)

    def test_missing_or_skipped_native_evidence_rejected(self):
        with self.assertRaisesRegex(ValueError, 'omitted/failed/skipped'):
            verify_report({'source_fingerprint_sha256': 'digest', 'artifacts': {}, 'passed_tests': {}},
                          {'package::TestRequired'}, {}, 'digest')

    def test_stale_fingerprint_rejected(self):
        with self.assertRaisesRegex(ValueError, 'stale B'):
            verify_report({'source_fingerprint_sha256': 'old', 'artifacts': {}}, set(), {}, 'new')

    def test_incomplete_and_failed_go_runs_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            log = Path(tmp)/'run.jsonl'
            for events, message in [([{'Action':'start','Package':'p'}], 'incomplete'),
                                    ([{'Action':'start','Package':'p'}, {'Action':'fail','Package':'p'}], 'failed')]:
                log.write_text(''.join(json.dumps(e)+'\n' for e in events))
                with self.assertRaisesRegex(ValueError, message):
                    read_events(log)

    def test_missing_normative_source_rejected(self):
        for eip in (7702, 2929, 214, 1153):
            with self.subTest(eip=eip):
                manifest = self.manifest()
                manifest['normative_sources'] = [s for s in manifest['normative_sources'] if s['eip'] != eip]
                with self.assertRaisesRegex(ValueError, 'missing normative source'):
                    validate_manifest(ROOT, manifest)

    def test_missing_dispatch_boundary_regression_rejected(self):
        for task, criterion_id, name in (
            ('HGT-015', 'bal', 'TestFrame8141DirectPrecompileBALGasBoundary'),
            ('HGT-015', 'bal', 'TestFrame8141StaticSstoreHasNoStorageAccess'),
            ('HGT-003', 'scope_identity', 'TestFrame8141ApproveEmptyReturnIgnoresOffset'),
            ('HGT-003', 'scope_identity', 'TestFrame8141ApproveReturnMemoryBoundaries'),
            ('HGT-011', 'pool_isolation', 'TestFrame8141PrefundedCreationBAL'),
            ('HGT-012', 'approval_batches', 'TestFrame8141PrefundedCreationRollbackBAL'),
            ('HGT-013', 'build_import', 'TestCanonicalFrame8141PrefundedCreateBuildImport'),
            ('HGT-015', 'bal', 'TestFrame8141PrefundedCreationBAL'),
            ('HGT-015', 'bal', 'TestCanonicalFrame8141PrefundedCreateBuildImport'),
            ('HGT-012', 'transient_warmness', 'TestCanonicalFrame8141PersistentBackendJournals'),
            ('HGT-013', 'native_route', 'TestCanonicalFrame8141PersistentBackendJournals'),
            ('HGT-015', 'bal', 'TestCanonicalFrame8141PersistentBackendJournals'),
            ('HGT-016', 'restart_history', 'TestCanonicalFrame8141PersistentBackendJournals'),
        ):
            with self.subTest(test=name):
                manifest = self.manifest()
                row = next(t for t in manifest['tasks'] if t['id'] == task)
                criterion = next(c for c in row['criteria'] if c['id'] == criterion_id)
                criterion['tests'] = [t for t in criterion['tests'] if not t.endswith(name)]
                with self.assertRaisesRegex(ValueError, 'mandatory boundary regression omitted'):
                    validate_manifest(ROOT, manifest)
