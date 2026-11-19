# Glamsterdam scope and acceptance matrix

This matrix consolidates the 2026-11-18 audits. It describes the recorded implementation baseline, not a newly measured implementation state. All development tasks below are initially open. See the [plan](DEVELOPMENT_PLAN.md) for sequencing and the [decision register](UPSTREAM_DECISIONS.md) for pending specifications.

Evidence classes: **Mismatch** = demonstrated source/behavior difference; **Missing** = required EL path not found; **CL boundary** = actual CL acceptance incomplete; **Foundation** = useful implementation exists but current-spec acceptance remains open. Source and behavior evidence are distinguished in the linked audits.

## Scheduled core EIPs

The authoritative scope input is [EIP-7773 at the audited revision](https://github.com/ethereum/EIPs/blob/3db38a3725105cb67b8804ac3e147967113bc3ca/EIPS/eip-7773.md). Individual specification URLs are linked by EIP number; their selected revisions must be recorded by GL-001 before development.

| EIP | Baseline | Required result and decisive acceptance | Tasks |
| --- | --- | --- | --- |
| [2780](https://eips.ethereum.org/EIPS/eip-2780) | Mismatch | State-independent intrinsic gas, correct runtime recipient/delegation/new-account charges; valid pre-execution OOG remains an included failed transaction with correct nonce/fees/rollback. Pool, estimate, builder and importer agree. | GL-020, GL-021, GL-024, GL-025 |
| [7688](https://eips.ethereum.org/EIPS/eip-7688) | CL boundary | Actual CL progressive-container/list migration, stable gindices/proofs, runtime limits and network size bounds pass selected vectors and fork transition. | GL-054, GL-061 |
| [7708](https://eips.ethereum.org/EIPS/eip-7708) | Mismatch | Only specified nonzero transfers emit logs, in order, including creation endowments; no extra fee-burn logs; reverted effects/logs disappear. Independent receipts roots agree. | GL-032, GL-033, GL-060 |
| [7732](https://eips.ethereum.org/EIPS/eip-7732) | Foundation | Actual CL/builder route authenticates payloads, implements FULL/EMPTY/PENDING weighting and optimistic invalidation, settles payments once, and recovers across gaps/reorgs. | GL-034, GL-050–053, GL-061 |
| [7778](https://eips.ethereum.org/EIPS/eip-7778) | Mismatch | Preserve capped user refunds; exclude them from the appropriate block execution accounting; combine with floor and state dimension without double counting. Storage-clearing transactions expose different user/block quantities when required. | GL-022, GL-024, GL-060 |
| [7843](https://eips.ethereum.org/EIPS/eip-7843) | Mismatch | Canonical slot header/RLP/hash, supplied beacon slot retained through persistence/replay and delivered to EVM/RPC. Nonzero-slot execution and independent hash vectors pass. CL enforces the beacon-slot relationship. | GL-010, GL-011, GL-050 |
| [7928](https://eips.ethereum.org/EIPS/eip-7928) | Mismatch | BAL field changes block hash; canonical recording/encoding passes native vectors; historical bodies retain/serve verified BAL data and P2P responses are correlated to headers. | GL-010, GL-040, GL-041, GL-060 |
| [7954](https://eips.ethereum.org/EIPS/eip-7954) | Mismatch | Audited 64/128 KiB deployed/initcode limits (refresh with selected revision) enforced consistently in execution, pool and estimates; exact limit accepted and limit+1 rejected at intended validation. | GL-030, GL-021, GL-025 |
| [7976](https://eips.ethereum.org/EIPS/eip-7976) | Foundation | Uniform calldata floor composes with selected intrinsic/access-list/state rules. Low-gas invalidity and binding/nonbinding floor cases match independent vectors. | GL-021, GL-024, GL-025 |
| [7981](https://eips.ethereum.org/EIPS/eip-7981) | Mismatch | Access-list byte surcharge independent of byte values, in the correct position relative to the max/floor. Equal-size zero-heavy/nonzero-heavy lists have equal surcharge. | GL-021, GL-024, GL-044 |
| [7997](https://eips.ethereum.org/EIPS/eip-7997) | Mismatch | Canonical factory address/code/nonce and availability by the specified deployment route; no invented fork injection at 0x12; canonical BLS precompile mapping restored and tested. | GL-031 |
| [8024](https://eips.ethereum.org/EIPS/eip-8024) | Foundation | DUPN/SWAPN/EXCHANGE current vectors cover immediates, invalid ranges, stack bounds, code-end behavior, JUMPDEST analysis and activation. | GL-030, GL-060 |
| [8037](https://eips.ethereum.org/EIPS/eip-8037) | Mismatch | Adopted execution/state gas model, total/execution caps, reservoir initialization, charges/refills and call rollback, system budgets, block admission and fee update all agree. Current 25% heuristic removed. Final refill design is gated by U-01. | GL-020–025, GL-060 |
| [8038](https://eips.ethereum.org/EIPS/eip-8038) | Mismatch | Selected access/write constants and operation composition; correct net SSTORE refunds, delegation, value transfer, creation and pre-state checks. Replace guessed TBD constants with frozen normative values. | GL-020–024, GL-060 |
| [8045](https://eips.ethereum.org/EIPS/eip-8045) | CL boundary | Slashed validators excluded from actual proposer selection/lookahead under the selected fork. Native helper divergence resolved if retained as a claimed implementation. | GL-050, GL-061 |
| [8061](https://eips.ethereum.org/EIPS/eip-8061) | CL boundary | Actual CL activation cap, uncapped exit churn, dedicated consolidation churn and weak-subjectivity calculations pass vectors and transitions. | GL-050, GL-054 |
| [8246](https://eips.ethereum.org/EIPS/eip-8246) | Missing | Balance-preserving SELFDESTRUCT and transaction finalization, correct nonce/code/storage clearing, empty-account cleanup and same-transaction later transfers. BAL and receipts agree. | GL-033, GL-040 |
| [8282](https://eips.ethereum.org/EIPS/eip-8282) | Missing | Canonical deposit/exit contracts, request types 0x03/0x04, queues/fees/drain limits and end-of-block extraction; Engine commitment and CL registration/exit readback. | GL-034, GL-050, GL-061 |

Counts: 10 mismatches, two missing EL paths, three CL boundaries, three foundations. Counts must be recomputed from evidence when statuses change; the existence of a task or test file does not change an EIP's acceptance state.

## Dependencies and supporting network protocols

| Requirement | Baseline and remaining work | Tasks |
| --- | --- | --- |
| 7495/7916 required by 7688 | SSZ utilities exist; verify production types and proof consumers rather than counting helper tests or future execution SSZ activation as completion. | GL-054 |
| 7975 — eth/70 | New receipt handlers exist but the node still advertises eth/68. Negotiate versions and their schemas/ranges, correlate requests and validate receipt/root/order/completeness. | GL-041 |
| 8159 — eth/71 | Correlate BAL replies to requested blocks/headers and call hash validation in the production consumer; handle unavailable data and serving limits according to the selected version. | GL-040, GL-041 |
| 8070 — eth/72 | Complete cell-mask/GetCells/Cells wire flow, validation, provider/sampler lifecycle, reconstruction and builder availability; sparse-pool structures alone are insufficient. | GL-042 |
| 8189 — snap/2 | Version negotiation, BAL-based pivot advancement, state application/root checks and reorg recovery; snap/1 implementation is not snap/2 support. | GL-043 |
| 8136 — CL cell deltas | Actual CL partial-message version/group IDs, bitmap/request/provide flows and cross-client exchange. Full-sidecar gossip tests are insufficient. | GL-055 |
| 7904/8261 — informational | Review as evidence for pricing/capacity and selected gas-limit schedule. No arbitrary standalone feature completion credit. Pin executable requirements in the fork manifest. | GL-001, GL-020, GL-062 |

## Production consumers and cross-cutting acceptance

| Requirement | Baseline and decisive acceptance | Tasks |
| --- | --- | --- |
| Fork activation | Separate Amsterdam/Glamsterdan gates leave prior devnet evidence incomplete. Use a documented coordinated feature set; prove every effective rule before/at/after activation. Exclude experimental later-fork activation. | GL-012, GL-061 |
| Header / block RPC | Expose correct BAL commitment and slot; encode canonical fields; latest, historical and post-restart reads agree with raw blocks. | GL-010, GL-011, GL-040 |
| Historical Engine bodies | Production converter omits BAL, alternative backend uses JSON bytes, retention uses 3,533 epochs. Verify both public backend paths, canonical hex/RLP, 14,299-epoch availability, gaps, restart and CL reconstruction. | GL-040 |
| eth_createAccessList | Empty-list placeholder must become actual access discovery under merged API semantics, with matching gasUsed and exclusions. | GL-044 |
| Call / estimate / simulate | Select adopted API method semantics, use correct historical fork context, account for intrinsic/runtime costs and gas caps, and detect caught-inner-failure pitfalls. | GL-025, GL-044 |
| Trace accounting | Regular/state/refund/reservoir dimensions reconciled to receipts and block counters. Exact pending schema gated by U-05. | GL-045 |
| Native official fixtures | Native EF fork table and geth-backed runner are distinct. Current native Amsterdam vectors and independent oracle agreement are required; unsupported oracle cases remain explicit. | GL-002, GL-060 |
| Test effectiveness | Repair stale tagged BAL expectation; negative cases must reach the intended check; targeted mutation must make each critical assertion fail. | GL-003, GL-060 |
| Cold sync and recovery | Import from zero, history/BAL availability, persistence, restart, reorgs and late/missing envelopes exercised on real client processes. | GL-043, GL-052, GL-061, GL-062 |
| Release evidence | Scoped commits, compatible spec/fixture/client revisions, effective fork flags, raw traces, independent roots, failures/recovery and performance limits retained. | GL-063 |

## Evidence inherited from the audits

Both audits are diagnostic, not final conformance runs. The latest recorded build passed; five focused packages passed in the follow-up, while the initial audit's tagged-core suite failed its stale missing-BAL-hash expectation. Native probes demonstrated omitted header commitments, zero-initialized SLOTNUM, missing body BAL, obsolete retention, incorrect gas/refund behavior and extra fee-burn logs. No fresh multi-client devnet or Lighthouse build was performed during those audits.

Open discussion reports are scenario candidates unless corroborated locally. Do not count the disputed RANDAO issue or the explicitly non-live SSZ cross-type example as confirmed local vulnerabilities. Retain their relevant boundary tests where justified.
