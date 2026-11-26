#!/usr/bin/env python3
"""Verify every Group C selected-profile criterion against native execution.

Explicit composition decisions and unresolved upstream gates are checked independently.
"""
import argparse
import hashlib
import json
import re
import subprocess
from pathlib import Path
from check_b import read_events, fingerprint, need, sha

ROOT = Path(__file__).resolve().parents[2]
MODULE = 'github.com/eth2030/eth2030/'
REQUIRED = {f'HGT-{g:03}': {f'C{g}.{i}' for i in range(1, 5)} for g in range(20, 30)}
MANDATORY = {
 'core/execution::TestFrameCAllSixteenPrefixesExecuteAndCaptureDependencies',
 'core/execution::TestFrameCPublicFactoryFundingDependencyScope',
 'core/execution::TestFrameCPublicTransientStorageScope',
 'core/types::TestFrame8141PublicSixteenPrefixShapes',
 'core/types::TestFrame8141PublicRootGrammar',
 'core/types::TestFrameCPublicWorkBudgetEquality',
 'core/vm::TestFrameCPublicVerifierPermissionsAreExactAndTopLevel',
 'core/vm::TestFrameCSenderDefaultCodeCallException',
 'txpool::TestFrameCPublicRecentRootSlotAndReorg',
 'txpool::TestFrameCValidationDependencyCache',
 'txpool::TestFrameCFailedAdmissionAndByteReplacementAreAtomic',
 'txpool::TestFrameCLocalPaymasterRequiresExplicitProfile',
 'txpool::TestFrameCBuildContextAndUnpaidRetryBound',
 'txpool::TestFrameCConfiguredReplacementAndSidecarBytes',
 'txpool::TestFrameCZeroFeeBudgetDoesNotPanic',
 'txpool::TestFrameCExposureUsesValidatedPayPrefixClassification',
 'txpool::TestFrameCExpiryQueuesReplacementAndRemoval',
 'txpool::TestFrameCCaughtInvalidOpcodeCannotApprovePublicPrefix',
 'txpool::TestFrameCIsolatedKeyedPolicyIdentityDependenciesAndExhaustion',
 'txpool::TestFrameCIsolatedKeyedSponsorCodeAdoptionCapsPaymaster',
 'txpool::TestFrameCIsolatedKeyedCachedDimensionalLimits',
 'core/block::TestFrameCBuilderOrdersSenderHeadsAtBlobFee',
 'eth::TestFrameCAnnouncementFetchMetadataAndBounds',
 'eth::TestFrameCEvictedAnnouncementSuppression',
 'eth::TestFrameCStructuredReceiptThroughHandler',
 'node/backend::TestFrameCSimulationAuthenticationOverridesAndCancellation',
 'node/backend::TestFrameCSimulationTraceLimitIsExplicit',
 'rpc/types::TestFrameCDerivedReceiptStatusIsNotConsensusRLP',
 'rpc/ethapi::TestFrameCPendingAndMinedQueryLocations',
 'node::TestFrameCNativePeersPoolBuildSimulationAndRestart',
 'node::TestFrame8250NativePeersPoolBuildSimulationAndRestart',
 'core/block::TestFrame8250BuildImportReorgAndRestart',
 'core/block::TestFrame8250BuilderOrderingKeepsKeyedDomainsIndependent',
 'core/config::TestFrameExtensionActivationAndCompatibility',
 'core/types::TestFrame8250PublicCopyAndUint256ChainID',
 'core/types::TestFrame8250DeltaIndependentWireGoldens',
 'core/execution::TestFrame8250PublicExecutionSchemaBoundary',
 'txpool::TestFrame8250PublicIdentityDependenciesAndForkEviction',
 'core/vm::TestEIP8151RecoveryCodeClassGasAndUnpaidAccess',
 'core/vm::TestEIP8151CallFamiliesAndOuterRevert',
 'txpool::TestFrame8151SignerClassesReorgAndSeparateHelperDependency',
 'txpool::TestFrame8151ReservedAndOtherDigestShares',
 'txpool::TestFrame8151ActivationInvalidatesPrefixCache',
 'node::TestLoadGenesisFile_ScheduledForksRoundTrip',
 'txpool::TestFrameCRevalidationDropsInvalidBeforeSharedExposure',
 'txpool::TestFrameCSharedPayerRevalidationEvictsByPolicy',
 'txpool::TestFrameCSignerRevalidationUsesOneHeadSnapshot',
 'core/block::TestFrameCBlobAndOrdinarySenderOrdering',
 'txpool::TestFrame8151PayRecoveriesCountDistinctPaymasters',
 'rpc/ethapi::TestFrame8250TxPoolMapDoesNotAliasLegacyNonce',
}

PAYMASTER_REQUIRED = {
	'core/vm::TestPR12041PaymasterRecognitionIsFrameBounded',
 'core/framepay::TestPR12041ReproducesUpstreamAssembly',
 'core/framepay::TestPR12041LayoutAndProfileIsolation',
 'core/execution::TestPR12041NativePaymentAuthorization',
 'core/execution::TestPR12041NativeDeployment',
 'core/execution::TestPR12041NativeTreasuryAndAdministration',
 'core/execution::TestPR12041AuthorizationBindsEntireTransaction',
 'core/execution::TestPR12041PublicPolicyDoesNotChangeConsensus',
 'core/execution::TestPR12041OrdinaryCallerAdministration',
 'txpool::TestPR12041PoolExactSelectionAndUsage',
 'txpool::TestPR12041PoolWithdrawalReservationAndLifecycle',
 'txpool::TestPR12041PoolTracksAllSlotsAndRotation',
 'txpool::TestPR12041PoolReplacementMovesExposure',
 'txpool::TestPR12041BuildRechecksWithdrawalAvailability',
 'node::TestPR12041NativePeersPoolBuildSimulationAndRestart',
 'node::TestPR12041KeyedNativePeersPoolBuildSimulationAndRestart',
 'cmd/eth2030::TestPR12041FlagSelection',
 'cmd/send-frame::TestMakeFramePR12041Interface',
}
MANDATORY |= PAYMASTER_REQUIRED

def validate_manifest(root, manifest):
    need(manifest['version'] == 2, 'unknown C manifest version')
    need(manifest['profiles']['8250-delta']['public_activation'] is False, 'independent oracle promoted to public activation')
    need(manifest['profiles']['8250-public'] == {'public_activation': True, 'activation': 'EIP8250Time; explicit, nil default', 'decision': 'D01-resolved-two-fee', 'fees': ['max_fee', 'max_priority_fee_per_gas']}, 'unqualified 8250 composition')
    decisions = json.loads((root/'specs/hegota/profiles.json').read_text())
    decision = next(d for d in decisions['decisions'] if d['id'] == 'D01')
    need(decision['status'] == 'resolved_selected_native_composition' and decision['resolution_record'] == 'docs/plans/hegota-c/COMPOSITION.md' and decision['previous_decisions'] and {'https://github.com/ethereum/EIPs/pull/11968', 'https://github.com/ethereum/EIPs/pull/12414'} <= set(decision['authority']), 'D01 decision/history/authority missing')
    need(manifest['profiles']['8151-conditional'] == {'activation': 'EIP8151Time; explicit, nil default', 'signer_dependencies': 'raw-code class, separate helper hash', 'local_shares': {'canonical_default':64, 'other_default':8, 'pay_counting':'distinct paymasters, separate from other frames'}}, 'conditional 8151 profile changed without qualification')
    template = manifest['profiles']['paymaster-v1']
    need(template['public_default'] is False and template['upstream_canonical'] is False, 'local template promoted to upstream canonical code')
    need(manifest['profiles']['paymaster-12041'] == {
        'public_default': False, 'upstream_canonical': False,
        'selection': '--experimental-frame-paymaster-12041',
        'revision': '4a98fcefa5c6f02a25289bf55b9d92902d02b4c6',
        'runtime_bytes': 355,
        'code_hash': '0xda42f0d11838c4c0c3129b8b8e93e9718127ad6b315e517e1088125707c4d45c',
        'storage': ['signer','pending_withdrawal_amount','pending_unlock_time','pending_new_signer'],
        'delay_seconds': 86400, 'local_pay_execution_limit': 15000,
        'authorization': 'protocol-verified index 1, non-ARBITRARY, empty msg',
        'ecRecover_dependencies': False,
    }, 'unqualified paymaster proposal/version/authority')
    need(set(manifest['public_gates']) == {'PAYMASTER-CODE'} and all(manifest['public_gates'].values()), 'public specification gate omitted')
    need({8141, 8250, 8272, 7594, 2718, 1474, 8151} <= {s['eip'] for s in manifest['normative_sources']}, 'missing C normative source')
    artifacts = {'specs/hegota/group-c.json', 'tools/hegota/check_b.py', *manifest['additional_artifacts']}
    for source in manifest['normative_sources']:
        need(sha(root/source['path']) == source['sha256'], 'C normative source changed: '+source['path'])
        artifacts.add(source['path'])
    rows = manifest['tasks']
    need(len(rows) == 10 and {r['id'] for r in rows} == set(REQUIRED), 'C task omission/duplicate')
    tests = set()
    for row in rows:
        criteria = row['criteria']
        need(len(criteria) == 4 and {c['id'] for c in criteria} == REQUIRED[row['id']], 'C criterion omission/duplicate: '+row['id'])
        for criterion in criteria:
            label = row['id']+'/'+criterion['id']
            need(criterion['statement'] and criterion['clauses'] and criterion['files'] and criterion['tests'] and criterion['scope'], label+': missing clause/code/consumer/scope')
            artifacts.update(criterion['files'])
            for test in criterion['tests']:
                package, name = test.split('::')
                need(package.startswith(MODULE), label+': non-native consumer')
                owners = [p for p in (root/'pkg'/package.removeprefix(MODULE)).glob('*_test.go') if re.search(r'^func '+re.escape(name.split('/')[0])+r'\(', p.read_text(), re.M)]
                need(owners, label+': no executable consumer: '+test)
                artifacts.update(str(p.relative_to(root)) for p in owners)
                tests.add(test)
    need({MODULE+n for n in MANDATORY} <= tests, 'mandatory C boundary consumer omitted')
    for path in artifacts:
        need((root/path).is_file(), 'C artifact missing: '+path)
    return tests, artifacts

def verify_report(report, manifest, tests, hashes, digest):
    need(report['source_fingerprint_sha256'] == digest and report['artifacts'] == hashes, 'stale C source evidence')
    need(report['tasks'] == 10 and report['criteria'] == 40, 'C coverage count differs')
    need(report['public_complete'] is False and report['public_gates'] == manifest['public_gates'], 'unresolved C public gates misrepresented')
    need(report['selected_profile_consumers_passed'] is True, 'C mechanisms not qualified')
    need(report['passed_tests'] == {t: 'pass' for t in tests}, 'C required consumer skipped/omitted/failed')
    run = report['run']
    need(not run['named'].get('fail') and not run['packages'].get('fail') and run['named'].get('pass', 0) >= len(tests) and run['packages'].get('pass', 0) > 0 and run['completed_packages'] == sum(run['packages'].values()), 'C run empty/incomplete/failed')

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--log', type=Path)
    parser.add_argument('--write', action='store_true')
    args = parser.parse_args()
    manifest = json.loads((ROOT/'specs/hegota/group-c.json').read_text())
    tests, artifacts = validate_manifest(ROOT, manifest)
    hashes, digest = fingerprint(ROOT, artifacts)
    path = ROOT/'evidence/hegota-c/verification.json'
    if args.write:
        need(args.log is not None, '--write needs actual native Go JSON log')
        passed, run = read_events(args.log)
        need(tests <= passed, 'unexecuted C consumers: '+str(sorted(tests-passed)))
        path.parent.mkdir(parents=True, exist_ok=True)
        report = {'date': '2026-11-26', 'timezone': 'Asia/Shanghai', 'source_revision': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip(), 'tasks': 10, 'criteria': 40, 'selected_profile_consumers_passed': True, 'public_complete': False, 'public_gates': manifest['public_gates'], 'artifacts': hashes, 'source_fingerprint_sha256': digest, 'passed_tests': {t: 'pass' for t in sorted(tests)}, 'run': run}
        path.write_text(json.dumps(report, indent=2)+'\n')
    report = json.loads(path.read_text())
    verify_report(report, manifest, tests, hashes, digest)
    print(json.dumps({'result': 'pass', 'tasks': 10, 'criteria': 40, 'native_consumers': len(tests), 'public_complete': False, 'public_gates': manifest['public_gates']}))

if __name__ == '__main__':
    main()
