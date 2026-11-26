# Hegota Group C checklist

## Public integration follow-through (2026-11-26)

The earlier D01 gate below is historical. EIPs #11968 establishes inheritance
from EIP-8141; #12414 changes the inherited fee tuple, corroborated by the
EELS eips/bogota/eip-8250 branch at 6013c4eaed5c9cc1cbb8428541972d6976a8cabd.
The selected composition uses [max_fee, max_priority_fee_per_gas]. Qualification
of its public consumers remains pending until the following work is complete.

- [ ] Integrate the eight-field envelope, signing hash and deep copies with native transaction readers; preserve historical decoding.
- [ ] Add explicit optional EIP-8250 activation with strict pre/post-fork schema validation and no irregular state installation.
- [ ] Connect keyed nonce validation/execution and dependency tracking to public admission, replacement, reorg, builder and import paths.
- [ ] Cover keyed RPC queries/simulation, blob sidecars, P2P and restart with native consumers.
- [ ] Audit EIP-8151's optional activation and signer-class public-pool requirements against its latest merged text.
- [ ] Update decisions, plans and source-bound evidence without claiming an unmerged canonical-paymaster template is normative.
- [ ] Run relevant/full/race tests, build and vet; commit, push and requalify the English PR.

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
Hosted current-head verification remains required. Its live result, exact head SHA and immutable workflow links are recorded in the PR description/current checks, without creating a new source revision solely to attest to itself.

Public specification gates (unresolved, not counted as activated):

- [ ] D01: authoritative EIP-8141/EIP-8250 combined envelope; the nonce-only profile remains isolated.
- [ ] Upstream canonical-paymaster compiled runtime: PR #12041 remains unmerged; the local v1 template remains opt-in research code.

## Follow-up completeness audit (2026-11-26)

- [x] Recheck upstream source identities, active discussions and the two public specification gates.
- [x] Re-read Group C normative clauses and independently compare all forty criteria with production code.
- [x] Exercise missing adversarial boundaries with native regression tests; implement every confirmed omission.
- [x] Requalify changed implementation with builds, relevant/full tests and race checks.
- [x] Refresh source-bound A/B/C evidence, update English PR content and push all authored changes.

The follow-up implementation is `0c849a53d86259fffd25b7ee9ee7e46f44a43779`. All 54 mandatory C consumers pass. Exact pushed-head hosted results are maintained in the English PR description/current checks and remain distinct from local qualification and the two unresolved public gates.
