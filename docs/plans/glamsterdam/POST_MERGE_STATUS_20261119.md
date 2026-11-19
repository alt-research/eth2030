> Active continuation: [Completion progress](COMPLETION_PROGRESS_20261119.md). This audit remains the historical master-5fbc7c13 baseline.

# Glamsterdam post-merge status and remaining work

Snapshot: 2026-11-19. Audited EL master: [`5fbc7c13`](https://github.com/alt-research/eth2030/commit/5fbc7c133b0e57352ca2a3d6fdd9a63726c42502). [Structured inventory](post-merge-status-20261119.json). This report supersedes historical pending/absent labels in the development tracker; it does not erase their original evidence.

## Result and counting rule

All **24 EL reviews 64–87 are merged** and the EL repository has **zero open PRs** at this snapshot. Master has seven successful CI checks. Its complete `pkg/` tree equals the previously qualified final trace composition, so an implementation is not missing merely because an older status table calls its review open.

The remaining scope is organized into **13 deduplicated work packages**. These are not 13 missing production modules, 13 extra PRs or a percentage. The concrete confirmed EL findings are **one production correctness mismatch** and **one missing native fixture infrastructure path**. A separate integration package contains **three already implemented but unmerged Lighthouse reviews**. Nine packages concern protocol/security/API/application/operational qualification, some with conditional fixes, and one covers final input/release closeout.

| Primary category | Packages | Count |
| --- | --- | --- |
| Confirmed EL correctness defect | R01 | 1 |
| Native fixture infrastructure and independent corpus | R02 | 1 |
| Implemented CL code pending integration | R05 | 1 (three PRs) |
| Protocol/security/application/API/full-process/recovery/capacity qualification | R03, R04, R06–R12 | 9 |
| Compatible final target and release closeout | R13 | 1 |

An open package means sufficient closing evidence was not found in the audited source/records; it is not a claim about every possible external run. Confirmed code findings and unclosed qualification are separated below.

## What is already delivered

Canonical BAL/slot headers and slot propagation, fork/Engine/configuration authority, contract/initcode limits and stack encoding, factory/precompiles, ETH logs and SELFDESTRUCT lifecycle, mandatory builder requests, configured blob capacity/prices/fees, historical BAL serving, negotiated eth/68–72, sparse reconstruction/custody, snap/2, selected LIFO gas, access discovery, method-specific RPC, stateful simulation and real nested/dimensional traces are all on EL master with development tests. Broad final acceptance remains distinct from this implementation statement.

In particular, the PR 83 task **A83-06 is closed for development integration**: #83 and method reviews #85–#87 are merged, and the combined source has the actual HTTP/historical/override/gas reconciliation tests and seven successful mainline checks. They must not be implemented again. The old A83-07 Draft/merge-readiness step is also past; final protocol/release review remains part of R13.

## Confirmed findings after the merges

1. **Historical EIP-7702 warming still differs.** A fresh signed run on source-identical audit checkpoint `80169cdc` matches **42/42 Amsterdam cases**, but Osaka controls are **35/42 matches**. `authorization_bad_nonce_warms` is valid on both engines: native pays **48605 gas**, independent Geth v1.17.10 pays **46105**, and roots/balances differ. [The legacy helper](../../../pkg/core/eips/eip7702.go#L61) returns on the nonce check before warming the recovered authority; the selected EIP requires warming first. This case also exists in the earlier clean PR 83 checkpoint, so it is not caused by the merge repairs. The other six control differences are declared-cap direct-transition boundaries and require R04 entrypoint qualification, rather than being counted as six additional proven consensus bugs.

2. **Native official Osaka/Amsterdam fixture registration is absent.** [Native ForkConfig](../../../pkg/core/eftest/state_test_runner.go#L396) stops at Prague. The dedicated Geth-backed EF job calls [gethcore.ApplyMessage](../../../pkg/core/eftest/geth_runner.go#L247). Main CI does include native Go tests and all 42 selected native/reference Amsterdam transitions; the missing part is official fixture execution by the native Amsterdam adapter, not all native testing.

3. **CL review integration is separate from merged EL PRs.** [Lighthouse #1](https://github.com/jiayaoqijia/lighthouse/pull/1), [#2](https://github.com/jiayaoqijia/lighthouse/pull/2), [#3](https://github.com/jiayaoqijia/lighthouse/pull/3) remain open against `unstable` `8104a246`. They implement first-slot availability, independent onboarding and a real-BLS consumer of exact EL request bytes. The already tested local composition is `44324117` atop paired base `f85fa05b`; its archived 25 component tests/check/lint do not establish a fresh release build or live full-client campaign. This audit reads CL source and review state; it does not run new Rust tests.

The new signed report is `/tmp/eth2030-postmerge-corpus.json`; compact outcomes are included in the structured inventory. `go run ./cmd/glamsterdam-compare` returns a report even when controls differ, so zero process exit is not an all-forks parity assertion. The full local Go build also passed. No business code or reference submodule is edited by this audit.

## Remaining packages and completion conditions

Priorities describe engineering/release importance, not a request to re-authorize already selected development work. Current LIFO work stays authorized; open alternatives are conditional inputs.

### R01 — Historical EIP-7702 skipped-authorization warming

**P0; confirmed EL correctness defect.** Original tasks: GL-023, GL-060.

- [ ] Fix and independently regress authority warming before nonce/code rejection; paid gas 48605 versus 46105 and post-root must agree at Osaka, while preserving all Amsterdam cases.

### R02 — Native Osaka/Amsterdam fixtures and independent envelope corpus

**P1; missing native test infrastructure and qualification.** Original tasks: GL-002, GL-021, GL-022, GL-030, GL-060.

- [ ] Register coherent native fork contexts, pin selected official fixtures and expected counts, fail unsupported/missing required cases, and extend independent signed coverage beyond 32 type-2 and 10 type-4 cases to legacy/type-1/blob boundaries.

### R03 — Application and GAS/reservoir compatibility

**P1; application qualification.** Original tasks: GL-022, GL-032, GL-060.

- [ ] Run real wallet/router/delegation/failure-handler patterns, low forwarded execution with funded reservoir, nested refill/revert and side-effect/fee invariants; evaluate the selected model without silently adopting open alternatives.

### R04 — Outer limits and complete API/historical fixtures

**P1; API and entrypoint qualification.** Original tasks: GL-021, GL-024, GL-025, GL-044, GL-045, GL-060.

- [ ] Exercise correctly bound cap/floor/state-budget negative payloads in Node and memory backends, plus official call/estimate/access/simulate/trace and historical/override matrices. Six direct-transition cap controls are not six proven acceptance bugs; the prior oversized fixture did not reach receiver validation.

### R05 — Integrate the three Lighthouse reviews and compatible client artifact

**P1; implemented CL code pending integration.** Original tasks: GL-011, GL-034, GL-050.

- [ ] Review/merge or otherwise publish a compatible candidate carrying CL PRs 1/2/3, pin it against EL 5fbc7c13, rerun actual CL transition/BLS/request/slot component tests and complete builds, including payload/beacon slot binding, adopted signing domains, slashed-proposer exclusion and churn/weak-subjectivity fork boundaries. Existing 25-test candidate 44324117 is component evidence, not final live acceptance.

### R06 — Payload authentication and optimistic lifecycle

**P1; CL security/fork-choice qualification with conditional implementation.** Original tasks: GL-050, GL-051, GL-052.

- [ ] Freeze the selected authentication format, isolate contents/claimed-hash/request binding and seen-cache negatives, then qualify FULL/EMPTY/PENDING, latestValidHash, late/missing envelopes, descendant invalidation, empty heads and restart/reorg. Open 5770/5820/5781/5818 are not all mandatory patches.

### R07 — Builder settlement exactly once

**P1; CL invariant reproduction and qualification.** Original tasks: GL-053.

- [ ] Reproduce late-child/epoch-eviction/slashed/no-quorum/exit cases in actual selected CL and assert debits, credits, withdrawals and supply. Upstream issue 5754 is open; this audit demonstrates no local double-payment exploit.

### R08 — Progressive migration and real proof consumers

**P1; CL SSZ/proof qualification.** Original tasks: GL-054.

- [ ] Qualify nonzero fork-state migration, stable generalized indices, independent proof/root vectors, runtime sizes/limits and actual light-client consumers. Code already generates progressive proofs; the unused unsupported-error comment is not a missing implementation.

### R09 — Partial columns, version/group IDs and custody recovery

**P1; CL data/network qualification.** Original tasks: GL-042, GL-055.

- [ ] Cross real CL peers at the selected Gloas boundary with group/version, bitmap and cell ordering under loss/duplication/reconnect; compose actual EL custody/cell methods. Existing native eth/72 and progressive CL data types are retained evidence.

### R10 — Full independent multiclient and external-builder devnet

**P1; independent full-process qualification.** Original tasks: GL-011, GL-031, GL-034, GL-040, GL-041, GL-042, GL-043, GL-061.

- [ ] Use current native EL plus a compatible independent EL, actual Lighthouse plus independent CL, and external builder across a coordinated fork with contracts/blobs. Preserve exact revisions, actual Engine/P2P wire, independent commitments and rejection controls.

### R11 — Long cold sync, retention, reorg and durable recovery

**P1; recovery and retention qualification.** Original tasks: GL-040, GL-041, GL-043, GL-052, GL-062.

- [ ] Run native and independent process cold sync, missed-window/orphan history, pruning/retention edges, restart/reorg and storage-failure campaigns. Existing small encrypted/native recovery scenarios are not erased and do not establish the full campaign.

### R12 — Measured CPU/memory/disk/latency and queue budgets

**P1; operating-capacity qualification.** Original tasks: GL-042, GL-062.

- [ ] Set workload/hardware/target schedules and thresholds, then measure valid/adversarial execution/state/blob mixes, historical serving, cancellation and durable storage/queue pressure. CPSB rationale and successful block production are not measured ETH2030 operating headroom.

### R13 — Final compatible target, scope, artifact manifest and runbooks

**P0 release; input freeze and release closeout.** Original tasks: GL-001, GL-012, GL-020, GL-025, GL-045, GL-051, GL-063.

- [ ] Record final compatible EL/CL/Engine/devp2p/API/fixture/runtime/network inputs, resolve the declared gas/auth/schema choices, retain all unknowns, then publish qualified artifacts and upgrade/recovery procedure after gates pass. Merging the development model closes PR integration, not external protocol adoption.

## Reconciliation of all 33 original task IDs

The original plan still defines acceptance. The following **development classes** have different meanings: **D** = EL implementation delivered for the selected exercised target; **P** = partial/cross-client/compatibility or final-selection boundary; **V** = program verification/release work. D does not waive the shared release gates, and P does not mean all code is absent. Counts are **16 D, 12 P, 5 V**. Do not add these 33 overlapping task IDs to the 13 deduplicated packages or turn 16/33 into a release-completion percentage.

| Task | Class | Current assessment / remaining package |
| --- | --- | --- |
| GL-001 | V | Release input/client/fixture tuple is not finally frozen. **R13.** |
| GL-002 | P | Native fixture names/configuration stop at Prague; official native Amsterdam support remains missing. **R02.** |
| GL-003 | D | Original diagnostic repairs and intended regressions are implemented; newer findings remain separately tracked. **Original local implementation/regression delivery closed.** |
| GL-010 | D | Canonical BAL/slot codec, hashes and independent header tests are on master. **Original local implementation/regression delivery closed.** |
| GL-011 | P | EL slot propagation and actual HTTP are implemented; complete paired CL slot-boundary acceptance remains. **R05, R10.** |
| GL-012 | P | Canonical schedule/configuration/Engine windows and persisted authority exist; final release feature/network tuple remains. **R13.** |
| GL-020 | P | Selected LIFO model is implemented; final release-model decision remains separate. **R13.** |
| GL-021 | D | Ordinary intrinsic/runtime split and dual reservation are implemented; broader fixture/entrypoint matrices remain. **R02, R04.** |
| GL-022 | D | Selected frame/state accounting and rollback are implemented; official/application acceptance remains. **R02, R03.** |
| GL-023 | P | Amsterdam authorization/account lifecycle exists; historical skipped-authorization warming still differs. **R01, R02.** |
| GL-024 | D | Paid/gross/state settlement and bottleneck header/base fee are implemented and tested. **R02, R04.** |
| GL-025 | D | Call/estimate/access/simulate/trace are all merged; final API fixtures/schema acceptance remains. **R04, R13.** |
| GL-030 | D | Contract/initcode limits and current stack encoding are implemented; full native fixture combinations remain. **R02.** |
| GL-031 | D | Canonical factory/precompile and explicit allocation paths exist; actual process deployment remains a campaign. **R10.** |
| GL-032 | D | Transfer logs, ordering and rollback are implemented including tracing composition. **R02, R03.** |
| GL-033 | D | Balance-preserving destruction and final-state/BAL reconciliation are implemented. **R02.** |
| GL-034 | P | EL mandatory queues/contracts and real-BLS CL component evidence exist; CL reviews and full transport/lifecycle remain. **R05, R10.** |
| GL-040 | D | Canonical persisted BAL bodies, availability window and recovery paths exist; independent history/retention campaigns remain. **R10, R11.** |
| GL-041 | D | Negotiated eth/68–71 and correlated historical consumers are implemented; independent long sync remains. **R10, R11.** |
| GL-042 | D | eth/72 sparse relay/custody/reconstruction is implemented; complete live CL/independent gossip and budgets remain. **R09, R10, R12.** |
| GL-043 | D | snap/2 actual moving-pivot/reorg/restart/proof consumers are implemented; full independent cold sync remains. **R10, R11.** |
| GL-044 | D | Actual access discovery and returned-list gas are implemented; final API/historical fixture scope remains. **R04.** |
| GL-045 | D | Actual nested/dimensional tracing is implemented; adopted schema/fixture acceptance remains. **R04, R13.** |
| GL-050 | P | Actual CL fork/onboarding patches exist in separate open reviews; compatible client integration and transition matrix remain. **R05, R06.** |
| GL-051 | P | Signed envelope checks exist; selected authentication choice and adversarial binding/cache qualification remain. **R06, R13.** |
| GL-052 | P | CL processing/optimistic paths exist; complete per-payload FULL/EMPTY/PENDING/late/reorg/restart matrix is not closed. **R06, R11.** |
| GL-053 | P | Settlement code exists; issue variants and settlement-once invariants remain unreproduced/unqualified locally. **R07.** |
| GL-054 | P | Actual progressive beacon types and proof generation exist; independent nondefault migration/proof qualification remains. **R08.** |
| GL-055 | P | Gloas partial-column types/filters exist; real-peer version/group/loss/duplication recovery qualification remains. **R09.** |
| GL-060 | V | Selected native official/deployed application and complete API/execution corpus remains incomplete. **R02, R03, R04.** |
| GL-061 | V | Fresh independent multi-EL/multi-CL/external-builder fork campaign remains unrun for this complete head. **R10.** |
| GL-062 | V | Small native restart/reorg tests exist; long recovery/retention and measured operating budgets remain. **R11, R12.** |
| GL-063 | V | Development code/CI and this reconciliation are delivered; qualified release artifacts/runbook and all acceptance gates remain. **R13.** |

## Stale indicators deliberately excluded

- The old `IMPLEMENTATION.md` statement that GL-025, eth/72, snap/2 or current LIFO still need implementation is superseded by the merged reviews. Their broader acceptance scope remains in the packages above.
- The custody TODO in `engine/lighthouse_compat.go` is not evidence of missing canonical custody adoption. [Current FCU handling](../../../pkg/engine/handler.go#L594) parses the mask and calls SetCustodyColumns; the actual Node adapter forwards it to the eth handler.
- The CL `ProgressiveMerkleProofNotSupported` error comment is unused. Actual Gloas state types use progressive lists/containers and `BeaconState.generate_proof` calls `merkle_proof::progressive_container_proof` at the selected source; current/next committee and finalized-root proof consumers use it. R08 is independent migration/proof qualification, not a rewrite of absent primitives.
- Current CL code also has Gloas partial-column types/filters. R09 needs real peer/version/recovery acceptance, not a claim that these types do not exist.
- Optional EIPs 12581/12582, competing payload authentication proposals and unproved upstream payment allegations are not unconditional implementation requirements or demonstrated local exploits. Their dependent review/qualification remains explicit.
- The later/private FOCIL progressive-IL parser TODO is outside this declared Glamsterdam target. Expanding release scope to Heze/Bogota requires a separate extension inventory.
- Standalone ParallelProcessor and experimental Frame/AA lanes are not substitutes for tracing the actual canonical ordinary transaction consumer. Do not count an unused helper as a missing live path.

## Current upstream/client boundaries

At the live snapshot, EIPs [12581](https://github.com/ethereum/EIPs/pull/12581)/[12582](https://github.com/ethereum/EIPs/pull/12582), execution-apis [852](https://github.com/ethereum/execution-apis/pull/852)/[855](https://github.com/ethereum/execution-apis/pull/855), and consensus-specs [5770](https://github.com/ethereum/consensus-specs/pull/5770)/[5820](https://github.com/ethereum/consensus-specs/pull/5820)/[5781](https://github.com/ethereum/consensus-specs/pull/5781)/[5818](https://github.com/ethereum/consensus-specs/pull/5818) are open/unmerged. Payment issue [5754](https://github.com/ethereum/consensus-specs/issues/5754) is open; its alleged variants are not independently demonstrated here. These statuses guide R06/R07/R13; they are not permission to silently adopt a proposed model.

Recommended immediate work: **R01 → R02 → R05**, with R03/R04 and independent CL security/proof/network work progressing against pinned inputs. Run the complete R10/R11/R12 campaign against the resulting compatible artifact set, then close R13. Keep named ownership for every open obligation.

The old **14.5–27 engineer-week** estimate predates the now merged implementation work and is not a remaining-effort estimate. This audit does not derive a new calendar promise or percentage from PR counts. Re-estimate after the concrete defect/fixture work and compatible client target are established.
