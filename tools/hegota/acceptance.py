"""Validate criterion-level Group A evidence, independently of completion labels."""
import hashlib
import re


REQUIRED = {
    'HGT-001': {'source_pins', 'composition', 'reference_identity'},
    'HGT-004': {'native_route', 'wrapper_route', 'head_selection'},
    'HGT-005': {'legacy_boundaries', 'canonical_head_height', 'genesis_migration'},
    'HGT-006': {'differential', 'prerequisite_inventory', 'skip_inventory'},
    'HGT-007': {'encoding', 'approval', 'rollback', 'conditional_nonce_roots'},
    'HGT-008': {'bid_authentication', 'bls', 'pq', 'kzg', 'native_backend', 'il_validation', 'experimental'},
    'HGT-009': {'scope', 'update_history'},
}
COMMANDS = {'source_lock', 'profiles', 'reference_identity', 'runtime_manifest',
            'baseline', 'prerequisite_inventory', 'skip_inventory', 'semantic_oracles',
            'admission', 'update_history'}
MODULE = 'github.com/eth2030/eth2030/'


def need(value, message):
    if not value:
        raise ValueError(message)


def validate_reference(root, lock, profiles, baseline):
    active = lock['active_reference']
    version = re.search(r'github.com/ethereum/go-ethereum (v\S+)', (root/'pkg/go.mod').read_text()).group(1)
    need(active['version'] == version, 'active reference differs from go.mod')
    need(any(r['repo'] == 'ethereum/go-ethereum' and r['ref'] == version and r['sha'] == active['revision']
             for r in lock['repositories']), 'active reference revision is not locked')
    sums = (root/'pkg/go.sum').read_text().splitlines()
    need(f"github.com/ethereum/go-ethereum {version} {active['module_sum']}" in sums,
         'active reference module checksum differs')
    profile = next(p for p in profiles['profiles'] if p['id'] == 'native-history-v1')
    need(profile['reference_version'] == version and version in profile['oracle'] and
         f'go-ethereum@{version}/' in baseline['reference_backend'], 'profile/runner reference identity differs')


def validate_acceptance(root, register, matrix, evidence, runtime, checked_commands):
    rows = register['tasks']
    need(len(rows) == len(REQUIRED) and {t['id'] for t in rows} == set(REQUIRED), 'A task omission or duplicate')
    entries = matrix['tasks']
    need(len(entries) == len(REQUIRED) and {t['id'] for t in entries} == set(REQUIRED), 'A acceptance task omission or duplicate')
    recorded = evidence['acceptance']
    need(recorded['source_fingerprint_sha256'] == runtime['source_fingerprint_sha256'], 'acceptance evidence has a stale source fingerprint')
    need(evidence['full_suite']['exit_code'] == 0 and evidence['full_suite']['named_test_or_subtest_events']['fail'] == 0,
         'acceptance run failed')
    need(evidence['full_suite']['started_packages'] == evidence['full_suite']['completed_packages'], 'acceptance run is incomplete')
    tasks = {t['id']: t for t in rows}
    all_complete = True
    criteria_count = 0
    for row in entries:
        task = tasks[row['id']]
        need(task['status'] in {'planned', 'ready', 'in_progress', 'review', 'verified', 'blocked_by_spec', 'deferred', 'out_of_scope'}, 'unknown task status')
        need(task['owner'] and task['scope_limit'], 'missing task ownership/scope')
        criteria = row['criteria']
        need(len(criteria) == len(REQUIRED[row['id']]) and {c['id'] for c in criteria} == REQUIRED[row['id']],
             f"{row['id']}: missing or duplicate acceptance criterion")
        for criterion in criteria:
            criteria_count += 1
            label = row['id'] + '/' + criterion['id']
            need(criterion['statement'] and criterion['artifacts'], label + ': no acceptance artifacts')
            need(criterion['status'] in {'review', 'verified', 'blocked_by_spec'}, label + ': unknown criterion status')
            complete = criterion['status'] == 'verified'
            all_complete &= complete and task['status'] == 'verified'
            need(task['status'] != 'verified' or complete, label + ': verified task has an unresolved criterion')
            if not complete:
                need(criterion.get('remaining_work'), label + ': incomplete criterion lacks closure conditions')
                continue
            need(criterion.get('tests') or criterion.get('commands'), label + ': no executable evidence')
            for path in criterion['artifacts']:
                p = root/path
                need(p.is_file(), label + ': missing artifact ' + path)
                need(evidence['artifacts'].get(path) == hashlib.sha256(p.read_bytes()).hexdigest(),
                     label + ': stale or missing artifact fingerprint ' + path)
            for test in criterion.get('tests', []):
                need(recorded['passed_tests'].get(test) == 'pass', label + ': no passing execution for ' + test)
                package, name = test.split('::', 1)
                directory = root/'pkg'/package.removeprefix(MODULE)
                need(any(re.search(r'^func ' + re.escape(name.split('/')[0]) + r'\(', p.read_text(), re.M)
                         for p in directory.glob('*_test.go')), label + ': test has no current executable consumer')
            for command in criterion.get('commands', []):
                need(command in COMMANDS and command in checked_commands, label + ': missing command result ' + command)
    # Preserve the coverage that specifically closes the two substantive findings.
    by_id = {t['id']: {c['id']: c for c in t['criteria']} for t in entries}
    critical = by_id['HGT-005']['canonical_head_height']
    need(MODULE+'txpool::TestCanonicalFramePrefixUsesHeadBlockPrerequisites' in critical.get('tests', []), 'head-height regression omitted')
    semantics = by_id['HGT-007']
    native = set(semantics['approval'].get('tests', []) + semantics['rollback'].get('tests', []))
    expected = {'approval-zero', 'approval-wrong-target', 'approval-payment-first', 'approval-once',
                'verify-rollback', 'atomic-body-rollback', 'cross-frame-refill-rollback'}
    need({MODULE+'core/execution::TestHegotaSemanticGoldens/'+s for s in expected} <= native,
         'native semantic acceptance coverage omitted')
    return {'complete': bool(all_complete), 'tasks': len(rows), 'criteria': criteria_count}


def validate_inventory(root, inventory, evidence):
    rows = [r for r in inventory['requirements'] if r['requirement_id'].startswith('PRE-')]
    need(len(rows) == 16 and len({r['requirement_id'] for r in rows}) == 16, 'prerequisite inventory omission/duplicate')
    for row in rows:
        current = row.get('current_assessment', {})
        need(current.get('observation') and current.get('remaining_scope') and current.get('files') and current.get('tests'),
             row['requirement_id'] + ': missing current prerequisite assessment')
        need(re.fullmatch('[0-9a-f]{40}', current.get('source_base', '')), 'prerequisite assessment lacks a fixed base')
        for path in current['files']:
            need((root/path).is_file() and evidence['artifacts'].get(path) == hashlib.sha256((root/path).read_bytes()).hexdigest(),
                 'stale prerequisite source evidence: ' + path)
        for test in current['tests']:
            need(evidence['acceptance']['passed_tests'].get(test) == 'pass', 'unexecuted prerequisite evidence: ' + test)


def validate_skips(inventory, evidence):
    for field in ('full_suite', 'race_suite'):
        need(len(inventory[field]) == evidence[field]['named_test_or_subtest_events']['skip'], 'skip inventory/count mismatch')
