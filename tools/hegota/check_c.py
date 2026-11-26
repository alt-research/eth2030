#!/usr/bin/env python3
"""Verify every Group C selected-profile criterion against native execution.

Unresolved public specification gates are preserved, never counted as activated.
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
 'rpc/types::TestFrameCDerivedReceiptStatusIsNotConsensusRLP',
 'rpc/ethapi::TestFrameCPendingAndMinedQueryLocations',
 'node::TestFrameCNativePeersPoolBuildSimulationAndRestart',
}

def validate_manifest(root, manifest):
    need(manifest['version'] == 1, 'unknown C manifest version')
    need(manifest['profiles']['8250-delta'] == {'public_activation': False, 'decision': 'D01-open'}, 'D01 public gate changed')
    template = manifest['profiles']['paymaster-v1']
    need(template['public_default'] is False and template['upstream_canonical'] is False, 'local template promoted to upstream canonical code')
    need(set(manifest['public_gates']) == {'D01', 'PAYMASTER-CODE'} and all(manifest['public_gates'].values()), 'public specification gate omitted')
    need({8141, 8250, 8272, 7594, 2718, 1474} <= {s['eip'] for s in manifest['normative_sources']}, 'missing C normative source')
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
