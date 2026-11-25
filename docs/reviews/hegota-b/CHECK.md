# Hegota Group B execution checklist

Baseline: merged master `960baab3103e0d34435433be0f8314f127521052` (PR #168 merged). References remain read-only. Implement from pinned normative clauses and preserve existing native routes; source or helper presence alone does not verify a task.

- [ ] Pin/recheck the current EIP-8141/8250/8272 text and dependent gas/BAL/header rules; record clause-level requirements, differences and profile decisions.
- [ ] HGT-002: canonical transaction codec, hashes, constants and rejection/width boundary vectors.
- [ ] HGT-003: APPROVE identity/scopes, payment/nonce/escrow and nested/repeated/reverting calls.
- [ ] HGT-010: full transaction/frame/signature introspection, future result restrictions and calldata boundaries.
- [ ] HGT-011: independent execution/state/blob budgets, overflow/floor/refund/fee/reservation and frame isolation.
- [ ] HGT-012: call/frame/transaction/batch journals, warm sets, state-charge ownership, logs and transient clearing.
- [ ] HGT-013: shared native typed execution/build/import path, prerequisite rules and post-state agreement.
- [ ] HGT-014: canonical frame receipt encoding, payer/status/gas/logs, receipt roots and indices.
- [ ] HGT-015: published BAL encoding/replay, system accesses, roots and block gas/byte bounds.
- [ ] HGT-016: transaction/receipt persistence, replay/restart, old-format readback and migration boundaries.
- [ ] HGT-017: admitted/isolated keyed nonce schema, manager deployment, canonical keys, state/BAL/consumption and rollback/CREATE.
- [ ] HGT-018: recent-root contract/code, source/slot/window/overwrite/reorg, native slot and verifier integration.
- [ ] HGT-019: all adopted signature schemes, domain/hash commitment, lengths/malleability and default accounts.
- [ ] Execute clause-level positive/negative tests and native integration/differential qualification; retain exact source and skip provenance.
- [ ] Build/vet/test/race, requalify affected A boundaries, commit/push and publish an English PR with exact-head checks.
