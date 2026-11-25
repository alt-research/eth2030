# Glamsterdam development plan

> Current post-merge assessment (2026-11-19): [remaining-work inventory](POST_MERGE_STATUS_20261119.md). EL reviews 64–87 are merged at `5fbc7c13`; 13 deduplicated work packages remain across correctness, test infrastructure, CL integration and acceptance/release. Earlier statuses, baseline labels and effort estimates below are historical snapshots, not a current missing-code count.

Original planning status: planned, not implemented at its baseline; superseded for current implementation state by the linked post-merge assessment. Baseline and evidence are in the [program index](README.md); requirements are in the [gap matrix](GAP_MATRIX.md). Every GL task started open at that original baseline; the post-merge report reconciles all 33 IDs. This plan authorizes no claim that an upstream proposal has been adopted or that an unrun acceptance gate has passed.

## Delivery principles

- Fix known consensus commitment defects first. Put failing behavioral assertions in place before repairing implementation.
- Follow source → native execution → canonical bytes/hash → wire → production consumer → persistence/recovery. Exercise both Engine backend implementations where they expose the same API.
- Reuse `pkg/eiptest`, its real builder/pool/HTTP probes and `pkg/core/eftest`; add only missing native/vector/consumer capabilities. Do not build a second generic test framework or rewrite unrelated modules.
- Implement the selected fork behavior through existing scheduling, without new feature flags or environment switches. A design experiment uses an isolated checkout and versioned fixtures, not a production runtime toggle. Keep unscheduled public forks unscheduled.
- Preserve unrelated changes. Use task-scoped branches/worktrees when needed; edit, stage and commit exact owned paths. Lighthouse is an independent repository and needs its own commits and validation. Do not modify read-only `refs/` contents.
- Preserve older-fork behavior, receipt economics and required historical data. A header-format correction can invalidate blocks from an experimental devnet; record the incompatibility and use a fresh test chain or a documented migration. Do not silently reinterpret persisted chain data.
- An open upstream choice blocks only dependent final semantics/acceptance. Continue independent repairs, corpus construction and baseline tests.

## Ownership model and effort

Suggested responsibility roles: **EL** for execution/config/header changes; **Consumer** for Engine/RPC/P2P/storage/sync; **CL** for Lighthouse/builder/proofs; **Validation** for independent fixtures, adversarial scenarios and release evidence. These are roles, not assigned people or instructions to launch agents. Each task needs one accountable implementer; reviewers should cross the producer/consumer boundary. With fewer contributors, execute the same dependencies serially.

| Work package | Tasks | Estimated effort |
| --- | --- | --- |
| Baseline, decisions and acceptance inventory | GL-001–003 | 1–1.5 engineer-weeks |
| Canonical header, slot and activation | GL-010–012 | 1.5–2.5 engineer-weeks |
| Coherent gas model and method semantics | GL-020–025 | 4–7 engineer-weeks |
| State, opcode, factory and builder requests | GL-030–034 | 2–4 engineer-weeks |
| BAL serving, RPC and network consumers | GL-040–045 | 3–5 engineer-weeks |
| Actual CL, builder and proof closure | GL-050–055 | 1–3 engineer-weeks, conditional on CL baseline |
| Differential, recovery and release qualification | GL-060–063 | 2–4 engineer-weeks |

The sum is **14.5–27 engineer-weeks**, revised from the first audit to include the newly confirmed header/slot and consumer work. This is a provisional work-package estimate, not measured throughput or a delivery promise. It includes the broader eth/72 and snap/2 target. It excludes waiting for upstream decisions and major unexpected CL/sync architecture defects. Do not add per-task effort to these package estimates: tasks are their breakdown, not additional effort.

For planning, assume two to three experienced contributors and approximately **10–16 calendar weeks after a compatible implementation target is selected**, with work overlapping only where dependencies permit. Re-estimate after G1 and after the gas/authentication decisions. Calendar dates cannot be committed while the upstream target, staffing and measured test capacity are unresolved. A single contributor should plan several months rather than divide this range optimistically.

## Acceptance gates and dependency order

| Gate | Delivery and exit criterion | Dependency |
| --- | --- | --- |
| G0 — reproducible baseline | Compatible spec/fixture candidates, complete scope matrix, observed known failures, and reproducible test/build commands recorded. Pending decisions have explicit owners/status. | GL-001–003 baseline work |
| G1 — canonical blocks | BAL and slot affect canonical header/hash; slot survives build/import/replay/RPC; fork schedules are coherent; independent known-answer vectors pass. | G0, GL-010–012 |
| G2 — gas semantics | Selected execution/state accounting passes native exact-value/boundary tests across validation, EVM, settlement and method semantics. Release choice U-01 is recorded. | G1, GL-020–025 |
| G3 — core behavior | All 18 core EIPs have required native and/or actual CL evidence; factory/precompiles, logs, SELFDESTRUCT and builder requests compose with G2. | G2, GL-030–034 and relevant CL tasks |
| G4 — operational consumers | Real Engine/RPC/negotiated P2P consumers serve canonical data and reject malformed input; promised sync modes recover across restart/reorg. | G1/G2, GL-040–045 |
| G5 — CL/builder recovery | Actual CL/builder validates chosen payload authentication, optimistic/full/empty behavior, onboarding, payments, SSZ proofs and cell exchange. | G1–G4, GL-050–055, relevant upstream decisions |
| G6 — qualified release | Native vectors, application cases, cross-client fork transitions, recovery and measured operating limits pass; no silent exclusions or unresolved release-blocking mismatches. | GL-060–063, G3–G5 |

Gates are cumulative evidence milestones, not a prohibition on earlier independent task work. G3 cannot be called complete using EL-only tests for CL EIPs. A narrower interim release may finish G3 and part of G4, but must list unfinished network protocols and cannot mark G6 for this broader program complete.

## Task backlog

Dependencies name prerequisite artifacts or behavior, not necessarily an entire completed work package. “Selected” means a compatible, recorded specification set, not whichever draft a developer last read.

### Baseline and verification foundations

| ID | Owner / production scope | Work and required evidence | Dependencies |
| --- | --- | --- | --- |
| GL-001 | EL + Validation; config/spec inputs | Create a versioned release input manifest covering EIPs, Engine/devp2p/CL specs, fixtures, client revisions, factory/system runtimes, fork schedule and upstream decisions. Recheck live decisions before implementation; record timestamp and rationale. | Existing audit snapshot |
| GL-002 | Validation; `pkg/eiptest`, `pkg/core/eftest` | Map all 18 core items, SSZ dependencies, network and consumer surfaces to named cases and fixture sources. Add native Amsterdam fixture support where missing; distinguish native/geth/oracle results and unsupported cases. Registry closure must use this scope, not only IFLOW file presence. | GL-001 baseline |
| GL-003 | EL + Validation; core/Engine tests | Convert diagnostic probes into precise failing regressions, repair only the obsolete tagged missing-BAL-hash expectation, and capture baseline failures. Tests must distinguish byte/hash binding, production consumer behavior and intended rejection from unrelated errors. | GL-001/002 baseline |

G0 completion does not require unresolved protocol defects to be fixed; it requires their failures to be reproducible and classified. GL-003 closes only when its intended regressions pass after the corresponding repairs. Until then G0 can record them as expected diagnostic failures, not waived release checks.

### Canonical block and activation repairs

| ID | Owner / production scope | Work and required evidence | Dependencies |
| --- | --- | --- | --- |
| GL-010 | EL; `core/types`, Engine conversion, rawdb/RPC | Encode/decode BAL commitment and native slot in selected canonical header positions. Preserve nil/pre-fork vs present-zero/post-fork semantics, copies, hash caches, database roundtrip and block RPC. Mutating either field in independently constructed headers changes the hash; old-fork known hashes remain unchanged. | GL-001, GL-003 |
| GL-011 | EL + Consumer; builder, processor, replay, RPC | Propagate supplied slot from FCU/attributes into header and every EVM context, including historical calls/replay. Execute SLOTNUM in nonzero-slot blocks produced and independently imported over HTTP. CL rejects mismatched beacon slot; EL preserves rather than invents it from timestamps. | GL-010 |
| GL-012 | EL + CL; config, genesis loader, devnet schedule | Reconcile Amsterdam/Glamsterdan activation into one documented release feature set using existing scheduling. Exclude later-fork 7706/SSZ activation unless separately scheduled. Assert effective rules and header shapes before/at/after the boundary and serialize/load the same schedule. | GL-001, GL-010; feeds all integration tasks |

### Gas accounting as one coherent model

| ID | Owner / production scope | Work and required evidence | Dependencies |
| --- | --- | --- | --- |
| GL-020 | EL; execution/gas design | Write the adopted account/counter model with initialization, charge/refill/rollback/settlement equations, operation order and data ownership. Remove contradictory active helper schedules. Pin normative parameters and U-01 choice; no speculative abstraction for every rejected model. | GL-001, U-01 for final choice |
| GL-021 | EL; pool, processor, builder admission | Implement 2780 intrinsic/runtime split, total/execution caps, calldata floor, flat access-list bytes and 7702 authorization admission. Same transaction is consistently admitted/rejected by pool, builder and importer at exact boundaries; runtime OOG is not mislabeled invalid intrinsic gas. | GL-012, GL-020 |
| GL-022 | EL; VM/call/create/storage gas | Implement selected 8037/8038 access/write/state charges and call-frame bookkeeping, gas forwarding and refund behavior across CALL/CALLCODE/DELEGATECALL/STATICCALL/CREATE/CREATE2, SSTORE and system calls. Cover funded/empty/exhausted reservoir, static failure, revert, exceptional halt and nested refill. | GL-020/021; U-02 if selected |
| GL-023 | EL; execution/state lifecycle | Implement pre-execution new-account/delegation charges and rollback boundaries, existence/collision rules including nonempty storage, successful vs failed creation and code deposit. Assert sender nonce/fees, authority nonce/code, account existence and gas balances for each failure point. | GL-021/022; coordinates GL-033 |
| GL-024 | EL; result, receipts, gas pool, fee update | Reconcile user gas/refund/floor, execution/state block counters, admission reservations and base-fee input. Build and replay mixed blocks with binding/nonbinding floor, refunds and reverted state creation; roots, receipts, balances and next fee agree independently. | GL-021–023 |
| GL-025 | Consumer + EL; production RPC call methods | Implement adopted method-specific transaction vs call gas semantics, historical fork context and correct estimation bounds. Cover call/estimate/createAccessList/simulate/trace paths, supplied gas above/below the execution cap, new-account cases and caught-inner-failure observations. | GL-011, GL-021–024, U-05 |

Scenario construction and current-text baseline measurements can start before U-01 closes. Final consensus implementation acceptance cannot. If the chosen design changes, invalidate affected fixture results and requalify them; do not retain the old results as release evidence.

### State, opcodes, predeploys and requests

| ID | Owner / production scope | Work and required evidence | Dependencies |
| --- | --- | --- | --- |
| GL-030 | EL; VM, pool, creation | Correct 7954 deployed/initcode limits and verify 8024 immediates, stack bounds, code-end and JUMPDEST behavior. Test exact bounds through real transactions, creation, pool and estimates, with old-fork regression. | GL-012, GL-021/022 |
| GL-031 | EL; factory/genesis/precompiles | Use canonical 7997 factory address/runtime/nonzero nonce by specified deployment/genesis route; remove invented fork-time 0x12 injection and restore BLS mappings. Verify CREATE2 address/return bytes, value/failure behavior and relevant precompile vectors. | GL-010/012; final gas cases use GL-022 |
| GL-032 | EL; transfer/log execution | Correct 7708 emissions and ordering across transaction and nested call/create flows; remove extra fee-burn logs. Verify same-address/zero-value exclusions, failed/reverted inner calls, precompiles and receipt roots. | GL-022/023; integrates GL-033 |
| GL-033 | EL; state finalization/BAL | Implement 8246 balance-preserving destruction and clearing rules, including create/destroy in one transaction, transfers after the destruction mark and final empty-account cleanup. Cross-check balances, nonce, code, storage, logs and post-transaction BAL. | GL-023, GL-032; integrates GL-040 |
| GL-034 | EL + CL; system contracts/requests/Engine | Implement 8282 canonical runtimes, request types 0x03/0x04, fees, queue/drain limits and end-of-block extraction. Pin 7685/Engine list/hash validation; independent receiver and actual CL read back registrations/exits and reject altered requests. Retain 6110/7002/7251 regressions. | GL-001, GL-012, GL-023/024; integrates GL-050 |

### Production consumers, networking and storage

| ID | Owner / production scope | Work and required evidence | Dependencies |
| --- | --- | --- | --- |
| GL-040 | Consumer; BAL/rawdb/Engine/node backend | Serve canonical BAL RLP/hex through both historical bodies APIs/backends; persist or deterministically regenerate data for the required epoch/time window, including gaps. Remove JSON-as-BAL path. Verify ByHash/ByRange, restart, retention edges, orphan lookup and CL rehydration against header commitments. Do not collapse bodies to unavailable merely because a local cache missed. | GL-010/011/012; final BAL recording uses GL-023/033/034 |
| GL-041 | Consumer; eth handler/P2P/sync consumers | Negotiate eth/68/70/71 with correct status, ranges and versioned receipt schemas. Correlate request IDs to hashes/headers; validate BAL hashes and receipt commitments in production consumers, apply limits and distinguish version-specific unavailable markers. Tampered/truncated/reordered replies cannot be silently accepted. | GL-010, GL-040 |
| GL-042 | Consumer; eth/72/blobpool/DAS | Add selected 8070 cell masks and GetCells/Cells encoding plus provider/sampler validation/lifecycle. Reconstruct sufficient cells, reject bad proofs and prevent payload building before required data is available. Test negotiated 71/72 peers and incomplete peers. | GL-041, selected networking spec |
| GL-043 | Consumer; snap protocol/sync/rawdb | Add snap/2 BAL-based pivot advancement with state-root validation and bounded correlated serving. Verify cold state sync, missed pivot windows, unavailable data, invalid replies, pivot reorg and restart. Resolve any baseline snap/1 stubs required by this real flow; do not claim sync completion from a helper roundtrip. | GL-040/041, native root correctness |
| GL-044 | Consumer; access-list RPC and VM access tracking | Replace empty placeholder with actual access discovery, correct exclusions and gasUsed for the returned list. Exercise actual HTTP/node backend, historical blocks and repeated execution convergence; use selected merged API exact-match fixtures. | GL-021/025 |
| GL-045 | Consumer; tracing/hooks | Expose selected regular/state/refund/reservoir fields with correct root/nested presence and rollback semantics. Reconcile trace totals with receipts/block counters; no double-counting and no interpretation of nested gasUsed as pure execution by default. | GL-022/024/025, U-05 |

Avoid a broad backend refactor: unify canonical encoding or persistence helpers only where the demonstrated divergent implementations require it. Each consumer task must name which actual node path it validates; a simulated backend cannot stand in for the production adapter.

### Actual consensus client and builder

| ID | Owner / repository scope | Work and required evidence | Dependencies |
| --- | --- | --- | --- |
| GL-050 | CL; `../lighthouse`, builder, native helper claims | Audit/apply selected first-fork availability, bid/message domains, slashed proposer selection, churn and independent builder/validator routing. Verify pending deposits shared keys/signatures, branch/fork consistency and 8282 integration. Keep any retained native helper claims aligned or explicitly qualified. | GL-001/012, GL-034, U-04 |
| GL-051 | CL + EL; gossip/envelope/fork choice | Implement the selected payload-authentication alternative, canonical payload/requests binding and FULL/EMPTY/PENDING weighting. Test claimed-hash/content mismatch, poisoned seen/cache entries and exact rejection; do not put blocking EL execution on gossip solely as an unreviewed workaround. | GL-010/011, GL-050, U-03 |
| GL-052 | CL + Consumer; optimistic import/Engine | Implement and test per-payload optimistic state, latestValidHash handling, FULL descendant invalidation while preserving eligible EMPTY variants, missing/late envelopes, restart and reorg. Cover unknown/null/other-branch valid hashes and FCU for EMPTY heads. | GL-051, GL-040; U-04 selected sync spec |
| GL-053 | CL; payment/withdrawal lifecycle | Test and resolve settlement-once invariants across epoch-window eviction, late FULL children, no quorum, proposer slashing, exit and insufficient balances. Track U-06; independently reproduce upstream allegations before choosing fixes. Record builder debits, proposer credits, withdrawals and supply effects. | GL-034/050/051, U-06 for disputed semantics |
| GL-054 | CL + Validation; SSZ/proof consumers | Qualify 7688 with 7495/7916 on actual beacon containers/lists/bitlists and proof consumers, stable gindices, runtime limits, size bounds and nondefault fork state migration. Existing helper and future execution-SSZ tests alone do not close this task. | GL-001, GL-050 |
| GL-055 | CL; partial-message networking | Verify adopted 8136 group IDs/version bytes, bitmap/cell exchange and recovery under loss/duplication across real CL peers. Assert Gloas 0x01 handling where required by selected spec and compare with pre-fork behavior. | GL-050, selected CL network spec |

Lighthouse deliverables are committed and tested in that independent repository. An ETH2030 documentation commit cannot mark its implementation complete. Final integration records both repository revisions and actual binary/image identities.

### Independent qualification and release

| ID | Owner / scope | Work and required evidence | Dependencies |
| --- | --- | --- | --- |
| GL-060 | Validation; native vectors/invariant corpus | Run selected official Amsterdam fixtures on native execution; compare independent state/receipt/BAL/header results. Add deployed-idiom, gas/refund and conservation cases with explicit current/old-fork semantics. Use targeted mutations for critical assertions and seed nonzero migrated fields. Report fixture counts, unsupported and failed cases separately. | Incremental from GL-002; final pass requires G2/G3 |
| GL-061 | Validation + CL + EL; real processes | Fresh dedicated multi-client devnet: include native ETH2030 plus an independent compatible EL, actual Lighthouse plus an independent compatible CL, and an external builder. Cross a coordinated fork with nontrivial contracts and blobs. Verify independent proposer/importer results, all feature flags derived from the schedule, invalid payload rejection, missing slots/envelopes and actual CL consumers. Preserve complete relevant Engine/P2P/RPC traces. | G3/G4, GL-050–055, GL-060 |
| GL-062 | Consumer + Validation; recovery/capacity | Cold sync, persisted restart, retention boundaries and reorg replay; measure valid and adversarial payload latency, historical serving cost, memory/disk and queue pressure. Pin workload, hardware and fork schedule. Set pass/fail operating budgets before the final campaign; use schedule deadlines and measured headroom, not an invented constant or “kept producing blocks.” | GL-040–043/052; integrates GL-061 |
| GL-063 | Validation + maintainers; release | Reconcile requirements/tasks to passing evidence, run required Go/build/variant/static checks, record remaining limitations and upstream decisions, verify scoped commits and clean owned paths. Publish the qualified artifact manifest and recovery/upgrade procedure; no unresolved P0 or unsupported promised protocol can be hidden in a percentage. | GL-060–062, G3–G5 |

## Initial implementation batches

| Batch | Work to start | Reviewable result |
| --- | --- | --- |
| A | GL-001/002 and diagnostic GL-003; GL-010/011; GL-012 schedule design | Reproducible failures become canonical block/slot regression coverage and a passing G1 candidate. Fix the stale tagged expectation without weakening BAL validation. |
| B | GL-031, GL-040/041, GL-044 discovery; GL-050 merged-transition audit; gas/application scenario corpus | Concrete settled-rule corrections and consumer paths progress while U-01/U-03 remain pending. GAS-dependent final assertions wait for their model. |
| C | After U-01: GL-020–025, GL-030/032/033/034 | One integrated gas/state/core semantics package. Avoid shipping a partial constant change that leaves pool, replay or fees using old rules. |
| D | After selected CL/API/network inputs: GL-042/043/045 and GL-051–055 | Qualified protocol/sync/trace/CL lifecycle consumers; separate repo commits and exact payload format compatibility recorded. |
| E | GL-060–063 final campaign | Independent acceptance, recovery/capacity evidence and release scope reconciliation. |

Batch labels are review/integration groupings, not single oversized commits. Split each into coherent task commits with passing affected tests; record intermediate failures honestly. A header-format change and a gas-model change each need end-to-end consumers before their milestone closes.

## Validation protocol

Use existing eiptest helpers where they reach the production path: real signed pool admission, BuildAndProcess with independent receiver state, historical calls, reorg/finalize, raw RPC assertions and native differential replay. Extend production-node backend tests where the simulator uses a different adapter. Test actual peer handshakes for negotiated protocol support and actual processes for restart/cold sync.

For a critical negative case, first keep all unrelated bindings valid and demonstrate that the intended condition fails. Then temporarily remove/change that check in an isolated verification patch: the regression must fail. Restore the production implementation before committing. This targets test effectiveness without requiring indiscriminate mutation of every line.

Run checks appropriate to each changed package. Example commands from the repository root:

```sh
# Canonical header/slot batch; broaden to touched producers/consumers.
rtk go -C pkg test -count=1 ./core/types ./core/block ./core/execution ./engine/... ./node/backend ./eiptest

# Network/storage/RPC batch.
rtk go -C pkg test -count=1 ./bal ./core/rawdb ./eth ./p2p/... ./sync/... ./rpc/... ./node/...

# Final integration check; the baseline tagged failure must be resolved.
rtk go -C pkg test -count=1 -tags=integration ./core ./p2p/...

# Final native Go qualification, after all implementation batches.
rtk go -C pkg test -count=1 ./...
rtk go -C pkg vet ./...
rtk go -C pkg build ./...
```

Add targeted race checks for changed concurrent Engine/P2P/cache/queue paths and the repository's required variants. Discover actual fixture and devnet commands from their pinned tools before execution; document exact commands and client support rather than assuming `make eest` proves native acceptance. Do not repeat already passing full suites unless subsequent changes or unresolved failures justify it.

Every task's evidence record must include:

1. Requirement/spec/decision revision and task ID; tested Git revisions and dirty-worktree scope.
2. Reproduction or precise behavior assertion; named affected tests and actual outcomes.
3. Native vs geth vs independent client identity; vector inventories including unrun/unsupported cases.
4. For consumer tasks, complete relevant wire data, canonical commitments and independently observed state/receipt/log/request results.
5. For recovery/performance, hardware, workload, schedule, persistence root, restart/reorg sequence, budgets and observed outcomes.
6. Remaining limitation, regression scope and exact scoped commits. A failure, unknown or missing oracle is not converted into pass.

## Definition of done

A task is complete only after its code and tests are committed, affected builds/checks pass, required consumers and recovery assertions are demonstrated, and GAP_MATRIX/STATUS/CHECK point to the evidence. A milestone is complete only when all required tasks and decision gates for its declared scope pass. Updating documentation or creating tests alone does not close protocol implementation work.

The program is complete only when G6 passes for the declared release target and real ETH2030/Lighthouse/builder artifacts. Upstream application-compatibility clearance and official network activation remain external facts; local engineering completion cannot announce them.
