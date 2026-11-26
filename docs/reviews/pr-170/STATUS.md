# Hegota Group C status

Latest local qualification is complete on implementation `4ac957e9`: all 240 full-suite and 57 focused-race packages pass, with 24,005/9,588 passing events and zero failures. Builds, vet, both binaries, the 12/12 independent baseline and 91 Python checks per mode pass. Mandatory A/B/C consumers are 62/79/92, with zero required skips. The bounded-work audit reproduced 100 raw-code copies for 100 opcodes with recognition disabled; the per-frame/profile cache now performs zero disabled lookups and at most one selected lookup per immutable frame, following frame snapshots. The exact prior-source counterexample and four feature mutations retain positive controls. The exact PR-12041 proposal is fully implemented behind explicit selection; upstream adoption/fork pinning remains distinct. See [CANONICAL_PAYMASTER.md](CANONICAL_PAYMASTER.md). Exact pushed-head hosted results are maintained in the live English PR description/checks.

## Prior initial-proposal qualification

Canonical-paymaster source reconciliation and local qualification are complete on implementation `95a0382f`. The exact #12041 runtime, distinct layout, authorization, treasury/admin operations, public admission/dependencies/exposure and native transport/build/import/reorg/restart are implemented behind explicit proposal selection. A native counterexample exposed and fixed missing candidate-state withdrawal availability. All 240 full-suite and 57 focused-race packages pass (24,004/9,587 passing events, zero failures); native/wrapper builds, full vet, the independent 12/12 baseline and 91 Python checks per mode pass. Required A/B/C consumers are 62/79/91, with zero required skips. Four native feature-mutation controls fail as intended with passing positive controls. Exact pushed-head hosted results are maintained in the live English PR description/checks. See [CANONICAL_PAYMASTER.md](CANONICAL_PAYMASTER.md). Earlier qualifications below remain historical.

## Independent completeness re-audit completed locally

The latest EIPs master is 514f00c369f9586e4d70a25da798ae2bd3d0ead7. The four C
normative documents are text-identical to the pinned references; consensus-specs
and the open canonical-paymaster proposal are unchanged. The complete current
8141/8250/8272/8151 discussion streams and PR #170 reviews were checked. Unmerged
proof-frame, witness-read and verification-budget changes are not adopted.

All forty C criteria were rechecked. The audit reproduced and corrected four
cross-transaction omissions: invalid reservations interfering with valid peers,
hash-only pruning after a payer balance reduction, stale signer-share accounting
during simultaneous witness-path changes, and regular/blob partitioning dropping
an executable scalar successor of a blob Frame. Native frozen-prior-source
controls fail for each omission while ordinary and independent keyed controls
pass. See [the full reconciliation](RECHECK.md).

Final local qualification on `c4fffe3922d534acd88b919e29defffad239e5d7`:

- Full suite: 23,947 passing events, zero failures, 124 optional skips; all 240 packages complete (221 pass, 19 without executed tests).
- Focused race: 9,478 passing events, zero failures, 74 optional skips; all 54 packages complete (51 pass, three without executed tests).
- Native/wrapper builds, full vet and independent historical baseline 12/12 pass.
- Ordinary/optimized Python: 89 checks each. A/B/C required consumers: 62/79/74, all executed with zero required skips. Current source-bound A/B/C evidence passes.
- Reference contents and all independent oracle expectations are unchanged. Previous counterexamples remain preserved alongside the four new native controls.

The upstream canonical-paymaster bytecode gate remains the sole C specification
boundary. Current pushed-head hosted results are maintained in the English PR
description/current checks and are not inferred from the predecessor's 15 passes.
The implementation and evidence below are historical qualifications.

## Current public integration follow-through (2026-11-26)

D01 is resolved for the selected native 8141/8250 two-fee composition by merged author/base history, with the earlier decision preserved in profiles.json. See [COMPOSITION.md](../../plans/hegota-c/COMPOSITION.md). The public eight-field reader, explicit EIP8250Time, nonce-domain-aware pool/order/replacement/dependencies, public RPC/P2P/simulation/build/import/reorg/restart consumers and optional EIP8151Time signer protection are implemented. New native tests cover both activation directions, raw-code gas/BAL behavior and separate signer shares. Typed direct queries and txpool indexing also preserve complete keyed fields and independent replay identities.

Local qualification is complete on implementation revision `3fe23f60bf0ace7e28485dcbe67abe9464649c14`:

- Full suite: 23,935 passing test/subtest events, zero failures, 124 optional skips; 240 completed packages (221 pass, 19 without executed tests).
- Focused race: 9,466 passing events, zero failures, 74 optional skips; 54 completed packages (51 pass, three without executed tests).
- Native/wrapper builds, full vet and the independent 12/12 historical baseline pass. An initial test-only atomic Header copy flagged by vet was replaced with the native header-copy path before final qualification.
- Ordinary/optimized Python: 88 checks each. Required A/B/C consumers: 62/79/70, all executed with zero required skips. Current A/B/C source-bound evidence passes.
- Feature-disabling native overlays fail the keyed reset and EIP-8151 activation/dependency regressions while the ordinary last-nonce control passes. Earlier counterexamples remain preserved.

The upstream canonical-paymaster runtime remains the sole C specification gate; local v1 stays opt-in experimental. Exact pushed-head hosted results are maintained in the English PR description/current checks. The historical figures below are not reused for the new code.

## Historical base-profile qualification

HGT-020–029 mechanisms are locally qualified within the selected profiles, including the follow-up completeness audit. Public composed completion is **not claimed**: D01 and the missing merged reproducible canonical-paymaster runtime remain specification gates. See [the audit](AUDIT.md), [the detailed plan](../../plans/hegota-c/PLAN.md), [the API profile](../../plans/hegota-c/RPC_PROFILE.md) and [the forty-criterion register](../../../specs/hegota/group-c.json).

Implementation source: `0c849a53d86259fffd25b7ee9ee7e46f44a43779`. Subsequent edits are qualification/documentation only. The merged baseline remains `5d777ad06f886c8b20fb010a47e65ddb05df8287`; references and independent oracle expectations were not edited. Upstream EIPs/consensus-specs pins and paymaster PR #12041 were rechecked on 2026-11-26 and remain unchanged.

The follow-up audit corrected four omissions:

1. Cached isolated keyed validation now rechecks each block gas dimension before cache reuse. Equality reuses the validation object; a one-unit execution or state deficit evicts the entry.
2. Pending transaction queries return EIP-1474 null block location fields across native, pool and direct API adapters. Actual plain/blob native consumers verify pending, mined and restarted transaction metadata.
3. The selected Frame simulation API reports trace-limit exhaustion explicitly, accepts exact-limit traces and preserves untraced settlement and state isolation. General tracer truncation behavior is unchanged.
4. Generic public-prefix storage/state-write checks include EIP-1153 transient storage, with caught-helper and constructor controls. Sender transient reads and ordinary private consensus execution remain valid. This is an application of the generic trace clauses, without a new consensus opcode rule.

Final local qualification completed on 2026-11-26:

- Full suite: 23,874 passing test/subtest events, zero failures, 124 optional skips; all 240 packages completed (221 passing, 19 without executed tests).
- Focused race: 9,408 passing events, zero failures, 74 optional skips; all 54 packages completed (51 passing, three without executed tests).
- Native full build, native/Geth-wrapper binary builds, vet and the 12/12 independent historical differential baseline pass.
- Ordinary and optimized Python: 86 checks pass each. Independent encoding/hash, arithmetic/state/semantic and isolated keyed/root expectations are unchanged and pass.
- Mandatory A/B/C consumers: 62/79/54, all executed and passing with zero required skips. C covers ten tasks and forty criteria. Full/race optional reasons are separately inventoried in `evidence/hegota-a/skipped-tests.json`.
- Prior-code native/overlay controls reproduce the four follow-up omissions while retaining the relevant positive controls. Earlier factory-funding/paymaster counterexamples and the caught-INVALID review correction remain recorded honestly in C evidence.

All A/B/C source/evidence checks pass. English draft [PR #170](https://github.com/alt-research/eth2030/pull/170) targets `master`. Exact pushed-head hosted verification and immutable workflow links are maintained in its description/current checks; predecessor results are not reused. The prior qualification at `0894e72d` is historical, and its 15 successful hosted checks do not attest to this follow-up head.
