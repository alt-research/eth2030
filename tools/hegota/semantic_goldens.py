#!/usr/bin/env python3
"""Check hand-worked 8141 gas and 8272 domain oracles without native execution."""
import json
from pathlib import Path
from codec_goldens import keccak
ROOT=Path(__file__).resolve().parents[2]
def check(data=None):
 if data is None:data=json.loads((ROOT/'specs/hegota/semantic-goldens.json').read_text())
 for v in data['gas_vectors']:
  i=v['input'];raw=bytes.fromhex(i['frame_data'][2:]);sig=bytes.fromhex(i['signature'][2:]);tokens=raw.count(0)+4*(len(raw)-raw.count(0))
  intrinsic=12000+475+4*tokens+16*len(sig)+100
  floor=12000+475+100+16*4*(len(raw)+len(sig))
  standard=intrinsic+i['execution_limit']+i['state_limit'];maximum=max(standard,floor+i['state_limit'])
  required=maximum*i['base_fee']+i['blob_count']*131072*i['blob_base_fee']
  maximum_cost=required+min(i['tip_cap']*maximum,i['max_fee']-required)
  before=intrinsic+i['execution_used']+i['state_used'];refund=min(i['refund_counter'],before//5)
  execution=max(before-refund-i['state_used'],floor);used=execution+i['state_used']
  base=used*i['base_fee']+i['blob_count']*131072*i['blob_base_fee'];tip=min(i['tip_cap']*used,maximum_cost-base)
  result=dict(intrinsic=intrinsic,calldata_floor=floor,standard_limit=standard,max_gas=maximum,required_max_fee=required,max_cost=maximum_cost,gas_used=used,payer_refund=maximum_cost-base-tip,block_execution_gas=max(before-i['state_used'],floor),block_state_gas=i['state_used'])
  assert result==v['expected'],(v['id'],result)
 for v in data['recent_root_vectors']:
  decode=lambda k:bytes.fromhex(v[k][2:]);sid=keccak(decode('source_address')+decode('salt'));slot=v['slot'];index=slot%8192
  assert v['source_id']=='0x'+sid.hex() and v['index']==index
  assert v['entry_hash']=='0x'+keccak(keccak(b'RECENT_ROOT_ENTRY')+sid+slot.to_bytes(8,'big')+decode('root')).hex()
  assert v['storage_key']=='0x'+keccak(keccak(b'RECENT_ROOT_STORAGE')+sid+index.to_bytes(8,'big')).hex()
 expected_ids={'approval-zero','approval-wrong-target','approval-payment-first','approval-once','verify-rollback','atomic-body-rollback','cross-frame-refill-rollback','keyed-nonce-single-use','recent-root-slot-window'}
 traces=data['trace_vectors']
 assert len(traces)==9 and {v['id'] for v in traces}==expected_ids, 'missing/duplicate semantic case'
 for v in traces:
  assert isinstance(v['input'],dict) and isinstance(v['expected'],dict), 'semantic seed is not a concrete fixture'
  assert v['consumer'] and v['qualification'] and v['spec_commit']
  if v['eip']==8141:
   assert v['consumer']=='TestHegotaSemanticGoldens' and v['kind'] in ('approval_calls','transaction')
   if v['kind']=='approval_calls':
    assert len(v['expected']['steps'])==len(v['input']['frames'])
    for step in v['expected']['steps']:
     assert step['error'] in ('','reverted') and isinstance(step['approved'],bool)
     assert all(isinstance(step[k],int) for k in ('nonce','sender_balance','payer_balance','execution_gas'))
   else:
    assert isinstance(v['expected']['invalid'],bool)
    if not v['expected']['invalid']:
     assert len(v['expected']['statuses'])==len(v['expected']['state_gas'])==len(v['expected']['execution_gas'])==len(v['input']['frames'])
  elif v['eip']==8250:
   assert v['kind']=='keyed_nonce' and v['qualification']=='isolated_spec_oracle_not_network_activation'
   i=v['input'];keys=i['keys'];seq=i['sequence'];stored=i['stored'];maximum=2**64-1
   valid=0<len(keys)<=16 and keys==sorted(set(keys)) and all(0<=k<2**256 for k in keys) and 0<=seq<=maximum and len(keys)==len(stored)
   valid=valid and all((n==0 if seq==maximum else n==seq and n<maximum) for n in stored)
   charge=97920*stored.count(0)
   consumed=valid and i['approval_succeeds'] and not i['approval_reverts'] and i['state_budget']>=charge
   after=[maximum if seq==maximum else seq+1 for _ in keys] if consumed else stored
   replay=all(n==0 if seq==maximum else n==seq and n<maximum for n in after)
   # A body revert does not undo a successful payment prefix (8250 Nonce consumption).
   result=dict(valid=valid,legacy_nonce=i['legacy_nonce'],stored=after,state_charge=charge if consumed else 0,execution_access=12100*len(keys),replay_valid=replay)
   assert result==v['expected'],(v['id'],result)
  elif v['eip']==8272:
   assert v['kind']=='root_window' and v['qualification']=='isolated_spec_oracle_not_network_activation'
   i=v['input'];slot=i['written_slot'];current=i['current_slots']
   assert len(current)==len(i['canonical_entry_matches'])==len(i['orphaned_entry_matches'])
   allowed=lambda matches:[slot<s and s-slot<=8191 and match for s,match in zip(current,matches)]
   result=dict(index=slot%8192,canonical_valid=allowed(i['canonical_entry_matches']),after_reorg_valid=allowed(i['orphaned_entry_matches']))
   assert result==v['expected'],(v['id'],result)
  else:raise AssertionError('unknown semantic EIP')
 return {'gas_oracles':len(data['gas_vectors']),'root_domain_oracles':len(data['recent_root_vectors']),'native_semantic_cases':7,'conditional_rule_oracles':2,'unexecuted_semantic_trace_seeds':0}
if __name__=='__main__':print(json.dumps(check()))
