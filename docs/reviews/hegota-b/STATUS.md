# Hegota Group B status

Baseline: merged master `960baab3103e0d34435433be0f8314f127521052` (PR #168). The branch reuses the existing seven-field canonical transaction, shared native frame executor, journals, block builder/importer, BAL and receipt persistence.

Implementation in progress:

- Added the isolated EIP-8250 nonce-only delta profile: canonical eight-field RLP and length-committing signing hash, sorted uint256 key sets, intrinsic/floor access pricing, pre-state sequence checks, atomic APPROVE consumption, first-use state gas, non-warming protocol accesses, BAL writes and TXPARAM 0x0D–0x10. It is deliberately not a `TxData` and cannot enter public decode, block import, pool or configured networks.
- Reused native EVM execution for the verbatim EIP-8272 ordinary contract. Added fixed-domain source/entry/storage helpers, packed tuple parsing and verifier classification; deployment is never installed at a fork.
- Added real native keyless deployment tests from both EIPs, nonce overlap/disjoint/single-use/CREATE/rollback/BAL tests, recent-root slot/window/gas/reorg/overwrite/context tests and the base APPROVE/introspection negative matrices.
- Tightened canonical receipt validation: failed frames cannot carry state gas or logs. Approval nonce overflow exceptionally halts rather than becoming an ordinary REVERT.

Specification boundary: EIP-8250 says to replace only the nonce and otherwise preserve EIP-8141, but still shows a contradictory three-fee example. D01 is not closed. The isolated delta profile follows that explicit replacement instruction and 8141's two-field total fee budget; it is not represented as an upstream-resolved combined wire format. EIP-8272 needs no envelope change. Public-pool extension work belongs to HGT-020–022 (Group C), not Group B.

Verification so far: focused types/execution/block/chain tests pass, including both ordinary keyless deployments. Full qualification and the complete 12-task clause register remain in progress. No Group B task is marked verified solely from existing implementation presence.
