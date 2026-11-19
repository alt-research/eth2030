# Native VM conformance correction

The full native state campaign exposed shared EVM errors that the smaller gas corpus did not exercise. This review restores:

- MUL's five-gas tier, complete failed-precompile gas exhaustion, EIP-2200's stipend sentry and full cold SELFDESTRUCT cost.
- EIP-1052 zero results for empty accounts, zero-value CALL/STATICCALL/SELFDESTRUCT touches and dead-account value-transfer charges.
- Full-width calldata/code/return-data offsets, zero-length memory regions and overflow-safe word rounding without consensus-input panics.
- CREATE parent retention outside the child and code-deposit budget; top-level exceptional creation consumes its full child budget. Collisions retain parent gas, oversized opcode initcode halts before nonce mutation, eligible prefunded accounts reset storage, and prohibited code prefixes are rejected.
- EIP-211's empty return buffer after successful CREATE, configured full-width CHAINID, and final EIP-7702 destination-target warming after authorizations.
- Canonical CREATE address derivation through the existing RLP encoder. The previous custom integer encoder padded nonces above 127; its unit oracle called the same helpers. Those duplicate helpers are removed and nonce-boundary expectations now come from unchanged Geth.

Validation passes: all 212 Go test packages, Core integration, affected race for VM/execution/state/block/Node/backend/compare, full vet and build. Thirty-four actual native EVM controls match unchanged Geth v1.17.10; separate controls check two CHAINID widths and an unmodified, signed official Osaka delegation-reentry fixture against its frozen root and unchanged Geth execution. Eleven behavioral mutations were detected; see `mutations.json`. Initial red runs exposed the original defects before the corresponding corrections.

The composed EELS tests@v21.0.2 checkpoint at native fixture source `0b022c7a9597fa60ab1e4e5726543ecf5fb8d257` records Osaka 14,655/14,665 and Amsterdam 16,902/16,908 passes. All remaining failures are signature/chain/recipient/nonce/authorization validation cases owned by the separate validation review; there are no remaining post-state or log mismatches in this checkpoint. This review does not claim the entire program is complete or release-ready. Full independent process/application/recovery/resource acceptance remains in R03-R13.

Normative inputs include EIPs 2, 1014, 1052, 1344, 161, 196, 211, 2200, 2929, 3541, 3860 and 7702, plus the canonical execution-specification arithmetic gas tier. The selected merged LIFO target remains EIPs 3db38a3. This dependent review targets the [completion foundation PR 92](https://github.com/alt-research/eth2030/pull/92), leaving EL master integration for review.
