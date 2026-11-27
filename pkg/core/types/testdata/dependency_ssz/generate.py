"""Independent Gloas/EIP-8504 vectors, using remerkleable 2f0baeef0082.

The first 19 fields are consensus-specs Gloas ExecutionPayload; the appended
ProgressiveList[ProgressiveByteList] follows pinned ethlambda gloas.rs.
No eth2030 implementation is imported. Run with PYTHONPATH at the pinned repo.
"""
import json
from remerkleable.basic import uint64, uint256
from remerkleable.byte_arrays import ByteList, ByteVector, Bytes32
Bytes20 = ByteVector[20]
from remerkleable.complex import Container
from remerkleable.progressive import ProgressiveContainer, ProgressiveByteList, ProgressiveList

class Withdrawal(Container):
    index: uint64
    validator_index: uint64
    address: Bytes20
    amount: uint64

fields = dict(parent_hash=Bytes32, fee_recipient=Bytes20, state_root=Bytes32,
              receipts_root=Bytes32, logs_bloom=ByteVector[256], prev_randao=Bytes32,
              block_number=uint64, gas_limit=uint64, gas_used=uint64, timestamp=uint64,
              extra_data=ByteList[32], base_fee_per_gas=uint256, block_hash=Bytes32,
              transactions=ProgressiveList[ProgressiveByteList],
              withdrawals=ProgressiveList[Withdrawal], blob_gas_used=uint64,
              excess_blob_gas=uint64, block_access_list=ProgressiveByteList, slot_number=uint64)
values = dict(parent_hash=b'\x01'*32, fee_recipient=b'\x02'*20, state_root=b'\x03'*32,
              receipts_root=b'\x04'*32, logs_bloom=b'\x00'*3+b'\x5a'+b'\x00'*252,
              prev_randao=b'\x05'*32, block_number=6, gas_limit=7, gas_used=8,
              timestamp=9, extra_data=b'\x0a\x0b', base_fee_per_gas=0,
              block_hash=b'\x0c'*32, transactions=[b'\x02\xab', b'\x06'],
              withdrawals=[Withdrawal(index=13, validator_index=14, address=b'\x0f'*20, amount=16)],
              blob_gas_used=17, excess_blob_gas=18, block_access_list=b'\xc1\x80', slot_number=19)
cases=[]
for name, witnesses in [('absent', None), ('empty', []), ('two', [b'\xaa'*7740, b'\xbb'*7740])]:
    schema=dict(fields); inputs=dict(values)
    if witnesses is not None:
        schema['dependency_witnesses']=ProgressiveList[ProgressiveByteList]
        inputs['dependency_witnesses']=witnesses
    cls=type('Payload', (ProgressiveContainer(active_fields=[1]*len(schema)),), {'__annotations__':schema})
    value=cls(**inputs)
    cases.append(dict(name=name, fields=len(schema), ssz=value.encode_bytes().hex(), root=value.hash_tree_root().hex()))
print(json.dumps(dict(oracle='ethereum/remerkleable@2f0baeef0082d4278acaef7d822deb7009d7db7e',
                     reference='lambdaclass/ethlambda@7aef8bd75316078010193684715359b26b5c2be6',
                     reference_blob='4fa26a62b25f7165a8638874da7242b594339861', cases=cases), indent=2))
