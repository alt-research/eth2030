# Hegota Group B execution checklist

Baseline: merged master `960baab3103e0d34435433be0f8314f127521052` (PR #168 merged). References remain read-only. Implement from pinned normative clauses and preserve existing native routes; source or helper presence alone does not verify a task.

- [x] Pin/recheck the current EIP-8141/8250/8272 text and dependent gas/BAL/header rules; record clause-level requirements, differences and profile decisions.
- [x] HGT-002: canonical transaction codec, hashes, constants and rejection/width boundary vectors.
- [x] HGT-003: APPROVE identity/scopes, payment/nonce/escrow and nested/repeated/reverting calls.
- [x] HGT-010: full transaction/frame/signature introspection, future result restrictions and calldata boundaries.
- [x] HGT-011: independent execution/state/blob budgets, overflow/floor/refund/fee/reservation and frame isolation.
- [x] HGT-012: call/frame/transaction/batch journals, warm sets, state-charge ownership, logs and transient clearing.
- [x] HGT-013: shared native typed execution/build/import path, prerequisite rules and post-state agreement.
- [x] HGT-014: canonical frame receipt encoding, payer/status/gas/logs, receipt roots and indices.
- [x] HGT-015: published BAL encoding/replay, system accesses, roots and block gas/byte bounds.
- [x] HGT-016: transaction/receipt persistence, replay/restart, old-format readback and migration boundaries.
- [x] HGT-017: admitted/isolated keyed nonce schema, manager deployment, canonical keys, state/BAL/consumption and rollback/CREATE.
- [x] HGT-018: recent-root contract/code, source/slot/window/overwrite/reorg, native slot and verifier integration.
- [x] HGT-019: all adopted signature schemes, domain/hash commitment, lengths/malleability and default accounts.
- [x] Execute clause-level positive/negative tests and native integration/differential qualification; retain exact source and skip provenance.
- [x] Build/vet/test/race and requalify affected A boundaries.
- [x] Push and publish English PR #169.
- [x] Verify hosted checks against their tested revision; final descendant checks are linked on PR #169.

## Second clause-level audit, 2026-11-26

- [x] Recheck upstream text and review every Group B normative requirement against reachable implementation.
- [x] Reproduce and fix any missing execution, rollback, receipt, persistence or controlled-profile behavior.
- [x] Add independent expected-value and negative regression coverage for each finding.
- [x] Refresh Group B and affected Group A source-bound evidence; run local build, vet, full tests and affected race checks.
- [x] Commit/push fixes and verify the published source head and English description; final documentation-only descendant checks are reported on PR #169.

## Third independent clause audit, 2026-11-26

- [x] Recheck current upstream source identity and read all Group B normative clauses against reachable implementation.
- [x] Audit approval/default-account behavior, introspection widths, gas ownership, rollback and nonce domains beyond the previous dispatch fixes.
- [x] Audit codec, receipts, block commitments, persistence, deployment and recent-root boundaries for all 12 tasks.
- [x] Reproduce and correct concrete omissions with unit tests and independent expected values.
- [x] Refresh affected source-bound evidence and complete local build, vet, full and focused race qualification.
- [ ] Commit and push only owned changes, update the English PR description, and verify CI at the published head.
