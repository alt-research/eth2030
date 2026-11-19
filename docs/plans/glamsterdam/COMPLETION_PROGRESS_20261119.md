# Glamsterdam completion program: current engineering progress

This is the active continuation of the [post-merge audit](POST_MERGE_STATUS_20261119.md). The historical audit describes master `5fbc7c133b0e57352ca2a3d6fdd9a63726c42502`; its thirteen work packages remain the acceptance inventory. Counts below are test results, never release-completion percentages.

## Frozen authority and review workflow

The user authorized completion with separate EL feature PRs and direct Lighthouse integration into `eth2030`. The selected gas target remains EIPs `3db38a3725105cb67b8804ac3e147967113bc3ca` (merged LIFO); open alternatives 12581/12582 are not silently adopted. The selected signed-envelope format remains in effect pending an evidence-backed compatible target review. Public Ethereum adoption and mainnet scheduling are outside this implementation authority.

This foundation composes the independently reviewable corrections [PR 88](https://github.com/alt-research/eth2030/pull/88), [PR 89](https://github.com/alt-research/eth2030/pull/89), [PR 90](https://github.com/alt-research/eth2030/pull/90), and [PR 91](https://github.com/alt-research/eth2030/pull/91), retaining their original heads. It provides a common review base for subsequent independent work. New feature trackers belong in per-review directories; shared root trackers are owned by this integration document. No EL PR is merged into master by this task.

## Deduplicated acceptance inventory

| Package | Current result | Remaining acceptance |
| --- | --- | --- |
| R01 historical authorization | Original skipped-nonce warming/order/clear bug corrected in PR 88; actual Node import matches independent gas and full root. | Broader signature/envelope validation belongs to R02/R04. |
| R02 native fixtures and envelopes | All 1,223 selected historical EEST Osaka cases pass after composition. Native signed fixture CLI and five-envelope Amsterdam SDK consumers are implemented locally. New complete EELS tests@v21.0.2 inventory: Osaka 14,665; Amsterdam 16,908. | Resolve every complete-corpus mismatch and strict exception/schema classification, add pinned CI and preserve all receipts. |
| R03 application compatibility | Existing synthetic GAS/reservoir controls remain evidence. | Deploy representative actual application artifacts and qualify transaction sequences and bounded-gas behavior. |
| R04 outer/API/historical conformance | Existing merged HTTP, simulation, tracing and access-list consumers remain evidence. | Complete correctly bound negative payloads and entrypoint/historical/override matrices; investigate newly exposed historical validation cases. |
| R05 CL integration | Original three reviews retained and directly integrated into Lighthouse eth2030. Code source 50c8be1 passes release build/workspace check, 62 state-processing and 12 Heze tests. | Final compatible paired process acceptance is shared with R06/R10. |
| R06 payload lifecycle | First unqualified envelope run was found to return at fork guards and excluded. Explicit Gloas/Fulu fork-pinned repeats now execute their distinct bodies. | Complete authentication/hash/request/cache negatives and FULL/EMPTY/PENDING, late, reorg, invalidation and restart matrix. |
| R07 settlement once | Existing CL payment implementation retained; no local double-payment exploit is asserted. | Qualify actual late/slashed/no-quorum/exit/eviction debit/credit/withdrawal/supply invariants. |
| R08 progressive migration/proofs | Existing progressive types and proof generation retained. | Independent nondefault migration/proof/root and real light-client consumption qualification. |
| R09 columns/custody recovery | Existing eth/72 and CL sparse-column implementations retained. | Real peer version/group/bitmap/cell ordering, loss, duplication, reconnect and custody composition. |
| R10 independent multiclient/builder | Component independent SDK comparisons retained. | Full-process native plus independent EL, Lighthouse plus independent CL, external builder and coordinated fork with contracts/blobs. |
| R11 cold sync/durability | Existing encrypted native recovery scenarios retained. | Long independent cold sync, retention/pruning/reorg/orphans and storage-failure/restart campaigns. |
| R12 operating budgets | No operating headroom inferred from passing execution. | Pin workload/hardware/SLOs and measure CPU/memory/disk/latency/queues, including adversarial pressure and cancellation. |
| R13 target/release/runbooks | Engineering target and user authority recorded; code/review integration is separate from release acceptance. | Freeze the complete compatible tuple, publish qualified artifacts and upgrade/recovery procedures after the campaign gates pass. |

## Findings introduced by the complete fixture campaign

The modern official release is [execution-specs tests@v21.0.2](https://github.com/ethereum/execution-specs/releases/tag/tests%40v21.0.2), archive SHA-256 `e2fa7ea2ea516459f9ab92284b2927892f318f4e1f78ca802cb05a6c3b479ed9`. Its first strict run at source `e3a0ddcd5e62cb55177a5300b2c5ddcdb748e740` recorded Osaka 13,362 passes/1,303 failures and Amsterdam 16,047 passes/861 failures. These are unresolved first-pass diagnostic receipts, not thousands of independently distinct defects. They include adapter enum/schema errors and real shared execution errors.

Confirmed subsequent targets include MUL undercharging, failed-precompile remaining gas, memory rounding/zero-length bounds, EIP-1052 emptiness, and historical transaction validation. Each correction requires independent behavior regressions and complete-corpus reruns. Existing EIP-7702, ModExp, finalization and CLZ fixes remain individually reviewable. This document must be reconciled with final campaign artifacts before a completion or release claim.
