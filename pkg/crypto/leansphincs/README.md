# Pinned leanSPHINCS verifier

This package implements EIP-8504's referenced parameter set: Keccak-256,
16-byte nodes, WOTS+C with 64 two-bit chains and target sum 128, 24 FORS trees
of height 10, and hypertree heights 8/8/10. Public keys are 32 bytes and
signatures are exactly 7,740 bytes. A valid verification performs exactly 515
Keccak-f[1600] permutations. Invalid lengths are rejected before hashing;
verification never searches a counter or accepts dynamic parameters.

Normative revision: ethereum/EthLib
`c4474ed422d83b6b735ef802ee43960abe1b788f`, Cryptography/Sphincs.
Independent implementation: leanEthereum/leanVM
`92a97c919b2f2d6fe09853090b6b23f781ecec07`, crates/lean_sphincs.
The Go implementation follows those algorithms and address/hash layouts.

`testdata/ethlib-verify.json` preserves all 26 verification cases and their
expected results from the pinned leanVM mirror of EthLib's vectors, including
null inputs that the source decoder rejects. It records each source path and
SHA-256. Tests preserve those expectations, check valid-case permutation
counts, and reject every signature truncation before hashing.

This is a verification primitive, not a claim that EIP-8504 is finalized or
activated. Its parameters and gas values remain provisional upstream. It does
not substitute the repository's SHA2 SPHINCS+ implementation, aggregate STARKs,
or a mock verifier for leanSPHINCS.
