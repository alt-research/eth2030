# Fourth Group B clause audit, 2026-11-26

The review rechecked all 12 Group B tasks directly against the development plan and pinned normative text, rather than assuming that the existing criterion register was exhaustive. Upstream EIPs master still resolves to `ade9e9cab7e148bba4cd293f916a4a189cd7a8aa`, consensus-specs master to `d2021d2719235f0df4726a1d330502cb15aa8ea3`, and upstream project master to `960baab3103e0d34435433be0f8314f127521052`. Reference contents were not edited.

## Account reconstruction lost the balance baseline

EIP-8037, Gas accounting for new accounts, permits CREATE/CREATE2 at a balance-only address without charging for a new account leaf. The existing native creator rebuilds that account and restores its prefunding. Frame BAL tracking previously intercepted the later AddBalance, SetNonce and SetCode calls, but did not intercept CreateAccount. It therefore captured balance zero after the reset, instead of the transaction-original balance.

This violates EIP-7928, Balance (`balance_changes`): unchanged pre/post balances must be omitted. With seven wei of prefunding and no endowment, the old implementation published a false 0-to-7 change. With three wei of endowment it used 0-to-10 instead of 7-to-10. A reverted constructor, enclosing frame or atomic batch could still leave a false balance change in the published BAL, although canonical state correctly restored the original seven wei. State agreement alone did not expose this omission.

The Frame StateDB wrapper now captures original balance, nonce and code before CreateAccount resets the account. Later writes and rollbacks continue to use the existing shared journal. Reconciliation still retains accessed addresses while emitting only net transaction changes. No executor, wire format, fee rule or storage schema is added.

## Executable witnesses and independent checks

- `TestFrame8141PrefundedCreationBAL`: eight CREATE/CREATE2 cases covering successful deployment versus constructor revert and zero versus three-wei endowment. It checks original/final balances, omitted net-zero changes, nonce/code, and the exact 1530 state-gas code deposit without a new-account charge.
- Those eight call bodies also run in unchanged upstream Geth v1.17.10 Amsterdam. Destination balances, code, nonce and gas agree after accounting for the Frame-only entry access and the explicit state deposit. This compares ordinary creation semantics, not an independent Frame envelope or BAL producer.
- `TestFrame8141PrefundedCreationRollbackBAL`: four CREATE/CREATE2 cases covering enclosing Frame exception and atomic-batch rollback. The destination and creator return to pre-state, deposit state gas is zero, prior payment approval persists, and a later frame accesses the restored destination. BAL retains the account access with no balance, nonce or code changes.
- `TestCanonicalFrame8141PrefundedCreateBuildImport`: the production native builder, processor replay and independent block importer preserve prefunding and agree on state/BAL commitments. The published BAL contains the deployed nonce/code but no unchanged balance entry.

A Go overlay restores only `pkg/core/vm/frame_8141_bal.go` from `2a092e4344982614c3189437cf3bf5ebc28b1e56` while keeping the final regression tests. All three new parent tests fail, with 15 failed test/subtest events across execution and block packages. The corrected source passes. This is an actual assertion-failure control, not a compile failure or altered oracle. The record retains the original revision/file hash and negative-run digest.

## Complete task review

| Task | Fourth audit result |
| --- | --- |
| HGT-002 | Rechecked canonical envelope, integer/field boundaries and independent transaction/signing hash consumers. No encoding correction found. |
| HGT-003 | Rechecked APPROVE identity, scope, repeated/nested approval, current nonce consumption, escrow and RETURN memory semantics. Account reconstruction does not change approval rules. |
| HGT-010 | Rechecked parameter selectors, full-width operands, future structures/completed results, calldata padding/copy and arbitrary-only witness visibility. Existing exact consumers remain required. |
| HGT-011 | Added prefunded CREATE/CREATE2 execution/state-gas witnesses with upstream comparison. Independent reservations, refills, refund/floor, blob fees and clipped tip remain required. |
| HGT-012 | Added constructor, enclosing Frame and atomic-batch rollback witnesses for original account fields and state deposits. Existing approval/nonce, storage owner, logs, refund, warm-set and transient rollback tests remain required. |
| HGT-013 | Added real native build/replay/import coverage for the corrected published BAL. Existing signed transaction, blob, reorg and restart witnesses remain required. |
| HGT-014 | Rechecked payer/frame receipt encoding, invalid effects, logs/indices, receipt roots and derived fields. Receipt/state gas matches the existing deposit/rollback rules. |
| HGT-015 | Fixed transaction-original account fields during reconstruction. Rechecked previously corrected sentries, precompiles, delegation, SSTORE and SELFDESTRUCT accesses, final net changes, system updates and block size/item bounds. |
| HGT-016 | Rechecked raw canonical/historical readback, FileDB recovery/replay and canonical fork changes. No storage migration is introduced. |
| HGT-017 | Rechecked isolated keyed/legacy domains, pricing, exhaustion, atomic consumption, non-warming BAL bookkeeping, retry and CREATE introspection. No new composition resolution exists; D01 stays open. |
| HGT-018 | Rechecked ordinary deployment/runtime, domains/tuples, actual header slot, cold-key gas, duplicates, window, overwrite and canonical fork switching. Pool extension work remains Group C. |
| HGT-019 | Rechecked SECP256K1/P256 validation and default-account selection, explicit/empty digest commitments, low-s/length checks and arbitrary-witness separation. No new signature change found. |

The unchanged 36 criteria now require 78 native test identities. The checker explicitly requires both lifecycle tests and the real block test, and its negative tests reject removing them from the relevant criteria. No criterion was removed or relaxed.

## Compatibility and qualification

The correction changes the BAL commitment for research blocks that exercised the faulty reconstruction path. Such history may need rebuilding from a valid checkpoint. Roots are not silently rewritten, and public-chain database migration is not performed. Earlier successful state/CI results remain historical evidence and do not qualify the corrected source.

EIP-8250 still contains the fee-field conflict with current EIP-8141. HGT-017 remains complete only under the development plan's isolated state/vector allowance; no public composed decoder, activation, pool/import path or database migration is enabled.

Fresh local source-bound qualification is recorded in the A/B evidence. Hosted checks are reported against their tested commit on PR #169.

## Fresh local qualification

Functional revision `52a84ba38b62ae6aea35a9815143d75a5d04773f` completed all 240 packages with 23,761 named passes, zero failures and 124 explicit skips. The focused race run completed all 36 packages with 8,001 named passes, zero failures and 74 explicit skips. All 78 mandatory B tests and 62 mandatory A tests passed. Build/vet, both CLI builds, the 12-case historical differential baseline and 75 Python tests in normal/optimized modes passed. Source/log digests and individual skip reasons are retained in the evidence.
