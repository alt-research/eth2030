"""Generate canonical base/keyed PQ-wallet witnesses with pinned EELS Python.

Usage: python generate_d_pq_wallet.py VERIFY_SOURCE SIGN_SOURCE > vectors.json
Requires pycryptodome==3.23.0. Inputs are unmodified pinned upstream files; only
imports/type aliases are adapted. No native Go code supplies expected values.
"""
import ast
import hashlib
import json
import sys
from pathlib import Path
from Crypto.Hash import keccak

REVISION = '1d07f742873458b2a77d11af100f57a8d14180b8'
BLOBS = ['3f30abe823c3d0fcbdc7fce3b011e75444f5d565', '8d45f5d0fdaee5398df69f6c8cb21f92bf60614f']

def k256(data):
    return keccak.new(digest_bits=256, data=data).digest()

def load(path, blob, scope):
    raw = Path(path).read_bytes()
    if hashlib.sha1(b'blob '+str(len(raw)).encode()+b'\0'+raw).hexdigest() != blob:
        raise ValueError('upstream oracle source blob mismatch')
    tree = ast.parse(raw)
    tree.body = [n for n in tree.body if not isinstance(n, ast.ImportFrom) or n.module in ('typing', 'dataclasses', 'functools')]
    exec(compile(tree, path, 'exec'), scope)
    return hashlib.sha256(raw).hexdigest()

scope = dict(__name__=__name__, Bytes=bytes, Bytes32=bytes, Hash=bytes, keccak256=k256)
digests = [load(sys.argv[i+1], blob, scope) for i, blob in enumerate(BLOBS)]
key = scope['LeanSphincsKey'].from_index(1)
pk = key.public_key

def prefix(n, base):
    if n < 56:
        return bytes([base+n])
    raw = n.to_bytes((n.bit_length()+7)//8, 'big')
    return bytes([base+55+len(raw)])+raw

def rlp(value):
    if isinstance(value, list):
        raw = b''.join(map(rlp, value)); return prefix(len(raw), 0xc0)+raw
    if isinstance(value, int):
        value = value.to_bytes((value.bit_length()+7)//8, 'big')
    if len(value) == 1 and value[0] < 128:
        return value
    return prefix(len(value), 0x80)+value

code = bytearray()
def guard(expression):
    code.extend(expression)
    code.extend(b'\x61\x00\x00'); at = len(code)-2
    code.extend(b'\x57\x5f\x5f\xfd')
    code[at:at+2] = len(code).to_bytes(2, 'big')
    code.append(0x5b)

guard(bytes.fromhex('60026001b3600414'))       # FRAMEPARAM(mode,1)==4
guard(bytes.fromhex('60046001b3606014'))       # data length==96
guard(bytes.fromhex('60016000b1601014'))       # scheme word==0x10
guard(bytes.fromhex('60016020b115'))           # canonical signing-hash placeholder
guard(bytes.fromhex('60016040b17f')+pk+b'\x14')
code.extend(bytes.fromhex('60035f5faa'))
rows=[]
for keyed in (False, True):
    declaration=bytes(31)+b'\x10'+bytes(32)+pk
    frames=[[1,3,b'',[10000,97920 if keyed else 0],0,b''], [4,0,b'',[3000,0],0,declaration]]
    # Pinned EIP-8141 field order: mode,flags,target,limits,value,data.
    sender=bytes(19)+b'\x90'
    fields=([1337,[42],0,sender] if keyed else [1337,0,sender])+[frames,[],[10**18,10**9],[]]
    raw=b'\x06'+rlp(fields); digest=k256(raw); signature=key.sign(digest)
    if not scope['lean_sphincs_verify'](pk,digest,signature):
        raise ValueError('independent EELS signer/verifier mismatch')
    rows.append(dict(name='keyed' if keyed else 'base', transaction=raw.hex(), signing_hash=digest.hex(), witness=signature.hex()))
print(json.dumps(dict(reference='lambdaclass/execution-specs@'+REVISION, source_blobs=BLOBS, source_sha256=digests,
                     oracle='unmodified upstream algorithms; Python RLP and PyCryptodome 3.23.0 Keccak',
                     public_key=pk.hex(), wallet_code=code.hex(), cases=rows), indent=2))
