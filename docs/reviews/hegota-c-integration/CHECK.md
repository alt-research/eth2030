# Hegota Group C checklist

Baseline: merged master `5d777ad06f886c8b20fb010a47e65ddb05df8287`, after PR #169. References are read-only; existing B implementation and independent expectations remain authoritative.

- [ ] Read current EIP-8141/8250/8272 and dependent network/API clauses; audit and correct all HGT-020 through HGT-029 plan requirements before implementation.
- [ ] Record clause-level consensus/public-pool/local-policy/API distinctions, normative ambiguities, activation gates and executable acceptance criteria.
- [ ] HGT-020: all recognized prefixes, budgets, exact-code exceptions, trace dependencies and stopping boundaries.
- [ ] HGT-021: dependency tracking, state/code/fork/reorg invalidation and necessary revalidation.
- [ ] HGT-022: recent-root pool grammar/slot/dependencies and controlled keyed-nonce policy without bypassing D01.
- [ ] HGT-023: canonical/noncanonical paymaster admission, reservations, replacement, release and revalidation.
- [ ] HGT-024: actual fee ranking, replacement, dimensional budgets, count/byte limits and eviction.
- [ ] HGT-025: exact typed/blob P2P announcements, requests, responses, validation and peer bounds.
- [ ] HGT-026: builder selection/revalidation at exact payload state/slot and cancellation safety.
- [ ] HGT-027: submission, query and receipt schema/encoding on actual native RPC.
- [ ] HGT-028: call/simulate/estimate/trace context, two-dimensional fees, bounded execution and no state leakage.
- [ ] HGT-029: external RPC, two-peer propagation, build/import, receipts and restart/reorg end-to-end consumers.
- [ ] Add meaningful positive/negative tests and a complete clause-to-code-to-consumer register; preserve all A/B checks.
- [ ] Build/vet, focused/full/race qualification and source-bound evidence; commit/push and publish an English PR, then verify hosted checks.
