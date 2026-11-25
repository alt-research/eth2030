#!/usr/bin/env python3
"""Check hand-worked 8141 gas and 8272 domain oracles without native execution."""
import json
from pathlib import Path
from codec_goldens import keccak
ROOT=Path(__file__).resolve().parents[2]
def check():
 data=json.loads((ROOT/'specs/hegota/semantic-goldens.json').read_text())
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
 assert len(data['trace_vectors'])==9
 return {'gas_oracles':2,'root_domain_oracles':1,'unexecuted_semantic_trace_seeds':9}
if __name__=='__main__':print(json.dumps(check()))
