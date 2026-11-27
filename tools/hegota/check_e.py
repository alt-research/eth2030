#!/usr/bin/env python3
"""Source-bound Group E coverage. Completion labels cannot replace native evidence."""
import argparse
import hashlib
import json
import re
import subprocess
from pathlib import Path
from check_b import fingerprint, need, read_events, sha

ROOT = Path(__file__).resolve().parents[2]
MODULE = 'github.com/eth2030/eth2030/'
REQUIRED = {'HGT-040': {'E40.1': ['github.com/eth2030/eth2030/focil/heze::TestGroupEHezeIndependentSSZRootsAndDomain'],
             'E40.2': ['github.com/eth2030/eth2030/focil/heze::TestGroupEHezeBranchTimelinessEquivocationAndOwnership'],
             'E40.3': ['github.com/eth2030/eth2030/focil/heze::TestGroupEHezePayloadForkchoiceAndOptimisticTransitions'],
             'E40.4': ['github.com/eth2030/eth2030/focil::TestGroupEVOPSAccessCodeAndClaimedIndexActivationGate']},
 'HGT-041': {'E41.1': ['github.com/eth2030/eth2030/focil/heze::TestGroupEHezeIndependentSSZRootsAndDomain'],
             'E41.2': ['github.com/eth2030/eth2030/focil/heze::TestGroupEHezeBranchTimelinessEquivocationAndOwnership'],
             'E41.3': ['github.com/eth2030/eth2030/focil/heze::TestGroupEHezeAuthenticatedGossipAndFailureAtomicity'],
             'E41.4': ['github.com/eth2030/eth2030/focil/heze::TestGroupEHezeRequestsRetentionAndReauthenticatedRestart',
                       'github.com/eth2030/eth2030/focil/heze::TestGroupEHezeSizeAndOffsetBoundaries',
                       'github.com/eth2030/eth2030/focil/heze::TestGroupEHezeConcurrentQuotaAndCommittee',
                       'github.com/eth2030/eth2030/node::TestGroupEHezeReconfigureReauthenticatesMemoryAtomically']},
 'HGT-042': {'E42.1': ['github.com/eth2030/eth2030/engine::TestGroupEBogotaStrictAttributesOpaqueILAndBackendCapabilities'],
             'E42.2': ['github.com/eth2030/eth2030/engine::TestGroupELighthouseProgressiveSSZRejectsInvalidOffsets'],
             'E42.3': ['github.com/eth2030/eth2030/engine::TestGroupEBogotaSatisfactionStatusNullability'],
             'E42.4': ['github.com/eth2030/eth2030/node::TestBogotaDeferredWireAcrossRestart']},
 'HGT-043': {'E43.1': ['github.com/eth2030/eth2030/core/block::TestGroupEBuilderReverseNonceILCascade'],
             'E43.2': ['github.com/eth2030/eth2030/core/block::TestGroupEBuilderPoolFundingReopensILAndDeduplicates'],
             'E43.3': ['github.com/eth2030/eth2030/node/backend::TestGroupEEngineBuildReplacementOwnershipDeadlineAndClose'],
             'E43.4': ['github.com/eth2030/eth2030/core/block::TestGroupEBuilderCancellationIsNotAnOmissionExcuse']},
 'HGT-044': {'E44.1': ['github.com/eth2030/eth2030/focil::TestGroupEProfile1AuthenticatedAccountsWithoutExecution'],
             'E44.2': ['github.com/eth2030/eth2030/focil::TestGroupEProfile1InvalidSignatureFeeCodeNonceAndResources',
                       'github.com/eth2030/eth2030/core/execution::TestGroupETransactionEndIncludesPreSystemsAndExcludesPostSystems',
                       'github.com/eth2030/eth2030/node::TestGroupEWithdrawalDoesNotFundSameBlockOmission',
                       'github.com/eth2030/eth2030/node::TestGroupEPostSystemCannotChangeFrameOmissionVerdict'],
             'E44.3': ['github.com/eth2030/eth2030/core/types::TestGroupEExactAppendRLPAllPrefixAndGasBoundaries'],
             'E44.4': ['github.com/eth2030/eth2030/focil::TestGroupEProfile1OutsideProfilesAndUnavailableContext',
                       'github.com/eth2030/eth2030/core/block::TestGroupEBuilderExactRLPBudgetBoundaryAndRejectedAppendRollback',
                       'github.com/eth2030/eth2030/node::TestGroupEBaseFOCILBlobValidityAvailabilityAndSameIDRetry']},
 'HGT-045': {'E45.1': ['github.com/eth2030/eth2030/node::TestGroupEHezeGossipNativeEngineForkchoiceAndRestart'],
             'E45.2': ['github.com/eth2030/eth2030/node::TestCanonicalFrame8141ThroughEngineAndRestart'],
             'E45.3': ['github.com/eth2030/eth2030/node::TestBogotaDeferredWireAcrossRestart',
                       'github.com/eth2030/eth2030/core/chain::TestBogotaPayloadStorageRetainsFinalizedAndPendingViews',
                       'github.com/eth2030/eth2030/node::TestGroupEWithdrawalDoesNotFundSameBlockOmission',
                       'github.com/eth2030/eth2030/node::TestGroupEPostSystemCannotChangeFrameOmissionVerdict'],
             'E45.4': ['github.com/eth2030/eth2030/engine::TestGroupEBogotaStrictAttributesOpaqueILAndBackendCapabilities']},
 'HGT-046': {'E46.1': ['github.com/eth2030/eth2030/focil::TestGroupEVOPSFourShapesExpiryAndNoMempoolBudgetCoupling'],
             'E46.2': ['github.com/eth2030/eth2030/focil::TestGroupEVOPSOccurrenceBudgetSignatureDebitAndShapeFailure'],
             'E46.3': ['github.com/eth2030/eth2030/focil::TestGroupEVOPSAccessCodeAndClaimedIndexActivationGate'],
             'E46.4': ['github.com/eth2030/eth2030/focil::TestGroupEVOPSAccessCodeAndClaimedIndexActivationGate']},
 'HGT-047': {'E47.1': ['github.com/eth2030/eth2030/node::TestGroupEHezeGossipNativeEngineForkchoiceAndRestart'],
             'E47.2': ['github.com/eth2030/eth2030/focil/heze::TestGroupEHezePayloadForkchoiceAndOptimisticTransitions'],
             'E47.3': ['github.com/eth2030/eth2030/focil/heze::TestGroupEHezeBranchTimelinessEquivocationAndOwnership'],
             'E47.4': ['github.com/eth2030/eth2030/focil/heze::TestGroupEHezePayloadForkchoiceAndOptimisticTransitions']},
 'HGT-048': {'E48.1': ['github.com/eth2030/eth2030/focil/heze::TestGroupEPQCanonicalCapacityVerifierAndAuthenticatedPipeline',
                       'github.com/eth2030/eth2030/node::TestGroupECanonicalPQILUsesAuthenticatedPoolWitness'],
             'E48.2': ['github.com/eth2030/eth2030/focil/heze::TestGroupEPQCanonicalCapacityVerifierAndAuthenticatedPipeline'],
             'E48.3': ['github.com/eth2030/eth2030/focil/heze::TestGroupEPQCanonicalCapacityVerifierAndAuthenticatedPipeline'],
             'E48.4': ['github.com/eth2030/eth2030/focil/heze::TestGroupEPQCanonicalCapacityVerifierAndAuthenticatedPipeline']}}
GATES = ['8369 is Informational; Profile 2 enforcement requires a Standards Track extension',
 '8369 claim encoding/commitment/limits, canonical diffs, replay/code-load bounds, final '
 'slots/budgets and ePBS duties remain undefined',
 '8504 omitted-IL witness transport and validity composition remain unspecified',
 'External Heze CL gossipsub/reqresp/fork-digest and ePBS devnet qualification belongs to '
 'HGT-070/080; no public activation claimed']
SOURCE_PINS = [('refs/EIPs/EIPS/eip-4895.md',
  'ade9e9cab7e148bba4cd293f916a4a189cd7a8aa',
  '81af3b60b632bc9c03513d1d137f25410e3f4d34'),
 ('refs/EIPs/EIPS/eip-7002.md',
  'ade9e9cab7e148bba4cd293f916a4a189cd7a8aa',
  '31863eaebb637084d7e706f6e96e7e2ec53b7716'),
 ('refs/EIPs/EIPS/eip-7928.md',
  'ade9e9cab7e148bba4cd293f916a4a189cd7a8aa',
  '3c4d4ad300e4946303db45710bba02cfb836f9eb'),
 ('refs/EIPs/EIPS/eip-7934.md',
  'ade9e9cab7e148bba4cd293f916a4a189cd7a8aa',
  'c18cb0def4fa9ef731f358912f228d30470a85cd'),
 ('refs/EIPs/EIPS/eip-8141.md',
  'ade9e9cab7e148bba4cd293f916a4a189cd7a8aa',
  '9bc38a368881fa52e1a1a9970274e6511874d13f'),
 ('refs/EIPs/EIPS/eip-8250.md',
  'ade9e9cab7e148bba4cd293f916a4a189cd7a8aa',
  '10bd1d84c9534c9db65f9ccbffcf8d5902b9c1ae'),
 ('refs/EIPs/EIPS/eip-8272.md',
  'ade9e9cab7e148bba4cd293f916a4a189cd7a8aa',
  '0acc21ad8359be83c0929c7cc70e45326be8acff'),
 ('specs/hegota/upstream/2026-11-27/group-d/eip-8504.md',
  '96da3c17047ad31608c40bfe75049db78dc4417d',
  'd5b17df22eaa7a1b705985e76d41378ed12ec712'),
 ('specs/hegota/upstream/2026-11-27/group-e/eip-7805.md',
  '68db81b0a75edb0e16d8f181241f994875eba52d',
  '0a3955d9e8772b7666f560402b2d81ae032a3d06'),
 ('specs/hegota/upstream/2026-11-27/group-e/eip-7916.md',
  '68db81b0a75edb0e16d8f181241f994875eba52d',
  '8c6869b3b516f895a3b1bfde7196d5ae4096a49e'),
 ('specs/hegota/upstream/2026-11-27/group-e/eip-8369.md',
  '68db81b0a75edb0e16d8f181241f994875eba52d',
  'f76c4c162c40865ab48b7c8f7007953d72177841'),
 ('specs/hegota/upstream/2026-11-27/group-e/engine-bogota.md',
  '9e85b16c7fb54a838c9a612842c138c2bdf9bc8d',
  'b7708522d8dfdf499ca4af27d80babef526ef9e9'),
 ('specs/hegota/upstream/2026-11-27/group-e/heze/beacon-chain.md',
  '36888ee87b0faa6ad7804b017a0184e8e6e6a1c6',
  '968253fb36b755386b36a7818266645f88a6d02c'),
 ('specs/hegota/upstream/2026-11-27/group-e/heze/builder.md',
  '36888ee87b0faa6ad7804b017a0184e8e6e6a1c6',
  '362da2affbc53286a7fd78a41478bb65d952c10d'),
 ('specs/hegota/upstream/2026-11-27/group-e/heze/fork-choice.md',
  '36888ee87b0faa6ad7804b017a0184e8e6e6a1c6',
  '5d2a62ba543eb6327bd24440822e2d4671430367'),
 ('specs/hegota/upstream/2026-11-27/group-e/heze/fork.md',
  '36888ee87b0faa6ad7804b017a0184e8e6e6a1c6',
  '7c7fa35fbb1d1615f7e72d8a6851495b57161943'),
 ('specs/hegota/upstream/2026-11-27/group-e/heze/inclusion-list.md',
  '36888ee87b0faa6ad7804b017a0184e8e6e6a1c6',
  'b6f6e4e06a3fe1ae50ae71199318e6928f169835'),
 ('specs/hegota/upstream/2026-11-27/group-e/heze/optimistic-sync.md',
  '36888ee87b0faa6ad7804b017a0184e8e6e6a1c6',
  'a5b149a97634cb08c18e2914027b165e169cd5c8'),
 ('specs/hegota/upstream/2026-11-27/group-e/heze/p2p-interface.md',
  '36888ee87b0faa6ad7804b017a0184e8e6e6a1c6',
  '1f543491bfc9489be35bd6b341f903b8831821c1'),
 ('specs/hegota/upstream/2026-11-27/group-e/heze/validator.md',
  '36888ee87b0faa6ad7804b017a0184e8e6e6a1c6',
  '40b66dbf5a4c03da89eb75b0b07db4b792acc4ad')]


def validate_manifest(root, manifest):
    need(manifest['version'] == 1, 'unknown E manifest version')
    need(manifest['public_gates'] == GATES, 'unresolved public release gate removed')
    sources = manifest['normative_sources']
    need(len(sources) == len(SOURCE_PINS) and sorted((v['path'],v['revision'],v['blob']) for v in sources) == SOURCE_PINS, 'E pinned source omitted/duplicated/changed')
    artifacts = {'specs/hegota/group-e.json', 'tools/hegota/check_e.py', 'tools/hegota/test_group_e.py'}
    for s in sources:
        raw = (root/s['path']).read_bytes()
        need(re.fullmatch(r'[0-9a-f]{40}', s['revision']), 'mutable E source revision')
        need(hashlib.sha256(raw).hexdigest() == s['sha256'], 'E source changed: '+s['path'])
        need(hashlib.sha1(b'blob '+str(len(raw)).encode()+b'\0'+raw).hexdigest() == s['blob'], 'E source blob mismatch')
        artifacts.add(s['path'])
    rows = manifest['tasks']
    need(len(rows) == 9 and {r['id'] for r in rows} == set(REQUIRED), 'E task omitted/duplicated')
    tests = set()
    for row in rows:
        criteria = row['criteria']
        need(row['owner'] and row['scope'], 'E ownership/scope missing')
        need(len(criteria) == 4 and {c['id'] for c in criteria} == set(REQUIRED[row['id']]), 'E criterion omitted/duplicated')
        for c in criteria:
            need(c['statement'] and c['clauses'] and c['files'] and c['tests'], 'E clause/file/test mapping missing')
            need(set(REQUIRED[row['id']][c['id']]) <= set(c['tests']), 'E mandatory boundary consumer removed: '+c['id'])
            artifacts.update(c['files'])
            for name in c['tests']:
                package, test = name.split('::')
                need(package.startswith(MODULE), 'E non-native evidence')
                directory = root/'pkg'/package[len(MODULE):]
                owners = [p for p in directory.glob('*_test.go') if re.search(r'^func '+re.escape(test.split('/')[0])+r'\(', p.read_text(), re.M)]
                need(owners, 'E native test missing: '+name)
                artifacts.update(str(p.relative_to(root)) for p in owners)
                tests.add(name)
    artifacts.update(manifest['additional_artifacts'])
    need(all((root/p).is_file() for p in artifacts), 'E artifact missing')
    return tests, artifacts


def verify_report(report, manifest, tests, hashes, digest):
    need(report['source_fingerprint_sha256'] == digest and report['artifacts'] == hashes, 'stale E source evidence')
    need(report['tasks'] == 9 and report['criteria'] == 36, 'E coverage count changed')
    need(report['public_complete'] is False and report['public_gates'] == manifest['public_gates'], 'E public conformance claimed across open gates')
    need(report['selected_profile_consumers_passed'] is True and tests, 'E empty/unqualified native evidence')
    need(set(report['passed_tests']) == tests and all(v == 'pass' for v in report['passed_tests'].values()), 'E native consumer omitted/failed/skipped')
    run = report['run']
    need(run['named'].get('pass', 0) >= len(tests) and not run['named'].get('fail', 0), 'E named run incomplete/failed')
    need(run['packages'].get('pass', 0) > 0 and not run['packages'].get('fail', 0) and run['completed_packages'] == sum(run['packages'].values()), 'E package run incomplete/failed')
    need(re.fullmatch(r'[0-9a-f]{64}', run['log_sha256']), 'E run lacks log identity')


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--log', type=Path)
    parser.add_argument('--write', action='store_true')
    parser.add_argument('--validate-only', action='store_true')
    args = parser.parse_args()
    manifest = json.loads((ROOT/'specs/hegota/group-e.json').read_text())
    tests, artifacts = validate_manifest(ROOT, manifest)
    hashes, digest = fingerprint(ROOT, artifacts)
    path = ROOT/'evidence/hegota-e/verification.json'
    if args.write:
        need(args.log is not None, '--write requires an actual complete native Go JSON log')
        passed, run = read_events(args.log)
        need(tests <= passed, 'unexecuted E consumers: '+str(sorted(tests-passed)))
        report = {'date': '2026-11-27', 'timezone': 'Asia/Shanghai', 'source_revision': subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip(), 'tasks':9, 'criteria':36, 'selected_profile_consumers_passed':True, 'public_complete':False, 'public_gates':manifest['public_gates'], 'artifacts':hashes, 'source_fingerprint_sha256':digest, 'passed_tests':{name:'pass' for name in sorted(tests)}, 'run':run}
        path.parent.mkdir(parents=True,exist_ok=True)
        path.write_text(json.dumps(report,indent=2)+'\n')
    if not args.validate_only:
        verify_report(json.loads(path.read_text()), manifest, tests, hashes, digest)
    print(json.dumps({'result':'pass','tasks':9,'criteria':36,'native_consumers':len(tests),'public_complete':False}))


if __name__ == '__main__':
    main()
