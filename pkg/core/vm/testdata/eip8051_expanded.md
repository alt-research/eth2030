# EIP-8051 expanded-key fixtures

The fixed draft in `refs/EIPs/EIPS/eip-8051.md` specifies a 32-byte message,
a 2420-byte signature, and a 20512-byte expanded key. The key contains
row-major NTT matrix coefficients, the 32-byte `tr`, and four NTT polynomials
for `t1`. Field elements are unsigned big-endian 32-bit words. The SHAKE
variant applies the factor 8192 during verification. The Keccak variant
receives the already scaled vector.

The independent fixture generator uses ETHDILITHIUM revision
`f9f4fe1d84e68304b425e054c273c276fe44ba11` and NTT revision
`3ac5f5dc0b204e91df4f68e347ade2bd4a82bb37`. Use those two checkouts, Python
with PyCryptodome, and the following command from the repository root.

```sh
PYTHONPATH=/path/to/ETHDILITHIUM/pythonref:/path/to/NTT/assets/pythonref \
  python3 pkg/core/vm/testdata/generate_eip8051_expanded.py
```

The generator retains the reference signing and polynomial arithmetic and
adapts the message and `tr` to the draft pseudocode. The upstream compact-key
KATs use a 64-byte `tr` and FIPS message framing. The fixtures test the pinned
draft wire format.
