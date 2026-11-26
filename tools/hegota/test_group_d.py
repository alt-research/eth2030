import copy
import json
import unittest
from check_d import ROOT, REQUIRED, validate_manifest, verify_report

class GroupDIntegrityTests(unittest.TestCase):
    def setUp(self):
        self.manifest = json.loads((ROOT/'specs/hegota/group-d.json').read_text())
    def test_all_tasks_criteria_and_native_boundaries_present(self):
        tests, artifacts = validate_manifest(ROOT, self.manifest)
        self.assertEqual(sum(map(len, REQUIRED.values())), 32)
        self.assertGreaterEqual(len(tests), 45)
        self.assertIn('pkg/node/dependency_8504_integration_test.go', artifacts)
    def test_each_task_required(self):
        for i in range(8):
            m=copy.deepcopy(self.manifest);del m['tasks'][i]
            with self.assertRaises(ValueError):validate_manifest(ROOT,m)
    def test_each_criterion_required(self):
        for i in range(8):
            for j in range(4):
                m=copy.deepcopy(self.manifest);del m['tasks'][i]['criteria'][j]
                with self.assertRaises(ValueError):validate_manifest(ROOT,m)
    def test_each_boundary_consumer_required(self):
        for i,row in enumerate(self.manifest['tasks']):
            for j,c in enumerate(row['criteria']):
                for test in c['tests']:
                    m=copy.deepcopy(self.manifest);m['tasks'][i]['criteria'][j]['tests'].remove(test)
                    with self.assertRaises(ValueError):validate_manifest(ROOT,m)
    def test_normative_source_digest_and_revision_required(self):
        for key,value in [('sha256','0'*64),('blob','0'*40),('revision','master')]:
            m=copy.deepcopy(self.manifest);m['normative_sources'][0][key]=value
            with self.assertRaises(ValueError):validate_manifest(ROOT,m)
    def test_no_substitute_crypto_or_automatic_compensation(self):
        for profile,key,value in [('direct8504','scheme',1),('direct8504','witness_bytes',7856),('direct8504','aggregation',True),('assert7906','automatic_token_compensation',True),('adoption8298','source_created_this_tx','accept')]:
            m=copy.deepcopy(self.manifest);m['profiles'][profile][key]=value
            with self.assertRaises(ValueError):validate_manifest(ROOT,m)
    def test_no_silent_public_gate_closure(self):
        m=copy.deepcopy(self.manifest);m['public_gates']=[]
        with self.assertRaises(ValueError):validate_manifest(ROOT,m)
    def test_no_empty_or_fake_public_report(self):
        report={'source_fingerprint_sha256':'x','artifacts':{},'tasks':8,'criteria':32,'public_complete':False,'public_gates':self.manifest['public_gates'],'selected_profile_consumers_passed':True,'passed_tests':{},'run':{'named':{},'packages':{},'completed_packages':0,'log_sha256':'0'*64}}
        with self.assertRaises(ValueError):verify_report(report,self.manifest,set(),{},'x')
        report['public_complete']=True
        with self.assertRaises(ValueError):verify_report(report,self.manifest,{'test'}, {},'x')
