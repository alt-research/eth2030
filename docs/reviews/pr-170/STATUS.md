# Hegota Group C status

All HGT-020–029 implementation mechanisms are locally qualified within the selected profiles. Public composed completion is **not claimed**: D01 and the missing merged reproducible canonical-paymaster runtime remain specification gates. See [the audit](AUDIT.md), [the detailed plan](../../plans/hegota-c/PLAN.md), [the API profile](../../plans/hegota-c/RPC_PROFILE.md) and [the forty-criterion register](../../../specs/hegota/group-c.json).

Implementation source: `6f54ec223efd698e888fc4d87273c3880d420b1c`, including bounded timestamp/root expiry heaps, exact candidate context, native malformed-DATA rejection and the factory-funding dependency correction. Subsequent edits are qualification/documentation only. The merged baseline remains `5d777ad06f886c8b20fb010a47e65ddb05df8287`; references/oracles were not edited.

Local qualification completed on 2026-11-26:

- Full suite: 23,857 passing test/subtest events, zero failures, 124 optional skips; all 240 packages completed (221 passing, 19 without executed tests).
- Focused race: 9,391 passing test/subtest events, zero failures, 74 optional skips; all 54 packages completed (51 passing, three without executed tests).
- Native full build, native and Geth-wrapper binary builds, vet and the 12/12 independent historical differential baseline pass.
- Ordinary and optimized Python: 85 checks pass each. Encoding/hash, arithmetic/state/semantic and isolated keyed/root oracles remain unchanged and pass.
- Mandatory consumers: A 62, B 79, C 50, all executed and passing, with zero required skips. C covers ten tasks and forty criteria. Full/race optional reasons are recorded separately in `evidence/hegota-a/skipped-tests.json`.
- The actual two-peer plain/blob path covers HTTP submission, fetch/admission, ordinary pool building without IL injection, independent import/FCU, exact bytes, state/receipts and restart. Authenticated simulation/trace retains no canonical or pool effects.
- A prior-code native counterexample admits both forbidden CREATE/CREATE2 factory-funding cases; the current implementation rejects them while preserving the approved-factory-payer exception and private consensus validity.
- A prior isolated-policy overlay fails the code-adoption paymaster-cap counterexample. Caught INVALID also passes against the prior interpreter and is correctly recorded as extra coverage, not a new defect.

All A/B/C source/evidence checks pass. Shared-payer exposure uses already-validated prefix classification, avoiding repeated verification of unrelated signatures. English draft [PR #170](https://github.com/alt-research/eth2030/pull/170) is published from `codex/hegota-c-integration` against `master`. Exact-head hosted verification and immutable workflow links are maintained in the PR description/current checks. They remain required in addition to this source-bound local record; predecessor results are not reused for a new head.

## Follow-up completeness audit

A fresh clause-to-production-code audit is in progress at the user's request, starting from `0894e72d999bdd27b9515dd3264d8227fdd3ded8`. Prior success records are retained as historical qualification, not substituted for qualification of any follow-up changes. Upstream EIPs master and paymaster PR #12041 were rechecked: their identities and open specification gates are unchanged.

Two follow-up omissions are fixed: cached keyed validation now rechecks both block gas dimensions, and pending transaction queries return EIP-1474 null location fields across the native, pool and direct API adapters. The two dimensional cases fail on the old implementation. A prior-code API overlay fails all three pending controls while all three mined controls pass; current native plain/blob propagation, build/import and restart tests pass with pending/mined/restarted location assertions. Complete final-source qualification and refreshed evidence are pending. Earlier runs started before the API/test changes are not used as final-source qualification.

The follow-up audit also reproduced silent trace truncation in the selected simulation API and corrected it with a bounded overflow sentinel. Exact-limit, untraced settlement and no-state-effect controls pass; native ordinary `eth_simulateV1` accepts an ordinary call and rejects incomplete type-6 input. Mandatory C consumers now total 53. Final qualification will use a new full/race run after this last implementation change; the earlier successful 23,867/9,401-event runs belong to the preceding implementation and are not reused for the trace correction.

The final trace-rule sweep identified omitted transient-storage coverage: public deployment could perform factory TLOAD/TSTORE or constructor TSTORE. These are now checked under EIP-8141's generic storage/state-write rules using EIP-1153's storage semantics, including a caught-helper negative control and sender-read/private-execution positive controls. Required C consumers now total 54. Final-source qualification will be regenerated after this correction.
