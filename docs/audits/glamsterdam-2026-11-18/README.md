# Glamsterdam readiness audit — 2026-11-18

Development tracking: [Glamsterdam completion program](../../plans/glamsterdam/README.md). This audit remains the dated evidence snapshot.

> Follow-up correction: the [upstream-discussion audit](../glamsterdam-upstream-2026-11-18/README.md) confirms that the native header hash omits the BAL commitment and that native slot/header propagation is incomplete. It also identifies BAL body/retention and RPC gaps. The original 8-mismatch / 5-foundation classification below is historical; the revised core classification is 10 mismatches / 3 foundations. Current gas refill proposals remain unmerged, so current-text LIFO requirements are versioned rather than a final design commitment.

The current checkout has substantial BAL, Engine API and ePBS infrastructure, but it is **not ready for acceptance against the current Glamsterdam specification**. The historical “~99% complete” label is not a useful release measure. At least eight scheduled EIPs have concrete specification mismatches, two more lack the required EL path, and three CL requirements remain outside completed cross-client acceptance. Networking has additional production wiring gaps.

This is an audit, not an implementation change. No protocol code, reference submodule, running devnet or sibling repository was modified.

## Scope and evidence

- Audited branch: `feat/track2-hegota`, base HEAD `83e3860d02b3b2aebbf985264206fde32bec8e42`.
- Local EIPs reference: `49beda901dcf3747ed4b91f853a333595fb2cbc4`; consensus-specs: `14e6ce5a5334910b0d0575d782e468305ad92373`.
- Current official scope: [EIP-7773](https://eips.ethereum.org/EIPS/eip-7773), read on 2026-11-18. It lists 18 scheduled EIPs, five networking EIPs and two informational EIPs. Local EIP-7773 lists only 7732/7928 as scheduled and has neither 8246 nor 8282 in its scope. Current individual EIP sources were also retrieved from `ethereum/EIPs` master; these are dated observations of evolving specifications, not a permanent release pin.
- Existing unowned changes were preserved: dirty `refs/ethereum-package`, untracked `.playwright-mcp/` and `el_cl_genesis_data/`.
- Sibling Lighthouse was inspected selectively, at `f85fa05b6`; it already has builder deposit/exit processing. Its complete CL conformance, build, test suite and current deployment were not audited here. Native `pkg/consensus` helpers are not a substitute for acceptance of the actual Lighthouse process.
- Source inspection establishes concrete behavior and wiring; only the commands below establish newly observed execution. Earlier devnet observations in STATUS.md were not rerun.

## Scheduled EIP matrix

“Foundation” means useful code and tests exist, not that current-spec release acceptance is complete. “Mismatch” means an identified difference, not an estimate based on filenames. “Unverified” is not a claim of absence.

| EIP | Current assessment | Evidence and remaining acceptance |
| --- | --- | --- |
| 2780 | Mismatch | `core/vm/gas_table.go:104`, `core/execution/processor.go:999`: 4,500 base and state-dependent new-account intrinsic surcharge. Current spec uses 12,000 sender base, 3,000 recipient touch and 6,000 value cost, with state-dependent creation/delegation charges at runtime. Reconcile pool, execution and estimation. |
| 7688 | CL acceptance unverified | Progressive/stable SSZ utilities exist. That does not establish conversion of every evolving CL container/list, stable proof indices and light-client compatibility. Verify the actual Lighthouse fork and vectors. |
| 7708 | Mismatch | `core/execution/processor.go:1500` emits base-fee burn logs; `core/vm/instructions.go:1208` also emits self-burn logs. Current spec excludes fees/burns and requires transfer logs for the specified transaction/CALL/CREATE/CREATE2/SELFDESTRUCT cases. Cover order, rollback and receipts roots. |
| 7732 | Substantial foundation | `consensus/epbs_state.go`, `epbs_transition.go`, `fork_choice_ptc.go` and `eiptest/eip7732_test.go` cover bids, real BLS/PTC, payments and fork choice. Engine/devnet foundations exist. Current external-builder registration/exit, payment, missed payload and reorg acceptance are not closed; 8282 is missing on EL. |
| 7778 | Mismatch | Block/user counters exist, but `core/vm/instructions.go:643` explicitly removes SSTORE refunds. Current spec retains user refunds and excludes them only from block accounting. New execution/state dimensions must also compose correctly with this rule. |
| 7843 | Foundation | SLOTNUM opcode, slot header and Engine fields exist, with Engine tests. Retain vectors proving header hash, supplied slot, boundary behavior and consumer readback across actual clients. |
| 7928 | Substantial foundation | Strict mandatory hash in `core/block/validator.go:299`; Engine re-executes and compares derived BAL. Current-spec vectors must cover all accesses, system calls and failed/reverted operations; networking consumption and recovery need closure. One tagged regression is stale and fails. |
| 7954 | Mismatch | `core/vm/eip7954.go`: deployed/initcode limits 32/64 KiB. Current limits are 64/128 KiB. Update execution, pool admission, estimation and boundary vectors together. |
| 7976 | Foundation with shared gas dependency | Native processor uses uniform 64 gas/calldata byte for the floor, but its base and composition depend on outdated 2780/7981 accounting. Auxiliary `core/gas/glamsterdam_repricing.go` also contains a different floor model. Verify the actual transaction validation and settlement paths. |
| 7981 | Mismatch | `core/execution/processor.go:946` weights zero/nonzero access-list bytes differently and adds tokens to the floor. Current spec charges a flat 64 gas per address/key byte outside the max; it must not depend on the byte values. |
| 7997 | Mismatch | `core/eips/eip7997.go:21` injects a different factory at `0x12`; associated fork precompile sets remove the BLS precompile there. Current spec requires the existing factory at `0x4e59b44847b379578588920cA78FbF26c0B4956C`, canonical code and nonzero nonce, without a fork-boundary client existence check. Resolve state initialization and restore the canonical precompile map. |
| 8024 | Foundation | Immediate decoding, invalid ranges and stack operations are implemented in `core/vm/instructions.go:1267`. Check current vectors for immediates, out-of-code zero behavior, JUMPDEST analysis and fork gating. |
| 8037 | Major mismatch | Fixed CPSB=662, old storage/account byte weights and a 25% reservoir heuristic differ from current CPSB=1530 and cap-based reservoir. Need execution/state counters, LIFO refills, rollback, account/delegation/code creation, system-call budgets and per-dimension block admission/base-fee accounting. This is more than replacing constants. |
| 8038 | Major mismatch | `core/vm/gas_table.go:133` says TBD and uses guessed costs. Current cold account/storage/warm costs are 3000/2100/100; checkout uses 3500/2800/150. Missing current ACCOUNT_WRITE/STORAGE_WRITE composition and refund rules require semantic work. |
| 8045 | CL acceptance unverified; native gap | Native `ComputeProposerIndex` in `consensus/committee_selection.go:194` samples active validators without a slashed-candidate exclusion. Audit actual Lighthouse proposer lookahead/selection and fork vectors separately. Do not infer deployed CL behavior from this helper. |
| 8061 | CL acceptance unverified | Native registry churn helpers do not establish the current split activation/exit/consolidation rules. Verify actual Lighthouse configuration, uncapped exits, activation cap, consolidation quotient and weak-subjectivity behavior. |
| 8246 | Required path not found | No EIP-specific implementation/activation was found. Existing SELFDESTRUCT is not sufficient evidence of the new balance-preserving finalization semantics. Implement and verify same-transaction destruction, subsequent transfers, nonce/code/storage clearing and retained balances. |
| 8282 | Required EL path absent | `engine/requests/execution_requests.go:33` caps request types at `0x02`; no builder deposit/exit predeploy extraction/queue path was found. Current spec adds `0x03`/`0x04`, canonical contracts, fees and drain limits. Must work through finalization, requests commitment, Engine and Lighthouse processing. |

Thus the 18 split into **8 concrete mismatches, 2 missing required EL paths, 3 CL acceptance gaps, and 5 foundations awaiting current-spec acceptance**. This is an evidence classification, not a percentage complete; it does not give eight small fixes the same weight as the 8037 execution model.

FOCIL, frame transactions, future SSZ execution blocks and other later-fork work must not inflate this count. They are not part of the current scheduled 18. PeerDAS foundations are relevant dependencies, but do not themselves prove Glamsterdam completion.

## Production networking gaps

1. **Protocol negotiation:** `eth/handler.go:135` still advertises `eth/68`, length 13, and `runPeer` sends ETH68 status. `node/node.go:617` registers that protocol. Yet the dispatch path uses new receipt schemas on the legacy message codes. The existence of 70/71 constants and direct handler tests does not establish negotiation with real 68/70/71 peers. Implement version-specific negotiation, message ranges and schemas, and test real peer handshakes.
2. **BAL correlation and validation:** `handleBlockAccessLists` calls `validateReceivedBAL`, which only checks an RLP-list prefix for nonempty data. The correct `ValidateBlockAccessListForHeader` helper is called by tests, with no production call site found. Add request-ID to requested-block/header correlation, hash validation, response limits, persistence and sync/reorg consumption. Direct helper tamper tests do not protect this path.
3. **eth/72 (8070):** sparse-pool data structures exist, but the registered protocol does not negotiate 72 or implement the current cell-mask/GetCells/Cells wire flow. Complete providers/samplers, proof validation, reconstruction, admission and builder data availability.
4. **snap/2 (8189):** `p2p/snap/protocol.go:13` remains version 1 with GetTrieNodes/TrieNodes. BAL-based pivot advancement, ordered/limited responses, unavailable marker semantics and reorg recovery remain work. This is a separate networking/sync deliverable from mandatory core state transitions.
5. **8136:** no proposal-specific implementation was found in ETH2030. Cell-delta broadcast is a CL networking acceptance item; check actual Lighthouse support and peer compatibility before classifying it as missing across the pair.

Sources: [8070](https://eips.ethereum.org/EIPS/eip-8070), [8136](https://eips.ethereum.org/EIPS/eip-8136), [8159](https://eips.ethereum.org/EIPS/eip-8159), [8189](https://eips.ethereum.org/EIPS/eip-8189). Network EIPs are listed separately from scheduled core EIPs in EIP-7773; an interoperability target must state which protocol versions it promises.

## Fork activation and why prior devnet success is insufficient

`core/config/chain_config.go` has separate `AmsterdamTime` and `GlamsterdanTime`. BAL/Amsterdam Engine behavior and gas repricing can therefore activate independently. `TestConfig` intentionally enables Amsterdam while leaving gas repricing off; `TestConfigGlamsterdan` enables both.

The saved genesis under `refs/ethereum-package/el_cl_genesis_data/genesis.json` contains `amsterdamTime` but no `glamsterdanTime`. Its presence proves the saved schedule only; live startup arguments were not captured in this audit. No Glamsterdam override was found in the inspected devnet configuration/scripts. Prior successful block production is evidence for the exercised Amsterdam/BAL route, not proof that the current gas package activated. Require a single documented release schedule and an assertion of every effective fork rule at startup and at the transition.

Also resolve bundled experimental behavior: `IsEIP7706` follows `IsGlamsterdan` in the current config. Current Glamsterdam conformance must be tested against its exact scheduled feature set, rather than a mixture of experimental later-fork rules.

## Verification performed

From `pkg/`:

```sh
go build ./...
go test -count=1 ./core ./core/vm ./core/execution ./core/block ./core/config ./engine/... ./bal/... ./eth ./p2p/... ./eiptest ./consensus ./epbs/... ./txpool/...
go test -count=1 -tags=integration ./core
go run ../docs/audits/glamsterdam-2026-11-18/probe.go
```

- Build: passed.
- Focused normal suite: **61 packages passed**, five additional packages have no tests. Full output: [package-tests.txt](package-tests.txt).
- Tagged core suite: **failed**, `TestBAL_HashValidationAcrossFork`, `bal_integration_test.go:536`: `legacy post-Amsterdam payload rejected: invalid block access list hash: post-Amsterdam block missing block_access_list_hash`. The test expects the obsolete permissive rule; the validator correctly enforces mandatory BAL. Repair the expectation, not the strict rule. No protocol fix was attempted during this audit.
- [Behavior probe](probe.go), [observed output](probe-output.txt): zero-value transaction between existing distinct accounts consumed 4,500 gas and generated one fee-burn log; clearing an initially nonzero slot consumed 7,456 gas, with refund counter 0 and identical user/block usage. Both ran successfully through native ApplyMessage. Under current 2780 the first case has 12,000 base plus 3,000 recipient touch; under current 7708 this zero-value fee payment generates no transfer log. Current 7778/8038 preserve a positive capped refund for the second case. These are behavioral divergences even while the normal suite is green.

The probe uses an in-memory StateDB and TestConfigGlamsterdan, not a signed wire transaction or live network. It isolates execution behavior; it does not certify txpool, receipt serialization or peer acceptance. It prints observations and is not a passing conformance test.

Some existing “EIP tests” lock in local constants or monotonic increases (`core/eip8037_test.go`); `core/eip7778_test.go` uses simple transfers without an SSTORE refund. They cannot detect these semantic deviations. The eiptest registry checks IFLOW coverage/file existence rather than completeness of the current EIP-7773 list. The native EF runner's fork table stops at Prague; the separate geth runner supports Amsterdam through geth configuration. Passing geth-backed vectors cannot establish correctness of the native execution engine.

Not performed: full-module tests, race/vet checks, current official Amsterdam fixture campaign, Hive, a fresh devnet, external-builder operation, public-network sync, restart/reorg campaigns, or a Lighthouse rebuild. Remaining defects outside the inspected paths are unknown.

## Remaining work and acceptance order

| Work package | Required result | Rough engineering effort |
| --- | --- | --- |
| 1. Freeze scope and activation | Pin EIP/Engine/CL/vector revisions, resolve separate fork gates and experimental activation, cover all 18 in the acceptance matrix | 0.5–1 engineer-week |
| 2. Rebuild gas semantics | 2780/7778/7976/7981/8037/8038 as one coherent execution model; txpool, estimation, builder, replay, receipts, base fee and system calls agree | 4–7 engineer-weeks |
| 3. Correct state/log behavior and requests | 7708/7954/7997/8246/8282, canonical precompiles/contracts and independent Engine receiver verification | 2–4 engineer-weeks |
| 4. Close production P2P and sync | Negotiated 68/70/71 schemas, correlated verified BAL consumption; eth/72 and snap/2 when included in the release target | 3–5 engineer-weeks |
| 5. CL and builder acceptance | Actual Lighthouse 7688/8045/8061/8136 checks, external-builder lifecycle and adverse ePBS cases; fix discovered incompatibilities | 1–3 engineer-weeks, contingent on CL readiness |
| 6. Differential and release qualification | Current official vectors on the native path, cross-client roots/gas/receipts/BAL, fork transition, cold sync, restart, missed payloads and reorg recovery | 2–4 engineer-weeks |

These are planning estimates inferred from the confirmed gaps, **not measured delivery promises**. The ranges total approximately **12.5–24 engineer-weeks** for the broader networking-compatible release target. Dependencies and integration prevent linear division by team size. A small experienced team might plan roughly **2–4 calendar months**; a single engineer should plan several months. Excluding optional networking scope reduces work; new spec revisions or unresolved CL problems increase it. Audit depth does not justify a precise completion percentage.

First acceptance gate: current-spec gas/state tests and native official vectors must pass. Next: Engine and negotiated P2P consumers must reject tampering and agree on roots. Only then does a fresh multi-client fork-transition devnet count as complete Glamsterdam evidence. Record effective fork flags, client revisions, full request/reply traces, replayed roots and adverse-case outcomes with each run.

Individual current specifications: [2780](https://eips.ethereum.org/EIPS/eip-2780), [7708](https://eips.ethereum.org/EIPS/eip-7708), [7778](https://eips.ethereum.org/EIPS/eip-7778), [7954](https://eips.ethereum.org/EIPS/eip-7954), [7981](https://eips.ethereum.org/EIPS/eip-7981), [7997](https://eips.ethereum.org/EIPS/eip-7997), [8037](https://eips.ethereum.org/EIPS/eip-8037), [8038](https://eips.ethereum.org/EIPS/eip-8038), [8246](https://eips.ethereum.org/EIPS/eip-8246), [8282](https://eips.ethereum.org/EIPS/eip-8282).
