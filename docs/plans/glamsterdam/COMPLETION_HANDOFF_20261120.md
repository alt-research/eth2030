# Glamsterdam frozen-development completion handoff

The remaining engineering plan is complete for the selected merged LIFO EIPs revision and signed Gloas envelope. All 33 original GL IDs and 13 work packages are reconciled in the [task record](../../reviews/glamsterdam-final-campaign/TASK_RECONCILIATION.json). PR review/merge and the final public protocol/model decision remain separate; the user explicitly reserved the latter. No public-network activation or EL master merge is performed.

## Review delivery

PRs 116–119 are verified merged into master `a7c1ebc5`. The final independent master-targeted reviews are:

| Review | Result |
| --- | --- |
| [121](https://github.com/alt-research/eth2030/pull/121) | Preserve real fenced-storage errors through Engine and canonical forkchoice instead of misclassifying them as peer SYNCING. |
| [122](https://github.com/alt-research/eth2030/pull/122) | Download authenticated original snap historical bodies/BALs with restartable progress while retaining the pre-pivot execution/state boundary. |
| [120](https://github.com/alt-research/eth2030/pull/120) | Final real-process consumers, faults, independent replays, finite support records, operators, complete archives and this reconciliation. |
| [Lodestar 10693](https://github.com/ChainSafe/lodestar/pull/10693) | Separately review the installed libp2p topology race correction against upstream unstable. Local seven regressions, workspace build/type check/lint and actual automatic recoveries pass; full upstream repository CI has not run on that external head. |

The three EL reviews have independent master bases and retain normal merge ancestry. All six ordinary merge-commit orders are checked for conflicts and identical resulting trees. Current-head CI is checked at handoff. Repeated squash/rebase ancestry is outside this qualified merge workflow. Root CHECK/STATUS changes belong only to review 120; defect reviews use their own records.

## Qualified source and process tuple

The actual native executable is published at [eff0b823](https://github.com/alt-research/eth2030/tree/eff0b823fdf3e0e4d6262bc093b44e8a77787067) on `qualification/glamsterdam-runtime-20261120`. A clean isolated Go 1.26.11 rebuild is byte-identical: `78d75b0fb0a1c74cca3ad31322b5ccf9cd6db3b299e4edd711926dd63b5cf2dc`. Later tool/review commits have no native production-source delta. Composed `f407bdd2` passes 219 fresh package suites, eleven affected race suites, full vet and native/tool builds. Later command changes pass three affected race suites; all 26 Python regressions pass.

The [protocol manifest](../../reviews/glamsterdam-final-campaign/PROTOCOL_MANIFEST.json) distinguishes exact specification commits, compiled sources, running executable/module hashes, image labels, original validator clients, dependency/license/lock graphs, genesis, wire formats and phase resource limits. The network uses chain 1337, 384 validators, three-second slots, Fulu epoch zero and Gloas epoch two. Lighthouse BN source d547 and later test-only eth2030 head remain distinct. Lodestar application cf705 and its explicitly hashed libp2p dependency patch remain distinct. These are frozen private development inputs.

## Completed engineering gates

| Gate | Actual acceptance |
| --- | --- |
| G0 | Original promotion, 65,537-block ancestry and changed-path history corrections are merged; new process-found storage and historical-BAL gaps have separate reviews, composed regressions and local builds. |
| G1 | Exact compatible EIPs/consensus/Engine/devp2p/API/client tuple, genesis/forks, resources, source/license/lock inventories and byte-identical native rebuild are retained. Open alternatives are explicitly unadopted. |
| G2 | 73 complete signed lifecycle transitions, actual FULL/EMPTY/withheld/late/duplicate, two independently verified same-height process switches, six exact payments, true matching-header-hash/wrong-execution-state-root rejection, crash/restart, and the complete registry continuation below. |
| G3 | Actual getBlobs-disabled partial acquisition, 2,360 independent KZG verification checks across four two-row process campaigns, authentic ordering/signature negatives, custody 4→8→16 with provider loss/restart persistence, and three automatic gossip recoveries in 6.605/0.082/2.241 seconds. |
| G4 | Original pair-3 exclusive history upgrade 1,470.836 seconds; final original-native restarts 27.489/33.511 seconds; final observer restarts 24.529/46.686 seconds; fresh final-source empty-database full/snap 613.390/76.483 seconds under 3600/600; four independent proofs/getters, interrupted exact BAL/body backfill to original genesis and three actual exact-WAL OS-fault recoveries. |
| G5 | Final 1200-second steady window: 400/400 slots, 419 parent links. Final 180-second pressure: 90 normal rounds, 270 independent account proofs, 14,222 actual HTTP/WebSocket disconnects, 358 invalid-hash submissions, 60/60 slots and 75 links. Both have 512-participant independent finality, offline replay, original latency/capture/runtime-identity budgets and exact public Engine selections. Logical DB work remains distinct from sampled physical resources. |
| G6 engineering | Original-byte manifests/archives, independently replayed consumers, source/runtime/license inventory and upgrade/resync/recovery/rollback instructions are complete. Public protocol/model approval remains reserved. |

## Registry continuation and retained failures

The first ordinary 64-epoch registry cycle replays 112 complete signed transitions: epoch-766 sweep at slot 24512, exact 50-ETH credit on three ELs and index reuse at 24521. Its retired-key driver is retained unqualified: producer restart reapplied CLI fleet target one, then the old restoration order enabled lifecycle before restoring target two and caused an ordinary irreversible exit. A stateful regression and actual disabled-lifecycle endpoint preflight qualify the corrected restoration order.

A distinct continuation preserves the original 7200-second budget and unmodified chain clock/delay. The normal epoch-833 sweep occurs at slot 26656; index 1 is assigned a different public key at 26664. Another 160 complete signed reference transitions and three-EL proofs reproduce the sweep/reuse and exact 50-ETH credit. At actual proposer slot 26789 the retired signature over the untouched current message returns `400 BadSignature`; availability remains `404`; the original new-key signature returns `200`. Positive GET returns identical signed SSZ bytes. Independent reference verifies the old/current signatures, signed current bid and proposer block. All 138 related slots receive independent signed finality with 512 participants and 161 parent links; original and public-archive offline replays are retained. This authentication-only span evaluates no operating budget.

Original missing-slot, pressure-cadence, transport-route, storage, pruned-state and producer-restore failures remain separate. An incorrect-header hash control is not represented as execution-state-root validation. A legacy requested reorg that only extends the chain is not counted as a same-height reorg. The unqualified positive-GET route typo is retained beside the correct plural endpoint and typed SSZ byte check.

## Operators and remaining review actions

The [full result record](../../reviews/glamsterdam-final-campaign/STATUS.md), [original archives](../../reviews/glamsterdam-final-campaign/artifacts/README.md) and [operator runbook](../../operations/glamsterdam-recovery.md) identify exact evidence, replay commands and preparation versus daemon-start timing. The final producer executable/configuration/target one and observer IP/network/alias are verified restored; temporary proxy/network are absent. Ordinary cleanup key 1 is exiting with withdrawable epoch 903; its future sweep is not counted as observed. Only the four new validator-free cold-repetition processes are stopped after acceptance, with databases retained. Original campaign services and the unrelated dirty checkout remain preserved. Earlier producer inode fields were empty because root could not read the UID 999 process; those captures remain unchanged. The final inventory uses the original container UID, requires a readable inode and verifies the actual original producer hash. The earlier window identity comparisons cover container/image/command/resources/start time. Two red controls and all 26 Python tests cover this capture-tool repair.

The remaining actions are review/merge of the separate EL PRs, external-client upstream adoption review and the reserved final public protocol/model decision. The refreshed primary GitHub register still records gas/envelope/trace alternatives as open and unmerged; development completion does not adopt them. Selected finite support remains 31,573 modern native state cases, 1,223 historical cases, 107 files/108 official RPC requests, 31 published signed application transactions/17 getters and the real canonical factory campaign. Unexecuted blockchain fixture formats, ecosystem-wide application coverage, lifetime supply accounting and mainnet capacity are not inferred.
