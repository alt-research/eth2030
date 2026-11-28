"""Mutation controls for the frozen Group E acceptance register."""
import copy
import json
import unittest
from check_e import ROOT, REQUIRED, REQUIRED_ENCODING_FILES, validate_manifest, verify_report
from check_b import fingerprint


class GroupEAcceptanceTests(unittest.TestCase):
    def setUp(self):
        self.manifest = json.loads((ROOT/'specs/hegota/group-e.json').read_text())

    def test_complete_register(self):
        tests, artifacts = validate_manifest(ROOT, self.manifest)
        self.assertEqual(len(self.manifest['tasks']), 9)
        self.assertEqual(sum(len(t['criteria']) for t in self.manifest['tasks']), 36)
        self.assertGreaterEqual(len(tests), 25)
        self.assertTrue(artifacts)

    def test_missing_task(self):
        self.manifest['tasks'].pop()
        with self.assertRaises(ValueError): validate_manifest(ROOT, self.manifest)

    def test_missing_criterion(self):
        self.manifest['tasks'][0]['criteria'].pop()
        with self.assertRaises(ValueError): validate_manifest(ROOT, self.manifest)

    def test_removed_gate(self):
        self.manifest['public_gates'].pop()
        with self.assertRaises(ValueError): validate_manifest(ROOT, self.manifest)

    def test_removed_native_consumer(self):
        self.manifest['tasks'][0]['criteria'][0]['tests'] = []
        with self.assertRaises(ValueError): validate_manifest(ROOT, self.manifest)

    def test_source_digest_tampering(self):
        self.manifest['normative_sources'][0]['sha256'] = '0'*64
        with self.assertRaises(ValueError): validate_manifest(ROOT, self.manifest)

    def test_canonical_encoding_source_cannot_be_omitted(self):
        for path in REQUIRED_ENCODING_FILES:
            manifest = copy.deepcopy(self.manifest)
            for task in manifest['tasks']:
                for criterion in task['criteria']:
                    criterion['files'] = [p for p in criterion['files'] if p != path]
            manifest['additional_artifacts'] = [p for p in manifest['additional_artifacts'] if p != path]
            with self.assertRaises(ValueError): validate_manifest(ROOT, manifest)

    def test_report_cannot_claim_public_completion_or_skip_consumer(self):
        tests, artifacts = validate_manifest(ROOT, self.manifest)
        hashes, digest = fingerprint(ROOT, artifacts)
        report = dict(tasks=9, criteria=36, source_fingerprint_sha256=digest,
                      artifacts=hashes, public_complete=False,
                      public_gates=self.manifest['public_gates'],
                      selected_profile_consumers_passed=True,
                      passed_tests={t:'pass' for t in tests},
                      run=dict(named={'pass':len(tests)}, packages={'pass':1},
                               completed_packages=1, log_sha256='a'*64))
        verify_report(report,self.manifest,tests,hashes,digest)
        for field,value in [('public_complete',True),('source_fingerprint_sha256','0'*64),('passed_tests',{})]:
            changed=copy.deepcopy(report); changed[field]=value
            with self.assertRaises(ValueError): verify_report(changed,self.manifest,tests,hashes,digest)


if __name__ == '__main__': unittest.main()
