# Hegota Group C checklist

Checked task boxes qualify implemented mechanisms in the selected profiles. They do not close the two public specification gates below.

Baseline: merged master `5d777ad06f886c8b20fb010a47e65ddb05df8287`, after PR #169. References are read-only; existing B implementation and independent expectations remain authoritative.

- [x] Read current EIP-8141/8250/8272 and dependent network/API clauses; audit and correct all HGT-020 through HGT-029 plan requirements before implementation.
- [x] Record clause-level consensus/public-pool/local-policy/API distinctions, normative ambiguities, activation gates and executable acceptance criteria.
- [x] HGT-020: all recognized prefixes, budgets, exact-code exceptions, trace dependencies and stopping boundaries.
- [x] HGT-021: dependency tracking, state/code/fork/reorg invalidation and necessary revalidation.
- [x] HGT-022: recent-root pool grammar/slot/dependencies and controlled keyed-nonce policy without bypassing D01.
- [x] HGT-023: canonical/noncanonical paymaster admission, reservations, replacement, release and revalidation.
- [x] HGT-024: actual fee ranking, replacement, dimensional budgets, count/byte limits and eviction.
- [x] HGT-025: exact typed/blob P2P announcements, requests, responses, validation and peer bounds.
- [x] HGT-026: builder selection/revalidation at exact payload state/slot and cancellation safety.
- [x] HGT-027: submission, query and receipt schema/encoding on actual native RPC.
- [x] HGT-028: call/simulate/estimate/trace context, two-dimensional fees, bounded execution and no state leakage.
- [x] HGT-029: external RPC, two-peer propagation, build/import, receipts and restart/reorg end-to-end consumers.
- [x] Add meaningful positive/negative tests and a complete clause-to-code-to-consumer register; preserve all A/B checks.
- [x] Build/vet, full/race qualification, mandatory A/B/C consumers and source-bound evidence.
- [x] Push and publish English draft PR #170 in `alt-research/eth2030`.
- [ ] Verify current-head hosted checks.

Public specification gates (unresolved, not counted as activated):

- [ ] D01: authoritative EIP-8141/EIP-8250 combined envelope; the nonce-only profile remains isolated.
- [ ] Upstream canonical-paymaster compiled runtime: PR #12041 remains unmerged; the local v1 template remains opt-in research code.
