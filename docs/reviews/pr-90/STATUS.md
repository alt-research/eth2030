# Reversible transaction-boundary empty-account finalization

The native state transition retained empty accounts created while calling precompiles. Independent Osaka fixtures consequently had different complete roots even when the returned values, storage gas observations and sender payment matched. EIP-161 requires deleting touched empty accounts at transaction end, preserving untouched accounts and reverting deletion/touch effects on snapshot rollback.

This correction records reversible touches, adds explicit fork-aware finalization to the state contract, and wires ordinary, simulated, Frame and system execution. Memory, trie, snap and stateless state preserve the same deletion/recreation boundaries. Stateless snapshot copies retain touch state and allow later recreation of a finalized empty account. Earlier-fork callers keep empty accounts. No runtime setting or feature switch is introduced.

Validation includes read-only and reverted touches, pre-fork behavior, rollback after finalization, durable trie/snap reload, later recreation, and a real signed native identity-precompile transaction matching unchanged Geth v1.17.10 at 21,040 gas and root `0x7b307caf841f40f3adec88ab94b899fdbaf7ded6ccece28b604b73393bd15429`. Three behavioral mutations were detected: skipped empty deletion, leaked reverted touches, and omitted trie finalization. Full 212-package Go tests, Core integration, affected race (including Node), full vet and build pass; the final stateless recreation refinement also passed fresh full/affected race/integration/vet/build checks.

With the separate Osaka ModExp correction, all 23 EIP-7823 and 160 EIP-7883 official state fixtures match. CLZ activation, strict fixture exception handling and complete cross-client/release campaigns are separate work; this review does not close those obligations.

Normative input: [EIP-161 including the rollback addendum](https://github.com/ethereum/EIPs/blob/3db38a3725105cb67b8804ac3e147967113bc3ca/EIPS/eip-161.md). Discovery corpus: EEST v5.4.0 develop, SHA-256 `3e2b02d49fe903eda4fd8caca5cbf0d139c470e97e1de9a85299b1b034f97099`.
