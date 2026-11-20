# Native VM conformance qualification

- [x] Reproduce official Osaka/Amsterdam execution differences and memory panics against unchanged independent execution.
- [x] Correct opcode/precompile costs, full-width offsets and zero-length ranges, creation gas/return-data/address semantics, touches and CHAINID propagation.
- [x] Harden the CREATE oracle: remove self-referential RLP expectations and compare independent nonce-boundary vectors.
- [x] Pass 34 independent EVM controls, two full-width CHAINID contexts, the complete signed delegation-reentry fixture and eleven detected behavioral mutations.
- [x] Re-run complete modern official state corpora after composition: no remaining post-state mismatches; remaining cases are separately tracked raw transaction validation failures.
- [x] Pass full 212-package tests, Core integration, seven affected race packages, full vet and build; publish the dedicated corrective PR.
