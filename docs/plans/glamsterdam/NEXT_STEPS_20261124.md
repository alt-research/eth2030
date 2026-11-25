# Adopted Glamsterdam development completion - 2026-11-24

> Current specification follow-up: [closure review](SPEC_CLOSURE_20261124.md) reconciles all 28 current documents and implements five newly confirmed issue groups in independent EL 150/151 and Lighthouse 16 reviews. The original delivery/accounting below is retained at its source; current review/CI/integration state is tracked separately.

## Current completion scope

The adopted frozen LIFO / signed Gloas node-development scope is complete: 33/33 original task IDs, R01-R13, and 5/5 confirmed implementation follow-ups. EL PRs 142-147 are merged into master cfd39f3f; its Go package tree 48afbc34 equals the actually qualified cumulative source 50514bf3. Lighthouse 13-15 are integrated into eth2030 fa16f70a; the qualified CI head 215525de has 27 successful suite jobs plus local-testnet 9 success / 1 conditional checkpoint skip. Source/merge equivalence does not rename old executables or create a new runtime run.

There is no remaining confirmed adopted-scope implementation TODO after this review. This statement covers the adopted contracts and finite qualified consumers. Historical mapped EIP percentages and optional research/later-fork frameworks retain their own denominators and are not promoted to whole-public-fork completion.

## User-directed counting policy

| Item | Treatment in current development progress | Retained truth |
| --- | --- | --- |
| Original tasks and confirmed software defects | Counted; completed and merged | 33/33 original tasks and 5/5 follow-ups. Any future confirmed code defect remains actionable even if found during deferred acceptance. |
| 48h/larger-scale admission (G-03) | Deferred by the user; excluded from progress | No actual172800s or unrestricted-scale pass is claimed. Existing finite results keep their original source and budgets. |
| Lodestar maintainer approval/integration (G-04) | Deferred by the user; excluded from progress | Local code/tests and upstream PRs 10693/10725 are delivered. External approval is not fabricated as a passed/merged check. |
| Original 04 exact attribution (G-02) | Recorded historical unknown, not an unimplemented feature | The original 5.5h waiter was not captured. Confirmed later fsync/publication and directory-capacity defects are repaired independently. Missing original evidence remains unknown. |
| Public-network rollout (G-05) | Separate deployment decision; excluded from node-development checklist | No target public network or new activation parameter is chosen by this task. |
| Additional formats/handlers and later forks (G-06) | Conditional scope, not adopted into this denominator | Inactive/skipped cases remain unexecuted. Concrete new contracts require explicit adoption and their own implementation/qualification. |

Excluding acceptance does not exempt code correctness. The actual ext4 directory limit, durable batch delay, height gauge and RPC bind/address defects remain counted repairs, not waived scale work.

## What public activation means

Activation means deploying a compatible client tuple on a specified network and enabling the fork at that network's agreed epoch/timestamp. It requires the network/genesis identity, EL/CL/specification revisions, agreed parameters and rollout coordination. It is not another missing opcode, signature check, sync implementation or generic "enable everything" switch.

Code already has timestamp dispatch (`core/config/chain_config.go`), genesis loading, explicit fork overrides (`node/genesis_loader.go`, `cmd/eth2030/main.go`) and persisted-config/restart compatibility checks. The current mainnet preset leaves AmsterdamTime/GlamsterdanTime nil. That expresses no activation in that preset, rather than absence of activation support. No public preset is changed here. A draft/meta schedule entry alone must not silently override a pinned network configuration; choose and verify a matching public tuple when public rollout is actually requested.

The upstream [Glamsterdam meta EIP](https://eips.ethereum.org/EIPS/eip-7773#activation) lists activation separately by network. The currently retrieved table has no Mainnet/Hoodi activation values. That is a network schedule decision, not a requirement to invent a mainnet timestamp in this repository. The current development target remains the user-selected frozen LIFO/signed Gloas contract.

## Actual delivered validation

The merged package tree is byte-identical to the cumulative implementation that passed 219 packages, 7 affected race packages and integration checks. Native cmd/eth2030 recovery first probed RPC within 91.038s, caught up within 211.729s and observed 301.423s ready under its separately declared RPC300/catchup900/ready300 profile. 102 slots have real-BLS512-participant independent finality/offline ancestry, actual stopped slot 113651 body/header/state/MPT binding, 4 live + 2 stopped proofs and 60 matching positive gauge reads. All 2707772 actual snapshot values were independently compared across 256 directories. The initial 300s full-catchup failure remains failed; that profile's partial-batch endpoint mismatch is retained.

Historical 30m/6h45m, source-specific failures, missing slots and resource events are unchanged. No fresh 48h, unrestricted capacity or original 04 resolution is inferred.

## Next actions

No new coding work is required by this reviewed adopted-scope ledger. Review/merge the progress-accounting PR, then continue only on a newly observed concrete defect or an explicitly adopted next scope. Public rollout, deferred acceptance, upstream approval and separate Frame/Bogota/Dora/service/research work remain documented without lowering current node-development progress.

[Current accounting](../../reviews/development-scope-20261124/CURRENT_STATE.json) · [source audit](../../reviews/development-scope-20261124/AUDIT.md) · [progress](../../PROGRESS.md) · [roadmap](../../ROADMAP.md).
