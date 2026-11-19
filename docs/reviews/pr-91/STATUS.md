# Canonical Osaka and Amsterdam CLZ activation

CLZ already existed in the private legacy jump table but was absent from the canonical Osaka table and Amsterdam inheritance. This selects an explicit Osaka table and derives Amsterdam from it. The existing opcode logic and normative 5-gas cost are preserved. Prague still rejects CLZ.

Actual native EVM calls for zero and seven nonzero 256-bit boundary words match unchanged Geth output; the complete short program uses exactly 21 gas. The same native program is checked with Osaka and Amsterdam tables. All 212 Go packages, Core integration, affected race, full vet and build pass.

Composed with PRs 88/89/90 and the strict native fixture adapter, all 598 official EIP-7939 state cases pass, including call variants, delegated code, gas boundaries, underflow/overflow and code-copy paths. The complete selected Osaka corpus passes 1,223 cases; these are integration results, not a claim that this dependent PR alone contains the other corrections.

This review depends on [PR 89](https://github.com/alt-research/eth2030/pull/89) for the native Osaka rule and targets that branch. Other fixture/application/full-client/release obligations remain tracked separately. Normative input: [EIP-7939](https://github.com/ethereum/EIPs/blob/3db38a3725105cb67b8804ac3e147967113bc3ca/EIPS/eip-7939.md).
