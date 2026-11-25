#!/usr/bin/env python3
"""Verify frozen semantic oracles; never regenerate expected values during checks."""
import json
import re
from pathlib import Path
from codec_goldens import keccak

ROOT = Path(__file__).resolve().parents[2]
SPEC = 'ade9e9cab7e148bba4cd293f916a4a189cd7a8aa'
NATIVE = {'approval-zero', 'approval-wrong-target', 'approval-payment-first', 'approval-once',
          'verify-rollback', 'atomic-body-rollback', 'cross-frame-refill-rollback'}
TRACES = NATIVE | {'keyed-nonce-single-use', 'recent-root-slot-window'}
MAX_NONCE = 2**64 - 1


def require(value, message):
    if not value:
        raise ValueError(message)


def same(actual, expected):
    if type(actual) is not type(expected):
        return False
    if type(actual) is dict:
        return actual.keys() == expected.keys() and all(same(v, expected[k]) for k, v in actual.items())
    if type(actual) is list:
        return len(actual) == len(expected) and all(same(a, b) for a, b in zip(actual, expected))
    return actual == expected


def fields(value, required, optional=(), label='fixture'):
    require(type(value) is dict, label + ': expected an object, not a prose seed')
    require(set(required) <= set(value) <= set(required) | set(optional), label + ': missing or unknown fields')


def integer(value, maximum=None):
    require(type(value) is int and value >= 0 and (maximum is None or value <= maximum), 'invalid unsigned integer')
    return value


def unhex(value, size=None):
    require(type(value) is str and re.fullmatch(r'0x(?:[0-9a-fA-F]{2})*', value) is not None, 'invalid hex bytes')
    result = bytes.fromhex(value[2:])
    require(size is None or len(result) == size, 'wrong hex width')
    return result


def family(rows, expected, label):
    require(type(rows) is list and len(rows) == len(expected) and {r['id'] for r in rows} == expected,
            label + ': missing/duplicate oracle coverage')


def root_domain(source, salt, root, slot):
    integer(slot, MAX_NONCE)
    sid = keccak(unhex(source, 20) + unhex(salt, 32))
    index = slot % 8192
    return dict(source_id='0x'+sid.hex(), index=index,
                entry_hash='0x'+keccak(keccak(b'RECENT_ROOT_ENTRY')+sid+slot.to_bytes(8, 'big')+unhex(root, 32)).hex(),
                storage_key='0x'+keccak(keccak(b'RECENT_ROOT_STORAGE')+sid+index.to_bytes(8, 'big')).hex())


def keyed_nonce(i):
    # Isolated EIP-8250 nonce transition for an existing sender, not an envelope
    # codec or production state transition. Fees/composition remain gated by D01.
    fields(i, {'sender', 'legacy_nonce', 'keys', 'sequence', 'stored', 'state_budget',
               'approval_succeeds', 'approval_reverts', 'body_reverts'},
           {'legacy_nonce_at_approval', 'transaction_invalid'})
    unhex(i['sender'], 20)
    for k in ('legacy_nonce', 'state_budget'):
        integer(i[k], MAX_NONCE)
    for k in ('approval_succeeds', 'approval_reverts', 'body_reverts', 'transaction_invalid'):
        require(type(i.get(k, False)) is bool, 'nonce transition flag is not boolean')
    keys, seq, stored = i['keys'], i['sequence'], i['stored']
    require(type(keys) is list and type(stored) is list and type(seq) is int and
            all(type(k) is int for k in keys) and all(type(n) is int and 0 <= n <= MAX_NONCE for n in stored),
            'malformed nonce oracle input')
    legacy = i.get('legacy_nonce_at_approval', i['legacy_nonce'])
    integer(legacy, MAX_NONCE)
    shape = len(keys) <= 16 and len(keys) == len(stored) and keys == sorted(set(keys)) and all(0 <= k < 2**256 for k in keys)
    shape = shape and 0 <= seq <= MAX_NONCE and (bool(keys) or seq < MAX_NONCE)
    def valid_at(account_nonce, values):
        return shape and (all(n == 0 if seq == MAX_NONCE else n == seq and n < MAX_NONCE for n in values)
                          if keys else seq == account_nonce)
    valid = valid_at(i['legacy_nonce'], stored)
    charge = 97920 * stored.count(0) if keys else 0
    applied = (valid and i['approval_succeeds'] and not i['approval_reverts'] and
               not i.get('transaction_invalid', False) and i['state_budget'] >= charge and (bool(keys) or legacy < MAX_NONCE))
    after = [MAX_NONCE if seq == MAX_NONCE else seq+1 for _ in keys] if applied and keys else list(stored)
    after_legacy = (legacy if keys else legacy+1) if applied else i['legacy_nonce']
    return dict(valid=valid, approval_applied=applied, legacy_nonce=after_legacy, stored=after,
                state_charge=charge if applied else 0, execution_access=12100*len(keys) if shape else 0,
                replay_valid=valid_at(after_legacy, after))


def conditional(v):
    i = v['input']
    if v['kind'] == 'keyed_nonce':
        require(v['eip'] == 8250, 'nonce oracle EIP mismatch')
        return keyed_nonce(i)
    require(v['eip'] == 8272, 'root oracle EIP mismatch')
    if v['kind'] == 'root_window':
        fields(i, {'written_slot', 'current_slots', 'canonical_entry_matches', 'orphaned_entry_matches'})
        slot = integer(i['written_slot'], MAX_NONCE)
        current = i['current_slots']
        require(type(current) is list and len(current) > 0 and len(current) == len(i['canonical_entry_matches']) == len(i['orphaned_entry_matches']), 'root window coverage is empty or inconsistent')
        for s in current:
            integer(s, MAX_NONCE)
        for key in ('canonical_entry_matches', 'orphaned_entry_matches'):
            require(all(type(m) is bool for m in i[key]), 'root match flag is not boolean')
        def allowed(matches):
            return [slot < s and s-slot <= 8191 and match for s, match in zip(current, matches)]
        return dict(index=slot % 8192, canonical_valid=allowed(i['canonical_entry_matches']), after_reorg_valid=allowed(i['orphaned_entry_matches']))
    require(v['kind'] == 'root_dependency', 'unknown root oracle kind')
    fields(i, {'source_address', 'salt', 'root', 'slot', 'current_slot', 'storage'})
    result = root_domain(i['source_address'], i['salt'], i['root'], i['slot'])
    current = integer(i['current_slot'], MAX_NONCE)
    require(type(i['storage']) is dict, 'root storage is not a map')
    for key, value in i['storage'].items():
        unhex(key, 32); unhex(value, 32)
    result['valid'] = i['slot'] < current and current-i['slot'] <= 8191 and i['storage'].get(result['storage_key']) == result['entry_hash']
    return result


def native_shape(v):
    i, e = v['input'], v['expected']
    fields(i, {'sender', 'nonce', 'balance', 'codes', 'frames'}, label='native input')
    unhex(i['sender'], 20); integer(i['nonce'], MAX_NONCE); integer(i['balance'])
    require(type(i['codes']) is dict and type(i['frames']) is list and 0 < len(i['frames']) <= 64, 'native fixture lacks code/frames')
    for address, code in i['codes'].items():
        unhex(address, 20); unhex(code)
    for f in i['frames']:
        fields(f, {'target', 'flags', 'mode', 'execution', 'state', 'data'}, label='native frame')
        unhex(f['target'], 20); unhex(f['data'])
        for k in ('flags', 'mode', 'execution', 'state'):
            integer(f[k], MAX_NONCE)
        require(f['flags'] < 8 and f['mode'] in (0, 1, 2), 'invalid native mode/flags')
    if v['kind'] == 'approval_calls':
        fields(e, {'steps', 'escrow'} if v['id'] == 'approval-once' else {'steps'}, label='approval expected')
        if 'escrow' in e: integer(e['escrow'])
        require(len(e['steps']) == len(i['frames']), 'missing per-call expectation')
        for step in e['steps']:
            fields(step, {'error', 'approved', 'payer', 'nonce', 'sender_balance', 'payer_balance', 'execution_gas'}, label='approval step')
            require(step['error'] in ('', 'reverted') and type(step['approved']) is bool, 'invalid approval result')
            if step['payer'] is not None: unhex(step['payer'], 20)
            for k in ('nonce', 'sender_balance', 'payer_balance', 'execution_gas'): integer(step[k])
    else:
        require(v['kind'] == 'transaction', 'unknown native oracle kind')
        require(type(e.get('invalid')) is bool, 'missing transaction validity expectation')
        if e['invalid']:
            fields(e, {'invalid', 'nonce', 'root_unchanged', 'logs', 'gas_used', 'execution_pool_restored', 'state_pool_restored'}, label='invalid transaction expected')
            require(all(type(e[k]) is bool for k in ('root_unchanged', 'execution_pool_restored', 'state_pool_restored')), 'invalid rollback flags')
            integer(e['gas_used'])
        else:
            fields(e, {'invalid', 'nonce', 'payer', 'logs', 'statuses', 'state_gas', 'execution_gas', 'storage'}, label='transaction expected')
            unhex(e['payer'], 20)
            require(len(e['statuses']) == len(e['state_gas']) == len(e['execution_gas']) == len(i['frames']), 'missing frame receipt expectation')
            require(e['storage'], 'missing rollback storage expectation')
            for key in ('statuses', 'state_gas', 'execution_gas'):
                for n in e[key]: integer(n)
        integer(e['nonce']); integer(e['logs'])


def check(data=None):
    if data is None:
        data = json.loads((ROOT/'specs/hegota/semantic-goldens.json').read_text())
    fields(data, {'version', 'purpose', 'gas_vectors', 'recent_root_vectors', 'trace_vectors', 'boundary_vectors'})
    require(data['version'] == 4, 'unknown semantic corpus version')
    family(data['gas_vectors'], {'gas-standard', 'gas-floor-state', 'gas-refund-capped', 'gas-refund-floor', 'gas-clipped-tip-blob'}, 'gas')
    family(data['recent_root_vectors'], {'recent-root-domain'}, 'root domain')
    for v in data['gas_vectors']:
        fields(v, {'id', 'input', 'expected'}, label='gas vector')
        i = v['input']; raw = unhex(i['frame_data']); sig = unhex(i['signature'])
        fields(i, {'frame_data', 'signature', 'signature_scheme', 'empty_signature_message', 'execution_limit',
                   'state_limit', 'execution_used', 'state_used', 'refund_counter', 'base_fee', 'blob_base_fee',
                   'blob_count', 'max_fee', 'tip_cap'}, label='gas input')
        require(i['signature_scheme'] == 'ARBITRARY' and i['empty_signature_message'] is True, 'unsupported gas oracle signature model')
        for k, value in i.items():
            if k not in ('frame_data', 'signature', 'signature_scheme', 'empty_signature_message'): integer(value)
        require(i['execution_used'] <= i['execution_limit'] and i['state_used'] <= i['state_limit'], 'gas oracle exceeds frame budget')
        tokens = raw.count(0)+4*(len(raw)-raw.count(0))
        intrinsic = 12000+475+4*tokens+16*len(sig)+100
        floor = 12000+475+100+64*(len(raw)+len(sig))
        standard = intrinsic+i['execution_limit']+i['state_limit']; maximum = max(standard, floor+i['state_limit'])
        required = maximum*i['base_fee']+i['blob_count']*131072*i['blob_base_fee']
        require(i['max_fee'] >= required, 'gas oracle fee budget insufficient')
        escrow = required+min(i['tip_cap']*maximum, i['max_fee']-required)
        before = intrinsic+i['execution_used']+i['state_used']; refund = min(i['refund_counter'], before//5)
        used = max(before-refund-i['state_used'], floor)+i['state_used']
        base = used*i['base_fee']+i['blob_count']*131072*i['blob_base_fee']; tip = min(i['tip_cap']*used, escrow-base)
        require(used <= maximum and 0 <= base+tip <= escrow <= i['max_fee'], 'gas settlement invariant violated')
        result = dict(intrinsic=intrinsic, calldata_floor=floor, standard_limit=standard, max_gas=maximum, required_max_fee=required,
                      max_cost=escrow, gas_used=used, payer_refund=escrow-base-tip, block_execution_gas=max(before-i['state_used'], floor), block_state_gas=i['state_used'],
                      burned=base, priority_fee=tip, gas_price=i['base_fee']+(escrow-required)//maximum)
        require(same(result, v['expected']), v['id']+': gas oracle mismatch')
    for v in data['recent_root_vectors']:
        result = root_domain(v['source_address'], v['salt'], v['root'], v['slot'])
        require(all(same(v[k], value) for k, value in result.items()), 'root domain oracle mismatch')
    family(data['trace_vectors'], TRACES, 'semantic trace')
    for v in data['trace_vectors'] + data['boundary_vectors']:
        fields(v, {'id', 'kind', 'eip', 'spec_commit', 'consumer_task', 'consumer', 'qualification', 'input', 'expected'})
        fields(v['input'], set(v['input']) if type(v['input']) is dict else (), label='concrete fixture input')
        require(v['spec_commit'] == SPEC, 'semantic fixture source pin mismatch')
        if v['id'] in NATIVE:
            require(v['eip'] == 8141 and v['consumer'] == 'TestHegotaSemanticGoldens' and v['qualification'] == 'native_vm', 'native consumer/EIP mismatch')
            require(v['kind'] == ('approval_calls' if v['id'].startswith('approval-') else 'transaction'), 'native case kind mismatch')
            native_shape(v)
        else:
            require(v['eip'] in (8250, 8272) and v['consumer'] == 'tools/hegota/semantic_goldens.py' and
                    v['qualification'] == 'isolated_spec_oracle_not_network_activation', 'conditional consumer mismatch')
            expected_kind = ('keyed_nonce' if v['id'].startswith(('keyed-nonce-', 'nonce-')) else
                             'root_window' if v['id'] == 'recent-root-slot-window' else 'root_dependency')
            require(v['kind'] == expected_kind, 'conditional case kind mismatch')
            require(same(conditional(v), v['expected']), v['id']+': conditional oracle mismatch')
    ids = [v['id'] for v in data['boundary_vectors']]
    require(len(ids) == len(set(ids)) and not set(ids) & TRACES, 'duplicate boundary vector')
    require(set(ids) == BOUNDARY_IDS, 'missing conditional boundary coverage')
    return dict(gas_oracles=len(data['gas_vectors']), root_domain_oracles=1, native_semantic_cases=7,
                conditional_rule_oracles=2+len(ids), unexecuted_semantic_trace_seeds=0)


BOUNDARY_IDS = {'nonce-duplicate', 'nonce-unsorted', 'nonce-key-overflow', 'nonce-too-many-keys',
                'nonce-keyed-preserves-current-legacy', 'nonce-sixteen-keys',
                'nonce-sequential-first-use', 'nonce-sequential-existing', 'nonce-exhaustion', 'nonce-already-exhausted',
                'nonce-state-budget-short', 'nonce-approval-revert', 'nonce-transaction-invalid',
                'nonce-empty-legacy', 'nonce-legacy-created-before-approval', 'nonce-empty-max',
                'nonce-legacy-overflow', 'root-same-slot', 'root-next-slot', 'root-last-usable', 'root-expired',
                'root-source-domain', 'root-salt-domain', 'root-value-mismatch', 'root-orphaned'}

if __name__ == '__main__':
    print(json.dumps(check()))
