#!/usr/bin/env python3
"""Check every Group B criterion against current sources and executed Go tests."""
import argparse
import collections
import hashlib
import json
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
REQUIRED = {
 'HGT-002': {'codec', 'hashes', 'boundaries'},
 'HGT-003': {'scope_identity', 'nonce_payment', 'nested_rollback'},
 'HGT-010': {'parameters', 'visibility_bounds', 'non_frame'},
 'HGT-011': {'budgets_settlement', 'pool_isolation', 'block_dimensions'},
 'HGT-012': {'state_ownership', 'approval_batches', 'transient_warmness'},
 'HGT-013': {'native_route', 'build_import'},
 'HGT-014': {'encoding', 'logs_indices', 'invalid_effects'},
 'HGT-015': {'bal', 'invalid_commitments', 'bounds_system_updates'},
 'HGT-016': {'raw_persistence', 'restart_history'},
 'HGT-017': {'keys_codec', 'consumption_bal', 'rollback_create', 'deployment'},
 'HGT-018': {'domains_contract', 'slot_window', 'overwrite_reorg', 'deployment'},
 'HGT-019': {'schemes', 'signing_commitments', 'default_accounts'},
}
MODULE = 'github.com/eth2030/eth2030/'
CRITICAL_BOUNDARIES = {
 ('HGT-011', 'pool_isolation'): {'TestFrame8141DelegatedCallGasAndForwarding'},
 ('HGT-015', 'bal'): {'TestFrame8141DirectPrecompileBALGasBoundary',
                     'TestFrame8141SstoreBALReadSurvivesPostStateFailure',
                     'TestFrame8141SelfdestructBALPreStateBoundary',
                     'TestFrame8141DelegationBALBeforeFailedChildChecks'},
}

def need(ok, message):
    if not ok:
        raise ValueError(message)

def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

def validate_manifest(root, manifest):
    need(manifest['version'] == 1, 'unknown B manifest version')
    need(manifest['profiles']['8250-nonce-only-delta-isolated-v1']['public_activation'] is False,
         'D01 profile must stay isolated')
    need(manifest['profiles']['8250-nonce-only-delta-isolated-v1']['decision'] == 'D01-open',
         'D01 cannot be silently closed')
    sources = manifest['normative_sources']
    need({8141, 8250, 8272, 8037, 8038, 7843, 7928, 7934, 7702} <= {s['eip'] for s in sources},
         'missing normative source')
    for source in sources:
        need(sha(root/source['path']) == source['sha256'], 'normative source changed: '+source['path'])
    rows = manifest['tasks']
    need(len(rows) == len(REQUIRED) and {r['id'] for r in rows} == set(REQUIRED), 'B task omission/duplicate')
    tests, artifacts = set(), {'tools/hegota/check_b.py', 'tools/hegota/test_group_b.py', 'tools/hegota/b_vectors.py', 'specs/hegota/group-b.json'}
    for row in rows:
        criteria = row['criteria']
        need(len(criteria) == len(REQUIRED[row['id']]) and {c['id'] for c in criteria} == REQUIRED[row['id']],
             row['id']+': criterion omission/duplicate')
        for criterion in criteria:
            label = row['id']+'/'+criterion['id']
            critical = {MODULE+'core/execution::'+n for n in CRITICAL_BOUNDARIES.get((row['id'], criterion['id']), set())}
            need(critical <= set(criterion['tests']), label+': mandatory boundary regression omitted')
            need(criterion['clauses'] and criterion['statement'] and criterion['files'] and criterion['tests'],
                 label+': no clause/file/executed-test mapping')
            for path in criterion['files']:
                need((root/path).is_file(), label+': missing file '+path)
                artifacts.add(path)
            for test in criterion['tests']:
                package, name = test.split('::')
                need(package.startswith(MODULE), label+': non-native evidence')
                files = list((root/'pkg'/package[len(MODULE):]).glob('*_test.go'))
                owners = [p for p in files if re.search(r'^func '+re.escape(name.split('/')[0])+r'\(', p.read_text(), re.M)]
                need(owners, label+': missing native test '+test)
                artifacts.update(str(p.relative_to(root)) for p in owners)
                tests.add(test)
    artifacts.update(s['path'] for s in sources)
    return tests, artifacts

def read_events(path):
    started, ended, passed = set(), set(), set()
    named, packages = collections.Counter(), collections.Counter()
    for line in path.read_text().splitlines():
        event = json.loads(line)
        package, test = event.get('Package', ''), event.get('Test', '')
        if event['Action'] == 'start':
            started.add(package)
        if event['Action'] in ('pass', 'skip', 'fail'):
            (named if test else packages)[event['Action']] += 1
            if not test:
                ended.add(package)
            elif event['Action'] == 'pass':
                passed.add(package+'::'+test)
    need(started and started == ended, 'incomplete Go run')
    need(not named['fail'] and not packages['fail'], 'failed Go run')
    need(sum(packages.values()) == len(ended), 'duplicate Go package completions')
    return passed, {'log_sha256': sha(path), 'named': dict(named), 'packages': dict(packages), 'completed_packages': len(ended)}

def fingerprint(root, artifacts):
    hashes = {path: sha(root/path) for path in sorted(artifacts)}
    return hashes, hashlib.sha256(json.dumps(hashes, sort_keys=True).encode()).hexdigest()

def verify_report(report, tests, hashes, digest):
    need(report['source_fingerprint_sha256'] == digest and report['artifacts'] == hashes, 'stale B source evidence')
    need(set(report['passed_tests']) == tests and all(v == 'pass' for v in report['passed_tests'].values()),
         'B native test evidence omitted/failed/skipped')
    need(report['tasks'] == 12 and report['criteria'] == sum(map(len, REQUIRED.values())), 'B coverage count differs')
    run = report['run']
    need(run['named'].get('pass', 0) >= len(tests) and not run['named'].get('fail', 0)
         and run['packages'].get('pass', 0) > 0 and not run['packages'].get('fail', 0)
         and run['completed_packages'] == sum(run['packages'].values()), 'B run is empty, incomplete or failed')

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--log', type=Path)
    parser.add_argument('--write', action='store_true')
    args = parser.parse_args()
    manifest = json.loads((ROOT/'specs/hegota/group-b.json').read_text())
    tests, artifacts = validate_manifest(ROOT, manifest)
    hashes, digest = fingerprint(ROOT, artifacts)
    report_path = ROOT/'evidence/hegota-b/verification.json'
    if args.write:
        need(args.log is not None, '--write requires an actual Go JSON log')
        passed, summary = read_events(args.log)
        need(tests <= passed, 'unexecuted B tests: '+str(sorted(tests-passed)))
        report = {'date': '2026-11-26', 'timezone': 'Asia/Shanghai', 'tasks': 12,
                  'criteria': sum(map(len, REQUIRED.values())), 'run': summary,
                  'source_fingerprint_sha256': digest, 'artifacts': hashes,
                  'passed_tests': {test: 'pass' for test in sorted(tests)},
                  'scope': 'Selected base 8141 native execution; isolated 8250 nonce-only delta; ordinary 8272 contract. D01 and Group C public pool remain separate.'}
        report_path.write_text(json.dumps(report, indent=2)+'\n')
    report = json.loads(report_path.read_text())
    verify_report(report, tests, hashes, digest)
    print(json.dumps({'result': 'pass', 'tasks': 12, 'criteria': report['criteria'], 'native_tests': len(tests), 'D01': 'open; isolated profile only'}))

if __name__ == '__main__':
    main()
