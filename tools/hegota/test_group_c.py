import copy
import json
import unittest
from check_c import ROOT, REQUIRED, PAYMASTER_REQUIRED, MODULE, validate_manifest, verify_report

class GroupCIntegrityTests(unittest.TestCase):
    def setUp(self):
        self.manifest=json.loads((ROOT/'specs/hegota/group-c.json').read_text())
    def test_complete_native_register(self):
        tests,artifacts=validate_manifest(ROOT,self.manifest)
        self.assertEqual(sum(map(len,REQUIRED.values())),40)
        self.assertGreater(len(tests),30)
        self.assertIn('pkg/node/frame_c_integration_test.go',artifacts)
    def test_every_task_required(self):
        for i in range(10):
            m=copy.deepcopy(self.manifest);del m['tasks'][i]
            with self.assertRaises(ValueError):validate_manifest(ROOT,m)
    def test_every_criterion_required(self):
        for i in range(10):
            for j in range(4):
                m=copy.deepcopy(self.manifest);del m['tasks'][i]['criteria'][j]
                with self.assertRaises(ValueError):validate_manifest(ROOT,m)
    def test_duplicate_criterion_rejected(self):
        m=copy.deepcopy(self.manifest);m['tasks'][0]['criteria'][1]=m['tasks'][0]['criteria'][0]
        with self.assertRaises(ValueError):validate_manifest(ROOT,m)
    def test_unqualified_public_composition_rejected(self):
        for field,value in [('fees',['max_priority_fee_per_gas','max_fee_per_gas','max_fee_per_blob_gas']),('activation','HogotaTime'),('decision','D01-open')]:
            manifest=copy.deepcopy(self.manifest)
            manifest['profiles']['8250-public'][field]=value
            with self.assertRaises(ValueError):validate_manifest(ROOT,manifest)
    def test_conditional_signer_profile_required(self):
        del self.manifest['profiles']['8151-conditional']
        with self.assertRaises((ValueError,KeyError)):validate_manifest(ROOT,self.manifest)
    def test_public_integration_boundaries_required(self):
        for boundary in ['TestFrame8250NativePeersPoolBuildSimulationAndRestart','TestFrame8250PublicExecutionSchemaBoundary','TestFrame8250PublicIdentityDependenciesAndForkEviction','TestEIP8151CallFamiliesAndOuterRevert','TestFrame8151ReservedAndOtherDigestShares','TestFrame8151PayRecoveriesCountDistinctPaymasters','TestFrame8250TxPoolMapDoesNotAliasLegacyNonce']:
            manifest=copy.deepcopy(self.manifest)
            for row in manifest['tasks']:
                for criterion in row['criteria']:
                    criterion['tests']=[test for test in criterion['tests'] if boundary not in test]
            with self.assertRaises(ValueError):validate_manifest(ROOT,manifest)
    def test_local_template_cannot_be_canonical(self):
        self.manifest['profiles']['paymaster-v1']['upstream_canonical']=True
        with self.assertRaises(ValueError):validate_manifest(ROOT,self.manifest)
    def test_source_digest_required(self):
        self.manifest['normative_sources'][0]['sha256']='0'*64
        with self.assertRaises(ValueError):validate_manifest(ROOT,self.manifest)
    def test_boundary_test_required(self):
        for row in self.manifest['tasks']:
            for c in row['criteria']:
                c['tests']=[n for n in c['tests'] if 'FrameCNativePeers' not in n]
        with self.assertRaises(ValueError):validate_manifest(ROOT,self.manifest)
    def test_followup_boundaries_required(self):
        for boundary in ['TestFrameCIsolatedKeyedCachedDimensionalLimits', 'TestFrameCPendingAndMinedQueryLocations', 'TestFrameCSimulationTraceLimitIsExplicit', 'TestFrameCSimulationOverrideErrorsThroughRPC', 'TestFrameCPublicTransientStorageScope']:
            manifest=copy.deepcopy(self.manifest)
            for row in manifest['tasks']:
                for criterion in row['criteria']:
                    criterion['tests']=[name for name in criterion['tests'] if boundary not in name]
            with self.assertRaises(ValueError):validate_manifest(ROOT,manifest)
    def test_empty_unexecuted_report_rejected(self):
        report={'source_fingerprint_sha256':'x','artifacts':{},'tasks':10,'criteria':40,'public_complete':False,'public_gates':self.manifest['public_gates'],'selected_profile_consumers_passed':True,'passed_tests':{},'run':{'named':{},'packages':{},'completed_packages':0}}
        with self.assertRaises(ValueError):verify_report(report,self.manifest,set(),{},'x')
    def test_closed_gate_claim_rejected(self):
        report={'source_fingerprint_sha256':'x','artifacts':{},'tasks':10,'criteria':40,'public_complete':True,'public_gates':{},'selected_profile_consumers_passed':True,'passed_tests':{}}
        with self.assertRaises(ValueError):verify_report(report,self.manifest,set(),{},'x')

    def test_cross_transaction_boundaries_required(self):
        for boundary in ['TestFrameCRevalidationDropsInvalidBeforeSharedExposure','TestFrameCSharedPayerRevalidationEvictsByPolicy','TestFrameCSignerRevalidationUsesOneHeadSnapshot','TestFrameCBlobAndOrdinarySenderOrdering','TestFrameCRemovalReconcilesScalarSuccessors','TestFrameCEvictionIncludesNonTailScalarFrame']:
            manifest=copy.deepcopy(self.manifest)
            for row in manifest['tasks']:
                for criterion in row['criteria']:
                    criterion['tests']=[test for test in criterion['tests'] if boundary not in test]
            with self.assertRaises(ValueError):validate_manifest(ROOT,manifest)

    def test_paymaster_proposal_cannot_change_identity_or_authority(self):
        for key,value in [('public_default',True),('upstream_canonical',True),('revision','master'),('code_hash','0x'+'00'*32),('ecRecover_dependencies',True),('delay_seconds',43200)]:
            manifest=copy.deepcopy(self.manifest)
            manifest['profiles']['paymaster-12041'][key]=value
            with self.assertRaises(ValueError):validate_manifest(ROOT,manifest)

    def test_every_paymaster_consumer_required(self):
        for boundary in PAYMASTER_REQUIRED:
            manifest=copy.deepcopy(self.manifest)
            for row in manifest['tasks']:
                for criterion in row['criteria']:
                    criterion['tests']=[name for name in criterion['tests'] if name!=MODULE+boundary]
            with self.assertRaises(ValueError):validate_manifest(ROOT,manifest)
