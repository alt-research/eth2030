#!/usr/bin/env python3
"""Independent, standard-library-only EIP-8141 encoding/hash fixture generator.
This is a research oracle, never imported by the client. No signature/state
validity is implied by an encoding vector. Default mode verifies fixed goldens.
"""
import argparse
import copy
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
MASK = (1 << 64) - 1
RC = [0x0000000000000001,0x0000000000008082,0x800000000000808a,0x8000000080008000,0x000000000000808b,0x0000000080000001,0x8000000080008081,0x8000000000008009,0x000000000000008a,0x0000000000000088,0x0000000080008009,0x000000008000000a,0x000000008000808b,0x800000000000008b,0x8000000000008089,0x8000000000008003,0x8000000000008002,0x8000000000000080,0x000000000000800a,0x800000008000000a,0x8000000080008081,0x8000000000008080,0x0000000080000001,0x8000000080008008]
ROT = [[0,36,3,41,18],[1,44,10,45,2],[62,6,43,15,61],[28,55,25,21,56],[27,20,39,8,14]]
def rol(x,n): return ((x << n) | (x >> ((64-n) % 64))) & MASK

def keccak(data):
    rate = 136
    padded = bytearray(data); padded.append(1)
    padded.extend(b'\0' * ((rate-len(padded)%rate)%rate)); padded[-1] |= 128
    a = [0]*25
    for offset in range(0,len(padded),rate):
        block = padded[offset:offset+rate]
        for i in range(rate//8): a[i] ^= int.from_bytes(block[8*i:8*i+8],'little')
        for rc in RC:
            c=[a[x]^a[x+5]^a[x+10]^a[x+15]^a[x+20] for x in range(5)]
            d=[c[(x-1)%5]^rol(c[(x+1)%5],1) for x in range(5)]
            for x in range(5):
                for y in range(5): a[x+5*y] ^= d[x]
            b=[0]*25
            for x in range(5):
                for y in range(5): b[y+5*((2*x+3*y)%5)] = rol(a[x+5*y],ROT[x][y])
            for x in range(5):
                for y in range(5): a[x+5*y]=b[x+5*y]^((~b[(x+1)%5+5*y])&b[(x+2)%5+5*y])
            a[0] ^= rc
    return b''.join(x.to_bytes(8,'little') for x in a)[:32]

def integer(n):
    assert isinstance(n,int) and n >= 0
    return n.to_bytes((n.bit_length()+7)//8,'big')

def rlp(value):
    if isinstance(value,list):
        data=b''.join(rlp(x) for x in value); prefix=0xc0
    else:
        data=integer(value) if isinstance(value,int) else bytes.fromhex(value[2:])
        if len(data)==1 and data[0]<128:return data
        prefix=0x80
    if len(data)<56:return bytes([prefix+len(data)])+data
    size=integer(len(data));return bytes([prefix+55+len(size)])+size+data

def encode(payload):return b'\x06'+rlp(payload)
def sighash(payload):
    elided=copy.deepcopy(payload)
    for s in elided[4]:
        if s[2]=='0x':s[3]=len(bytes.fromhex(s[3][2:]))
    return keccak(encode(elided))

def vectors():
    # Deliberately ARBITRARY bytes: these vectors specify encoding and pricing,
    # not an authorization proof or a valid execution against an unknown state.
    base=[1,0,'0x'+'11'*20,[[1,3,'0x',[50000,0],0,'0x0100']],[[0,'0x','0x','0x010203']],[10000000,1],[]]
    variants=[('canonical-empty-message',copy.deepcopy(base))]
    p=copy.deepcopy(base);p[4][0][3]='0xaabbcc';variants.append(('same-length-signature',p))
    p=copy.deepcopy(base);p[4][0][3]='0x01020304';variants.append(('longer-signature',p))
    p=copy.deepcopy(base);p[4][0][2]='0x'+'22'*32;variants.append(('explicit-message',p))
    p=copy.deepcopy(variants[-1][1]);p[4][0][3]='0xaabbcc';variants.append(('explicit-message-changed-signature',p))
    p=copy.deepcopy(base);p[1]=(1<<64)-2;variants.append(('largest-base-nonce',p))
    p=copy.deepcopy(base);p[3]=p[3]*64;variants.append(('64-frames-long-rlp',p))
    out=[]
    for name,p in variants:
        encoded=encode(p)
        out.append({'id':name,'profile':'8141-base-research-v1','purpose':'encoding-and-hash-only','payload':p,'bytes':'0x'+encoded.hex(),'transaction_hash':'0x'+keccak(encoded).hex(),'signature_hash':'0x'+sighash(p).hex()})
    assert out[0]['signature_hash']==out[1]['signature_hash']
    assert out[0]['transaction_hash']!=out[1]['transaction_hash']
    assert out[0]['signature_hash']!=out[2]['signature_hash']
    assert out[3]['signature_hash']!=out[4]['signature_hash']
    return {'version':1,'source_eip':8141,'vectors':out}

def main():
    assert keccak(b'').hex()=='c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470'
    assert keccak(b'abc').hex()=='4e03657aea45a94fc7d47ba826c8d667c0d1e6e33a64a036ec44f58fa12d6c45'
    args=argparse.ArgumentParser();args.add_argument('--write',action='store_true');options=args.parse_args()
    path=ROOT/'specs/hegota/codec-goldens.json';data=vectors()
    if options.write:path.write_text(json.dumps(data,indent=2)+'\n')
    else:
        assert json.loads(path.read_text())==data,'goldens changed; review the specification before updating'
    print('verified 7 independent encoding/hash goldens')
if __name__=='__main__':main()
