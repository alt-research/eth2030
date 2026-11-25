# Glamsterdam remaining work and next development plan — 2026-11-20

> Historical plan/checkpoint. The remaining frozen-development engineering work is reconciled as complete in the [2026-11-20 final handoff](COMPLETION_HANDOFF_20261120.md), with exact process evidence, PRs and reserved public protocol/model review. Earlier pending rows below retain their original observation date.

## Baseline and scope

Master baseline: `7068cd97394570bb7b459eb88a5bbdc613911db4`. All original EL review heads 64–115 are delivered, including the original CLZ correction and later Engine projection fix. This is source delivery, not completion of all release gates. The post-merge audit found three concrete engineering issues, which the user has now authorized fixing with separate PRs. This plan also records the remaining integration, operating and target-selection work; it does not silently start new live pressure/fault campaigns.

The selected development gas model remains the pinned merged LIFO text, EIPs `3db38a3725105cb67b8804ac3e147967113bc3ca`. EIPs [12581](https://github.com/ethereum/EIPs/pull/12581) and [12582](https://github.com/ethereum/EIPs/pull/12582) were rechecked open/unmerged on 2026-11-20 at heads `2ba9176f` and `19d788c8`. Do not mix their alternatives into the current implementation. Final target/model review remains an explicit release decision.

## Immediate code corrections already in progress

| Work | Current observed state | Remaining acceptance for this correction | Review isolation |
| --- | --- | --- | --- |
| F1: full-sync ancestry horizon | [PR 117](https://github.com/alt-research/eth2030/pull/117) completes the 65,537-block case with disk-spooled hash-linked ancestry and the old memory budgets. Moving-head, cancellation, staging-fault/corruption, full/race/vet/build checks and all nine source-head CI jobs pass. | Composed-source tests pass; review/merge and verify canonical master CI. A component fixture is not a live 65k-block process campaign; preserve strict target authority. | Dedicated master-targeted sync PR. |
| F2: cells promotion snapshot gap | [PR 116](https://github.com/alt-research/eth2030/pull/116) captures owned authenticated staging cells/proofs before reading the destination pool. Deterministic real-signed/KZG and all earlier overlap/null/order cases pass; local full/race and ten original peer/restart repetitions pass. | All nine published-head CI jobs and composed-source tests pass. Review/merge and verify canonical master CI; preserve c64ac150 and every cryptographic check. | Dedicated master-targeted Engine/relay PR. |
| F3: whole-state history work | [PR 119](https://github.com/alt-research/eth2030/pull/119) implements dirty-path persistence and lazy indexed point reads/proofs with strict read-error propagation. Full/race/vet/build, independent roots/proofs and both ordinary/lazy Safe/Uniswap lanes pass; all nine source-head CI jobs pass. At 8,192 accounts, persistence/cold-read work is 14/7 logical DB reads rather than 11,445. | Review/merge and verify canonical master/deployed-source acceptance. Complete composed-source tests pass, but this is not mainnet throughput or a new process campaign. Preserve atomic root publication and explicit full audits. | Dedicated master-targeted state/RPC PR, separate from sync and cells. |

Completion artifacts for G0 must include exact review heads, original red controls, passing normal regressions, scope/counts and a composed source check. No EL merge is performed by the agent. All feature reviews target master directly and retain original ancestry; use merge commits for the overlapping history. Root tracking belongs to the final integration/plan owner, with per-review CHECK/STATUS records for each correction.

## G0 — Close the three audit findings

Priority: first. Owners: EL sync, state and RPC/Engine.

Engineering status: PRs 116/117/119 have green nine-job source-head CI. Local composed source `b52d3460` passes all 218 full package suites, thirteen focused race suites, vet/native build and conflict-free normal merges. This branch is a qualification composition, not master. Review/merge and canonical-master/deployed-tuple validation remain outstanding. [Exact composed-source evidence](../../reviews/glamsterdam-next-steps/COMPOSITION.json).

- F1 must cross the old cap, retain useful completed download work across moving targets, bound in-memory descriptors and body bytes, close staging on Stop/failure and reject corrupted metadata. Only actually execution-validated blocks may become anchors. An error can retain a valid executed prefix but must never publish an unvalidated nominated head.
- F2 must cover all promotion interleavings, full/staged duplicates, distinct transactions sharing a versioned hash, partial unions, unknown/missing cells and caller ownership of returned bytes. Lock scope must not include expensive KZG computation or another pool lock.
- F3 must make new-root persistence proportional to changed paths and cold balance/storage/proof access proportional to requested paths. Test one-account/one-slot changes across increasing total state sizes; count logical reads/node writes rather than inferring capacity from elapsed time alone. Lazy read errors must reach RPC and execution consumers, never become successful zero state. Existing complete root/code/storage audits remain available for startup/maintenance acceptance.
- Re-run the complete standard suite plus affected race/vet/native builds. Keep native EELS state fixtures, exact official API replies and actual published application workflows. Add independent roots/proofs, same-root historical isolation, snapshot/revert, failed batch/retry and restart controls for the new state path.
- Simulate all three current review heads against master and cumulatively before a final completion claim. A green individual PR does not qualify an untested composition.

Exit: three reviewable PRs with green source-head CI, reproducible behavior/scale evidence and an explicit composed test revision.

## G1 — Freeze the compatible protocol and deployed-source tuple

Priority: parallel preparation while G0 qualifies. Owners: EL/CL integration and release.

Record a versioned matrix of EIPs, execution-apis, devp2p, consensus-specs, EL, Lighthouse, independent CL, Geth, builder, reference consumers, cryptographic backends, genesis/fork schedule and resources. Retain the selected envelope/authentication/RPC formats and explicitly list unadopted alternatives. A repository branch SHA, compiled binary SHA and container image label are different evidence; check the actual process binaries.

Lighthouse authorized `eth2030` source was verified through `42f6fc26e`. The paired transport fix [rust-libp2p 579](https://github.com/sigp/rust-libp2p/pull/579) is open/unmerged; [Lodestar 10678](https://github.com/ChainSafe/lodestar/pull/10678) is closed/unmerged at the audit snapshot. Decide explicitly whether the qualified release ships frozen forks/patches or waits for compatible upstream adoption. These labels alone do not establish whether equivalent upstream code exists by another route.

Exit: a reproducible build/deployment manifest with exact source and actual executable identities, a compatible target decision, and named unsupported/experimental boundaries. Do not relabel a test-only source update as a deployed runtime update.

## G2 — Complete process-level payload lifecycle and settlement coverage

Depends on G0/G1. Owners: CL, Engine and independent validation.

Existing explicit-fork component checks, signed registry lifecycle tests, real external builder wins/reveals, actual 384-validator SSZ/BLS/KZG consumers and independently authenticated payment 9600→9601 are retained. One exact 0.101-ETH debit/credit is not a lifetime payment or whole-chain supply audit.

Complete the declared FULL/EMPTY/PENDING, missing/late payload, duplicate delivery, reorg, invalidation, replaced-key/exit/reuse and restart combinations in actual independent processes. Exercise settlement across canonical replacement and recovery. For every case retain source tuple, signed inputs, head/finality choices, full header/state/receipt commitments, builder/validator effects and independent consumer results.

Exit: no premature payload availability, no finality rollback, no duplicated debit/credit or settlement on an orphan, exact canonical hash/root agreement, and reproducible positive/negative/lifecycle records. Split major CL lifecycle and EL recovery defects into their own reviews if discovered.

## G3 — Finish the actual partial-column peer/custody campaign

Depends on F2 and G1; can overlap G2. Owners: CL networking and validation.

Existing real two-row blob observations across three CLs, native encrypted TCP loss/reconnect tests, authentic SSZ bitmap/row negatives and independent KZG remain useful. They do not complete the missing full-process acquisition observation.

Use a declared test setup that prevents getBlobs from satisfying the target acquisition, then observe actual partial peer delivery/reconstruction. Cover custody expansion, multiple providers, one provider loss, group/version mismatch, missing/duplicated/reordered cells, multi-row ordering, reconnect and malformed authentic-length data. Preserve protocol-defined IGNORE versus REJECT behavior and prevent incomplete local data from becoming executable or fully available.

Exit: the actual process recovers the needed cells through the intended peer path, verifies every commitment/proof and reaches common non-optimistic finality; known-invalid inputs fail without a false availability announcement. Keep all raw wire/data and original failed controls. Do not introduce a production bypass to make the campaign pass.

## G4 — Finish storage, cold sync and recovery process coverage

Depends on F1/F3/G1. Owners: EL storage/sync and process validation.

Retain the existing independently observed fresh snap completion upper bound of 273.973 seconds under 600, fresh full history readback, and local restart 28.459 seconds under 300. These observations apply to their original source tuples and roles. Preserve earlier failed controls.

- Qualify the final source on original validator-bearing pairs, including the previously deferred pair-3 upgrade through the exclusive verified history procedure.
- Exercise full and snap restart, interrupted indexing/download, changing finalized targets and source loss with authenticated header/state/receipt readback.
- Complete BAL/history retention and readback, pruning, orphan/nonfinality/reorg recovery and exact original-byte availability requirements.
- Inject actual process disk-full/short-write/I/O failure/crash boundaries, including committed WAL before file materialization. Require write fencing, recoverable committed intent and exact authenticated state/head restoration.
- Keep resource failures, absent history, corrupt committed data and invalid network data distinguishable. Do not turn unreadable committed state into zero values or a successful fallback.

Exit: normal next-block execution, required retained historical consumers and independently finalized hashes agree after every accepted recovery. Re-declare budgets before any new campaign; never raise them after a failing run.

## G5 — Qualify the final operating and scale budgets

Depends on G0–G4. Owners: performance and independent validation.

Start with fixed small state changes while growing total account/storage size, then actual transaction/gas/blob workloads and valid/adversarial request pressure. Measure CPU, memory, disk/temporary staging, latency, allocation, cancellation, network bytes and block/slot outcomes. Keep logical DB work counts distinct from physical I/O and single observations distinct from repeated distributions.

The earlier declared 20-minute window failed: 129/400 missing slots (32.25% > 10%). Manual pubsub recovery and merged PRs do not make that window a pass. Run a new predeclared steady window on the final tuple, independently authenticate finalized ancestry, then run pressure/capacity/fault scenarios. Include both full/cold read paths and warm-cache controls so caching cannot conceal an algorithmic scaling problem.

Exit: all predeclared operating budgets pass on the recorded final tuple, without disabled validation, relaxed expected output or silently changed resources. Retain every failed run separately. No mainnet capacity is inferred from an 8,192-account micro-measurement.

## G6 — Release readiness and operator artifacts

Depends on every preceding gate. Owners: release and EL/CL maintainers.

Publish exact supported protocol/fork/capability versions, reproducible artifacts and source/license inventory, upgrade/resync/recovery guidance, operational budgets, remaining unsupported modes and rollback procedures. Reconcile the R01–R13 inventory against final process evidence. Obtain the final compatible-target/model decision and record it separately from development authorization.

Exit: the source, binaries, runtime tuple, independent consumers and operator procedures refer to the same qualified release. Only then close release acceptance. A feature being merged, a helper existing or a unit test passing is not a substitute for the missing consumer/process evidence.

## Execution order and review boundaries

1. Complete the authorized G0 correction PRs and this plan; keep their published heads and CI state explicit.
2. Freeze G1 and prepare the exact G2/G3/G4 cases without changing consensus authority.
3. Run lifecycle/settlement, peer/custody and durability campaigns; publish any newly confirmed defects separately.
4. Freeze the final tuple and execute G5 with original failures retained.
5. Reconcile artifacts and make the explicit G6 release decision.

This sequence is gate-based rather than an unsupported date estimate. G2–G4 can run independently only after their source/authority dependencies are frozen. A newly found defect returns the affected gate to open; do not waive an obligation to make the checklist appear closed.
