# PR 83: remaining work and completion criteria

Audit date: 2026-11-19. Audited implementation head: `a339283acb74c71377da7a50ce734d3bcaa1ba71`. PR 83 is open and Draft. Master observed at `d31bd720` includes PRs 80–82. This document is a work inventory, not permission to merge or a new gas-model choice. [Machine-readable audit evidence](completion-audit-20261119.json).

## Verdict and evidence boundaries

**Do not label PR 83 fully accepted yet.** The selected ordinary-transaction LIFO implementation is present, its exercised Amsterdam cases match the independent reference, and the audited head has seven successful CI checks. That establishes a qualified development implementation, not exhaustive native fixture, historical compatibility, application, full-client or release acceptance.

This audit proves no new Amsterdam consensus defect in the exercised scope. It does reproduce a historical Osaka authorization mismatch and confirms a missing native Amsterdam fixture adapter. These must remain visible work, rather than be hidden behind green CI. The historical mismatch also exists in the earlier clean PR 83 checkpoint; it is not caused by the recent merge/CI repairs. Its earlier introduction was not established here.

The PR description's claim that RPC/simulation/tracing are still being implemented is stale: #85, #86 and #87 now contain those implementations and have successful CI. They are separate open reviews, not absent code. Their review/integration remains a gate; do not implement them a second time inside PR 83.

## Demonstrated implementation: retain these results

| Area | Evidence in the audited head | Status |
| --- | --- | --- |
| Fixed CPSB 1530, 2^24 execution cap and 2^32-1 total cap | [VM parameters](../../../pkg/core/vm/amsterdam_state_gas.go#L12), [intrinsic validation](../../../pkg/core/gas/amsterdam_intrinsic.go) | Implemented; fixed CPSB is correct for this selected revision, not a missing dynamic-price feature |
| Reservoir-first charging, LIFO refill, child merge, revert/exception and authorization baseline | [Frame accounting](../../../pkg/core/vm/amsterdam_state_gas.go), its existing boundary tests | Implemented and exercised |
| State-independent validity, authorization/account charges, two-dimensional reservation and settlement | [Ordinary transition](../../../pkg/core/execution/amsterdam_message.go#L97) | Implemented and exercised |
| Gross execution/net state header versus paid receipt gas | [Settlement](../../../pkg/core/execution/amsterdam_message.go#L219), [block total](../../../pkg/core/types/receipt.go#L101) | Implemented; existing actual HTTP/import/restart test rerun passes |
| Producer/pool/import and system margins | [Node integration](../../../pkg/node/amsterdam_state_gas_integration_test.go), [actual system margin](../../../pkg/core/execution/amsterdam_system_gas_test.go) | Selected real consumers exercised; broad campaigns remain below |
| Fresh signed independent comparison | Audit checkpoint `23d127c`, source-identical to the PR head; unmodified Geth v1.17.10 with explicit fixed CPSB | 42/42 Amsterdam cases match roots, accounts, logs and gas dimensions |

The 42 signed Amsterdam cases comprise 32 type-2 and 10 type-4 envelopes. A type-2 transaction with an access-list field is not independent coverage of a type-1 signed envelope. The corpus explicitly excludes pool and outer block validation. Those consumers have separate native tests, not 42 independent full-block comparisons.

## Open tasks

### A83-01 — Historical authorization warming mismatch (P1, confirmed compatibility defect)

- [ ] Repair/triage the pre-Amsterdam authorization path with a separately reviewable regression; retain this obligation even if ownership is moved to a corrective PR.

**Evidence:** `authorization_bad_nonce_warms`, Osaka control, is valid on both engines. Native pays **48605 gas**, independent Geth pays **46105 gas**. State roots and sender/miner balances differ. The same input and result exist in clean checkpoint `2cde17ad`. This seventh Osaka difference is **not** one of the six declared-gas-cap controls.

[EIP-7702](https://github.com/ethereum/EIPs/blob/3db38a3725105cb67b8804ac3e147967113bc3ca/EIPS/eip-7702.md) warms the recovered authority before checking its code/nonce, even when a later check skips the tuple. The legacy [authorization helper](../../../pkg/core/eips/eip7702.go#L61) checks nonce and returns before any warming; the [legacy caller](../../../pkg/core/execution/processor.go#L1380) uses that helper. The Amsterdam helper already warms before its checks. Preserve that working lane.

**Done when:** the exact committed signed input matches independent gas 46105, root, balances, logs and nonce/code at Osaka; add wrong/max-nonce, disallowed-code, zero-target clearing, invalid-signature/chain-ID and duplicate-authority controls with correct step ordering. Exercise an actual valid native block/Engine consumer and confirm all 42 Amsterdam cases stay green. Record baseline provenance instead of attributing an unproved introduction to this PR.

### A83-02 — Native official Amsterdam fixture path (P1, missing test infrastructure)

- [ ] Extend the native fixture adapter and add a pinned native Amsterdam corpus to required validation.

**Evidence:** [native ForkConfig/SupportedForks](../../../pkg/core/eftest/state_test_runner.go#L395) stops at Prague; Osaka/Amsterdam are unsupported. CI's `Geth-backed EF state tests` calls [gethcore.ApplyMessage](../../../pkg/core/eftest/geth_runner.go#L247). Its success is SDK fixture evidence, not execution of official Amsterdam fixtures by the native VM.

**Done when:** selected Osaka/Amsterdam fixtures run through native configuration/transition/VM, with coherent predecessors, fixed target parameters and correct expected-exception handling. Pin the fixture revision and expected case counts; missing or unsupported required fixtures fail visibly. Compare roots/logs/gas against authoritative results and an independent implementation. Keep the SDK-backed job as a distinct check.

### A83-03 — Expand independent envelope and boundary qualification (P1, missing coverage)

- [ ] Extend the signed reference corpus beyond type 2/4 and the exercised combinations.

**Done when:** independently compare valid/invalid legacy, type-1 and blob envelopes as well as existing dynamic/set-code cases; cold/warm and prefunded creation, CREATE/CREATE2 collision/OOG/depth, delegated call families, repeated/invalid authorizations, nested refill/revert and floor/refund interactions. Include exact execution/total cap boundaries and overflow inputs. Freeze raw signed bytes, pre-state, both configurations and all outputs. Existing native legacy/blob tests count as evidence already present, but do not replace the missing independent cross-product. Keep negative cases attributable to their intended guard.

### A83-04 — Close outer-validation control boundaries (P1, consumer qualification)

- [ ] Trace all six above-Osaka-cap direct-transition controls through actual pool, construction and import entrypoints, including the standalone memory Engine backend.

**Evidence:** the fresh run has 35/42 matching Osaka controls. Six validity differences use declared gas above the Osaka cap; [BlockValidator.ValidateBody](../../../pkg/core/block/validator.go#L174) has an outer fork-aware cap guard. The comparison command directly invokes ApplyTransaction and does not call it. Do not count these as six proven Amsterdam bugs or simply erase them as universally safe. The memory request path directly uses ProcessWithBAL and needs explicit entrypoint evidence. An audit probe using the normal producer excluded the oversized transaction and therefore did **not** reach receiver validation; it proves no acceptance bug.

**Done when:** supply correctly encoded/rehashed invalid payloads that actually reach each intended receiver; exact-cap payloads pass and cap+1 payloads fail at the gas guard with head/state/config unchanged. Verify before/at Amsterdam and both Node/memory backends. Distinguish caller validation from transition semantics and retain explicit exemptions for uncapped read-only call modes; do not apply a blanket cap that breaks #85/#86 policy.

### A83-05 — Application compatibility and gas-dependent behavior (P1, unrun acceptance campaign)

- [ ] Qualify real application patterns against the explicitly selected LIFO/reservoir behavior and review their implications.

**Done when:** run representative wallet/router/delegated-account/failure-handler patterns and GAS-dependent caller/callee controls, including low forwarded execution gas with a funded reservoir, allocation/clear/refill across frames and exceptional/reverted children. Check success/failure, durable effects, fees and limits independently. Under the selected model, a cold new slot can consume roughly 12100 execution gas while its 97920 state gas is reservoir-funded; do not assert legacy execution-gas footprints or monotonic GAS as if those were already requirements. Document actual compatibility and the release decision. Adopting a different model is conditional A83-07 work, not a hidden test waiver.

### A83-06 — Integrate already implemented method consumers (P1, integration/review)

- [ ] Review and compose #85 → #86 → #87 with the exact accepted gas head; refresh Draft/PR-description claims about unfinished method implementation.

**Evidence:** all three remain separate open PRs at audit. They implement method-specific call/estimate/access-list execution, stateful simulation and real nested/dimensional tracing. Their code and CI exist. PR 83 itself still has the older method routes.

**Done when:** the chosen combined revision passes actual HTTP call/estimate/access-list/simulate/trace, historical fork/prefix replay, override isolation and gas/receipt/header reconciliation. Preserve existing API-proposal boundaries. If PR 83 is merged first as a development-only dependency, explicitly retain this activation/release gate and its ownership; do not reclassify the other PRs as missing implementation.

### A83-07 — Final protocol and merge-readiness decision (P0 review gate, not an automatic rewrite)

- [ ] Complete the independent gas review, state the accepted revision and application implications, and update Draft/description only after an explicit readiness decision.

The standing user decision is to implement the currently merged LIFO model and retain final design review. At audit, upstream master `b055feec` has identical 2780/8037/8038 text to selected pin `3db38a3`; there is no observed drift in those files to repair. [12581](https://github.com/ethereum/EIPs/pull/12581) and [12582](https://github.com/ethereum/EIPs/pull/12582) are still open/unmerged. Implementing them now is **not** an unconditional missing task. If adopted later, isolate the model change in a feature PR and regenerate all normative/reference/application evidence. No final public-network activation timestamp is invented here. PR 83 has no submitted review at the audit snapshot; green CI does not resolve that review decision.

### A83-08 — Independent full-client lifecycle and recovery (P2 release gate)

- [ ] Run the selected gas configuration through independent full EL/CL/builder processes and operational recovery.

**Done when:** exercise activation boundaries, mixed execution/state-heavy blocks, cold sync, restart, forkchoice/reorg and rejected payloads; retain exact client/configuration revisions, real wire requests/responses, roots, fees and recovery readback. The existing two-native-Node HTTP/reopen test is valuable, but it is not an independent full-client network or power-loss qualification.

### A83-09 — Operating-capacity measurements (P2 release gate)

- [ ] Measure latency, CPU, peak memory, disk/state growth, queue/admission behavior and cancellation under declared target block budgets and adversarial workloads.

**Done when:** publish reproducible budgets and measured acceptance thresholds for code/state-heavy and mixed blocks, repeated warm/cold accesses, nested reservoir workloads and pending/deferred transactions. Include long-running storage/recovery behavior. Do not call the reference CPSB derivation a measured ETH2030 operating-capacity result or silently make CPSB depend on the current block gas limit.

## Scope controls and completion procedure

Keep A83-01 (observed mismatch), A83-02 (infrastructure), A83-03/04/05 (qualification), A83-06 (implemented elsewhere), A83-07 (review) and A83-08/09 (release) distinct. A development-only merge decision may define a narrower scope, but all remaining client/release obligations must keep named ownership and acceptance evidence.

Experimental Frame/AA transaction lanes and the unused standalone ParallelProcessor are not proof that the audited canonical ordinary serial Node/import path is missing. Qualify such extension/optimization work separately if claimed. The audit traced Node blockchain and memory Engine request processing to StateProcessor.ProcessWithBAL; neither calls ProcessParallel. Similarly, ordinary receipt RLP intentionally omits derived gas dimensions; no broken required consumer was demonstrated solely by that source observation.

Reproduce the fresh comparison from `pkg/`:

```sh
GOTOOLCHAIN=go1.26.11 go run ./cmd/glamsterdam-compare > /tmp/pr83-current-corpus.json
```

Inspect `differences` per fork; the command's zero exit status does not mean all controls match. The audit also reran focused VM/intrinsic/execution/system/actual Node tests and `go build ./...`; those passed. The gaspool package matched no tests under the chosen name filter and is not counted as a fresh targeted test pass. Earlier exact-head full/integration/race and seven CI results remain separate evidence.

This change adds only audit/TODO documentation and a compact evidence record. It fixes no business code, promotes no Draft, merges no PR and closes none of the unchecked obligations above.
