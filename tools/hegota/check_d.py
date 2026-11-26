#!/usr/bin/env python3
"""Source-bound Group D coverage. Completion labels cannot replace native evidence."""
import argparse
import hashlib
import json
import re
import subprocess
from pathlib import Path
from check_b import fingerprint, need, read_events, sha

ROOT = Path(__file__).resolve().parents[2]
MODULE = 'github.com/eth2030/eth2030/'
REQUIRED = {'HGT-030': {'D30.1': ['github.com/eth2030/eth2030/core/types::TestDependency8504IndependentGloasProgressivePayload', 'github.com/eth2030/eth2030/crypto/leansphincs::TestEIP8504IndependentEthLibVerificationVectors'], 'D30.2': ['github.com/eth2030/eth2030/core/frameproof::TestDependency8504MissingTamperedAndWrapperControls'], 'D30.3': ['github.com/eth2030/eth2030/core/config::TestFrameDOptionalActivationAndCompatibility'], 'D30.4': ['github.com/eth2030/eth2030/eth::TestDependency8504NativePeerRequestAdmissionAndIsolation']}, 'HGT-031': {'D31.1': ['github.com/eth2030/eth2030/core/types::TestFrameDProofConstraintsAndAssertComposition'], 'D31.2': ['github.com/eth2030/eth2030/core/types::TestFrameDDependenciesSortDeduplicateAndUnion'], 'D31.3': ['github.com/eth2030/eth2030/core/frameproof::TestDependency8504IndependentSubmissionWrapperAndBlock'], 'D31.4': ['github.com/eth2030/eth2030/core/types::TestFrameDIndependentProofFrameWireHashAndGas']}, 'HGT-032': {'D32.1': ['github.com/eth2030/eth2030/crypto/leansphincs::TestEIP8504IndependentEthLibVerificationVectors'], 'D32.2': ['github.com/eth2030/eth2030/crypto/leansphincs::TestEIP8504IndependentEthLibVerificationVectors', 'github.com/eth2030/eth2030/core/execution::TestDependency8504CanonicalSigningHashWalletBaseAndKeyed'], 'D32.3': ['github.com/eth2030/eth2030/crypto/leansphincs::TestEIP8504VerifierRejectsEveryTruncationBeforeHashing'], 'D32.4': ['github.com/eth2030/eth2030/core/vm::TestFrameDProofExecutionHasNoCallOrAccountAccess', 'github.com/eth2030/eth2030/core/types::TestFrameDProofPrefixPositionsAndExcludedWork']}, 'HGT-033': {'D33.1': ['github.com/eth2030/eth2030/txpool::TestDependency8504PoolRejectsBodyWitnessBeforeSimulation'], 'D33.2': ['github.com/eth2030/eth2030/txpool::TestDependency8504PoolAdmissionLifecycleAndBytes', 'github.com/eth2030/eth2030/txpool::TestDependency8504ExpiryAndActualWrapperByteBudget', 'github.com/eth2030/eth2030/txpool::TestDependency8504ReplacementWrapperAtomicAccounting'], 'D33.3': ['github.com/eth2030/eth2030/eth::TestDependency8504NativePeerRequestAdmissionAndIsolation'], 'D33.4': ['github.com/eth2030/eth2030/node::TestDependency8504ActualRPCSimulationEngineBodiesAndRestart', 'github.com/eth2030/eth2030/eth::TestDependency8504PeerRejectsWrongProfileAndUnsolicitedResponse', 'github.com/eth2030/eth2030/eth::TestDependency8504PeerDrainsBoundedAnnouncementQueue']}, 'HGT-034': {'D34.1': ['github.com/eth2030/eth2030/core/frameproof::TestDependency8504IndependentSubmissionWrapperAndBlock', 'github.com/eth2030/eth2030/core/frameproof::TestDependency8504EmptyBlockCommitment'], 'D34.2': ['github.com/eth2030/eth2030/core/block::TestDependency8504NativeBuildImportInvalidControlsAndRestart', 'github.com/eth2030/eth2030/engine/payload::TestDependency8504PayloadAndBodyJSONPresence', 'github.com/eth2030/eth2030/p2p/ethproto::TestDependency8504BlockBodiesPreserveActiveEmptyAndBytes'], 'D34.3': ['github.com/eth2030/eth2030/core/block::TestDependency8504NativeBuildImportInvalidControlsAndRestart'], 'D34.4': ['github.com/eth2030/eth2030/core/types::TestDependency8504IndependentGloasProgressivePayload']}, 'HGT-035': {'D35.1': ['github.com/eth2030/eth2030/core/vm::TestSetCodeFrom8298GasValidityAndPaidAccess', 'github.com/eth2030/eth2030/core/vm::TestSetCodeFrom8298EarlyGuardsAndImmediateScan'], 'D35.2': ['github.com/eth2030/eth2030/core/vm::TestSetCodeFrom8298ContextsCreationAndRollback'], 'D35.3': ['github.com/eth2030/eth2030/core/vm::TestSetCodeFrom8298ContextsCreationAndRollback'], 'D35.4': ['github.com/eth2030/eth2030/core/vm::TestSetCodeFrom8298LoadedCodeReentryAndPersistentAvailability', 'github.com/eth2030/eth2030/bal::TestCodeAdoption8298CanonicalWireAndAvailability', 'github.com/eth2030/eth2030/core/execution::TestMigration8298NativeDelegationAdoptionRetirementAndFrameControl']}, 'HGT-036': {'D36.1': ['github.com/eth2030/eth2030/txpool::TestMigration8298AssertionJournalsPreserve8151SignerClass', 'github.com/eth2030/eth2030/core/vm::TestEIP8151RecoveryCodeClassGasAndUnpaidAccess', 'github.com/eth2030/eth2030/core/vm::TestEIP8151CallFamiliesAndOuterRevert'], 'D36.2': ['github.com/eth2030/eth2030/core/execution::TestMigration8298NativeDelegationAdoptionRetirementAndFrameControl', 'github.com/eth2030/eth2030/txpool::TestMigration8298PoolEvictsRetiredOriginsAndAllowsFrames'], 'D36.3': ['github.com/eth2030/eth2030/txpool::TestMigration8298ProxyDependencyLimitReplacementEvictionAndReorg'], 'D36.4': ['github.com/eth2030/eth2030/core/vm::TestMigration8151PermitMultisigAndFaultProofAuthorization']}, 'HGT-037': {'D37.1': ['github.com/eth2030/eth2030/core/execution::TestAssertion7906WholeBodyRollbackAndPrefixSettlement', 'github.com/eth2030/eth2030/core/execution::TestAssertion7906CaughtChildApprovalIsOrdinaryStaticFailure', 'github.com/eth2030/eth2030/core/execution::TestAssertion7906FailureWithoutPayerInvalidatesTransaction'], 'D37.2': ['github.com/eth2030/eth2030/core/vm::TestAssertion7906AllSelectorsAndFullWidthBounds', 'github.com/eth2030/eth2030/core/execution::TestAssertion7906PrestateNetDiffSortedAndLazy', 'github.com/eth2030/eth2030/core/execution::TestAssertion7906ActualEscrowAndPrefixPrestateTrace'], 'D37.3': ['github.com/eth2030/eth2030/core/vm::TestAssertion7906EventCopyBoundsAndGas'], 'D37.4': ['github.com/eth2030/eth2030/core/execution::TestAssertion7906PaymasterTokenFalseReturnRetryAndAtomic', 'github.com/eth2030/eth2030/core/execution::TestAssertion7906TraceOpcodesExecuteAgainstPaidPrefix']}}
PROFILES = {'direct8504': {'scheme': 16, 'witness_bytes': 7740, 'verify_permutations': 515, 'per_triple_gas': 3000, 'per_dependency_witness_gas': 495360, 'activation': 'EIP8504Time; nil by default', 'transport': 'hgtp/1; explicit native namespace; no official eth assignment', 'aggregation': False}, 'adoption8298': {'activation': 'EIP8298Time; nil by default', 'source_created_this_tx': 'reject', 'code_change': 'hash-only third element', 'proxy_pending_share': 8}, 'assert7906': {'activation': 'EIP7906Time; nil by default', 'mode': 3, 'txtrace': 182, 'eventdatacopy': 183, 'rollback': 'entire body to post-payer prefix', 'automatic_token_compensation': False}}
GATES = ['8504 provisional parameters and official transport/Engine/CL assignments', '8288 AGGREGATED_VK and DEPS_HASH remain unspecified; direct profile only', '8151 ecosystem/rollup compatibility qualification and canonical-paymaster upstream adoption', '7807/8504 execution-block identity composition remains unspecified', 'Official cross-client/devnet and later Group E-J release qualification', '8504 omitted inclusion-list dependency validity requires an upstream witness-bearing IL composition']


def validate_manifest(root, manifest):
    need(manifest['version'] == 1, 'unknown D manifest version')
    need(manifest['profiles'] == PROFILES, 'D pinned parameters, semantics or activation weakened')
    need(manifest['public_gates'] == GATES, 'unresolved public release gate removed')
    sources = manifest['normative_sources']
    need({8504, 8288, 8298, 8151, 7906, 8141, 8250, 8037, 8038, 7928, 6780, 3541, 3607, 7702, 7807} <= {s['eip'] for s in sources}, 'D normative source omitted')
    artifacts = {'specs/hegota/group-d.json', 'tools/hegota/check_d.py', 'tools/hegota/test_group_d.py'}
    for s in sources:
        raw = (root/s['path']).read_bytes()
        need(re.fullmatch(r'[0-9a-f]{40}', s['revision']), 'mutable D source revision')
        need(hashlib.sha256(raw).hexdigest() == s['sha256'], 'D source changed: '+s['path'])
        need(hashlib.sha1(b'blob '+str(len(raw)).encode()+b'\0'+raw).hexdigest() == s['blob'], 'D source blob mismatch')
        artifacts.add(s['path'])
    rows = manifest['tasks']
    need(len(rows) == 8 and {r['id'] for r in rows} == set(REQUIRED), 'D task omitted/duplicated')
    tests = set()
    for row in rows:
        criteria = row['criteria']
        need(row['owner'] and row['scope'], 'D ownership/scope missing')
        need(len(criteria) == 4 and {c['id'] for c in criteria} == set(REQUIRED[row['id']]), 'D criterion omitted/duplicated')
        for c in criteria:
            need(c['statement'] and c['clauses'] and c['files'] and c['tests'], 'D clause/file/test mapping missing')
            need(set(REQUIRED[row['id']][c['id']]) <= set(c['tests']), 'D mandatory boundary consumer removed: '+c['id'])
            artifacts.update(c['files'])
            for name in c['tests']:
                package, test = name.split('::')
                need(package.startswith(MODULE), 'D non-native evidence')
                directory = root/'pkg'/package[len(MODULE):]
                owners = [p for p in directory.glob('*_test.go') if re.search(r'^func '+re.escape(test.split('/')[0])+r'\(', p.read_text(), re.M)]
                need(owners, 'D native test missing: '+name)
                artifacts.update(str(p.relative_to(root)) for p in owners)
                tests.add(name)
    artifacts.update(manifest['additional_artifacts'])
    need(all((root/p).is_file() for p in artifacts), 'D artifact missing')
    return tests, artifacts


def verify_report(report, manifest, tests, hashes, digest):
    need(report['source_fingerprint_sha256'] == digest and report['artifacts'] == hashes, 'stale D source evidence')
    need(report['tasks'] == 8 and report['criteria'] == 32, 'D coverage count changed')
    need(report['public_complete'] is False and report['public_gates'] == manifest['public_gates'], 'D public conformance claimed across open gates')
    need(report['selected_profile_consumers_passed'] is True and tests, 'D empty/unqualified native evidence')
    need(set(report['passed_tests']) == tests and all(v == 'pass' for v in report['passed_tests'].values()), 'D native consumer omitted/failed/skipped')
    run = report['run']
    need(run['named'].get('pass', 0) >= len(tests) and not run['named'].get('fail', 0), 'D named run incomplete/failed')
    need(run['packages'].get('pass', 0) > 0 and not run['packages'].get('fail', 0) and run['completed_packages'] == sum(run['packages'].values()), 'D package run incomplete/failed')
    need(re.fullmatch(r'[0-9a-f]{64}', run['log_sha256']), 'D run lacks log identity')


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--log', type=Path)
    parser.add_argument('--write', action='store_true')
    parser.add_argument('--validate-only', action='store_true')
    args = parser.parse_args()
    manifest = json.loads((ROOT/'specs/hegota/group-d.json').read_text())
    tests, artifacts = validate_manifest(ROOT, manifest)
    hashes, digest = fingerprint(ROOT, artifacts)
    path = ROOT/'evidence/hegota-d/verification.json'
    if args.write:
        need(args.log is not None, '--write requires an actual complete native Go JSON log')
        passed, run = read_events(args.log)
        need(tests <= passed, 'unexecuted D consumers: '+str(sorted(tests-passed)))
        report = {'date': '2026-11-27', 'timezone': 'Asia/Shanghai', 'source_revision': subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip(), 'tasks':8, 'criteria':32, 'selected_profile_consumers_passed':True, 'public_complete':False, 'public_gates':manifest['public_gates'], 'artifacts':hashes, 'source_fingerprint_sha256':digest, 'passed_tests':{name:'pass' for name in sorted(tests)}, 'run':run}
        path.parent.mkdir(parents=True,exist_ok=True)
        path.write_text(json.dumps(report,indent=2)+'\n')
    if not args.validate_only:
        verify_report(json.loads(path.read_text()), manifest, tests, hashes, digest)
    print(json.dumps({'result':'pass','tasks':8,'criteria':32,'native_consumers':len(tests),'public_complete':False}))


if __name__ == '__main__':
    main()
