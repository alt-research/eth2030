#!/usr/bin/env python3
"""Independent SHA-256/SSZ Heze oracle. Never reads native Go output."""
import hashlib,json,struct
from pathlib import Path
H=lambda a,b:hashlib.sha256(a+b).digest()
Z=bytes(32)
def merkle(chunks,width):
 v=list(chunks)+[Z]*(width-len(chunks))
 while len(v)>1:v=[H(v[i],v[i+1]) for i in range(0,len(v),2)]
 return v[0]
def progressive(v,width=1):return Z if not v else H(merkle(v[:width],width),progressive(v[width:],width*4))
def sized(root,n):return H(root,struct.pack('<Q',n)+bytes(24))
def byte_root(data):return sized(progressive([data[i:i+32].ljust(32,b'\x00') for i in range(0,len(data),32)]),len(data))
def encode(txs):
 offsets=[];n=4*len(txs)
 for tx in txs:offsets.append(struct.pack('<I',n));n+=len(tx)
 return b''.join(offsets+txs)
def make(name,txs):
 slot=100;validator=7;dep=bytes(range(32));fork=bytes.fromhex('08000000');genesis=bytes([0x42])*32
 root=merkle([struct.pack('<Q',slot)+bytes(24),struct.pack('<Q',validator)+bytes(24),dep,sized(progressive([byte_root(x)for x in txs]),len(txs))],4)
 domain=bytes.fromhex('10000000')+H(fork+bytes(28),genesis)[:28]
 message=struct.pack('<QQ',slot,validator)+dep+struct.pack('<I',52)+encode(txs)
 return {'name':name,'slot':slot,'validator':validator,'dependent_root':dep.hex(),'transactions':[x.hex()for x in txs],'message_ssz':message.hex(),'signed_ssz':(struct.pack('<I',100)+bytes([0x99])*96+message).hex(),'root':root.hex(),'fork_version':fork.hex(),'genesis_validators_root':genesis.hex(),'domain':domain.hex(),'signing_root':H(root,domain).hex()}
if __name__=='__main__':
 out={'oracle':'Independent Python hashlib + explicit SSZ/progressive Merkle rules; signature bytes are codec data, not a valid signature claim','cases':[make('empty-codec',[]),make('opaque-transactions',[b'\xff',b'\x00\xc0',bytes(range(33))]),make('more-than-old-sixteen',[bytes([i])for i in range(20)])]}
 p=Path(__file__).resolve().parents[2]/'pkg/focil/heze/testdata/ssz.json';p.parent.mkdir(parents=True,exist_ok=True);p.write_text(json.dumps(out,indent=2)+'\n')
