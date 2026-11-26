# Hegota Group C checklist

## Canonical paymaster source and discussion audit

- [ ] Re-read current EIP-8141, its Solidity asset, PR #12041's complete history/reviews and current implementer discussion; distinguish merged requirements from the proposed runtime.
- [ ] Reconcile runtime recognition, authorization, storage, delayed withdrawal/rotation, EIP-8151 and pool lifecycle requirements against production paths.
- [ ] Implement every determinate omission with independent native positive/negative controls and explicit profile selection where upstream proposals diverge.
- [ ] Verify builds, relevant/full/race tests and source-bound A/B/C evidence; commit/push and update the English PR with exact current-head results.

## Independent completeness re-audit

- [x] Recheck upstream EIP/consensus identities, merged changes and live discussion/review status.
- [x] Re-read every C normative clause and trace actual admission, revalidation, builder, transport and RPC consumers; record all forty criteria in RECHECK.md.
- [x] Exercise simultaneous failures and shared-resource transitions; reproduce invalid reservation, expiry/priority, stale signer-share and mixed blob/scalar ordering omissions.
- [x] Fix all four confirmed omissions with native prior-source red controls and preserved positive controls; leave references and independent oracles unchanged.
- [x] Requalify the final source with full/race/build/vet/baseline and 89 Python checks per mode; refresh A/B/C evidence and prepare the English PR publication.

Exact pushed-head hosted results are recorded in the live English PR description
and checks. The canonical-paymaster upstream runtime boundary remains explicit.

## Public integration follow-through (2026-11-26)

The earlier D01 gate below is historical. EIPs #11968 establishes inheritance
from EIP-8141; #12414 changes the inherited fee tuple, corroborated by the
EELS eips/bogota/eip-8250 branch at 6013c4eaed5c9cc1cbb8428541972d6976a8cabd.
The selected composition uses [max_fee, max_priority_fee_per_gas]. Qualification
of its public consumers remains pending until the following work is complete.

- [x] Integrate the eight-field envelope, signing hash and deep copies with native transaction readers; preserve historical decoding.
- [x] Add explicit optional EIP-8250 activation with strict pre/post-fork schema validation and no irregular state installation.
- [x] Connect keyed nonce validation/execution and dependency tracking to public admission, replacement, reorg, builder and import paths.
- [x] Cover keyed RPC queries/simulation, blob sidecars, P2P and restart with native consumers.
- [x] Audit EIP-8151's optional activation and signer-class public-pool requirements against its latest merged text.
- [x] Update decisions, plans and source-bound evidence without claiming an unmerged canonical-paymaster template is normative.
- [x] Run relevant/full/race tests, build and vet; prepare committed source-bound evidence and the English PR update.

Pushed-head hosted qualification is recorded in the live English PR description/current checks; it is distinct from local qualification and must match the actual final head.

Historical checklist entries below describe the earlier qualification. Current composition and its remaining canonical-runtime gate are documented in COMPOSITION.md and the follow-through section above.

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

Public specification decisions and remaining gate:

- [x] D01: selected two-fee EIP-8141/EIP-8250 composition resolved by merged history and qualified public consumers; explicit optional activation, no mainnet schedule. Independent oracle provenance is preserved.
- [ ] Upstream canonical-paymaster compiled runtime: PR #12041 remains unmerged; the local v1 template remains opt-in research code.

## Follow-up completeness audit (2026-11-26)

- [x] Recheck upstream source identities, active discussions and the two public specification gates.
- [x] Re-read Group C normative clauses and independently compare all forty criteria with production code.
- [x] Exercise missing adversarial boundaries with native regression tests; implement every confirmed omission.
- [x] Requalify changed implementation with builds, relevant/full tests and race checks.
- [x] Refresh source-bound A/B/C evidence, update English PR content and push all authored changes.

The follow-up implementation is `0c849a53d86259fffd25b7ee9ee7e46f44a43779`. All 54 mandatory C consumers pass. Exact pushed-head hosted results are maintained in the English PR description/current checks and remain distinct from local qualification and the two unresolved public gates.
