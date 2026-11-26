#!/usr/bin/env python3
"""Independent RLP/Keccak vectors for the isolated nonce-only EIP-8250 delta.
This does not resolve D01 or register a public wire format.
"""
import argparse
import copy
import json
from pathlib import Path
from codec_goldens import encode, keccak

ROOT = Path(__file__).resolve().parents[2]

def vectors():
    rows = []
    for name, keys, seq in [('empty', [], 0), ('zero-key', [0], 0),
                            ('single-use', [0, 42], 2**64-1),
                            ('uint256-boundary', [2**256-1], 2**64-2)]:
        payload = [2**256-1, keys, seq, '0x'+'11'*20,
                   [[1, 3, '0x', [50000, 195840], 0, '0x0100']],
                   [[0, '0x', '0x', '0x010203']], [10000000, 1], []]
        elided = copy.deepcopy(payload)
        elided[5][0][3] = 3
        wire = encode(payload)
        rows.append({'id': name, 'wire': '0x'+wire.hex(),
                     'hash': '0x'+keccak(wire).hex(),
                     'signing_hash': '0x'+keccak(encode(elided)).hex(),
                     'key_set_hash': '0x'+keccak(len(keys).to_bytes(32, 'big')+
                         b''.join(k.to_bytes(32, 'big') for k in keys)).hex()})
    return {'profile': '8250-nonce-only-delta-isolated-v1', 'decision': 'D01-open',
            'public_activation': False, 'vectors': rows}

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--write', action='store_true')
    args = parser.parse_args()
    path = ROOT/'specs/hegota/b-codec-goldens.json'
    expected = vectors()
    if args.write:
        path.write_text(json.dumps(expected, indent=2)+'\n')
    if json.loads(path.read_text()) != expected:
        raise SystemExit('B nonce golden mismatch')
    print('verified 4 isolated EIP-8250 encoding/hash/key-set goldens')
