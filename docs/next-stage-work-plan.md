# ETH2030 next-stage development plan

Updated November24,2026. The adopted frozen Glamsterdam node scope is complete and merged:33/33 original task IDs plus5/5 confirmed follow-ups. There is no active confirmed implementation TODO in this reviewed scope. [Current ledger and plan](plans/glamsterdam/NEXT_STEPS_20261124.md) · [Progress](PROGRESS.md).

48h/larger-scale acceptance and upstream Lodestar approval are deferred by the user and excluded from development progress. Their absence is not converted to acceptance. Original04 exact historical attribution remains unknown. Public activation is a separate network rollout decision, using already implemented timestamp/genesis/override and persisted compatibility paths. No new public fork parameters or future formats are adopted here.

Continue on a concrete newly observed defect or a specifically adopted next scope. Research/later-fork service work below has its own contracts and does not reopen the completed Glamsterdam denominator.

## Separate later-fork and service sequence

1. Select an independent source with the same seven-field Frame/Bogotá envelope, nonce/gas/receipt/Engine contract. Keyed-nonce experiments require a separately versioned profile. Verify actual state/receipt roots, signed verdicts and restart, not just a same-implementation pair.
2. Integrate Dora's upstream-shaped SSZ decoder into real epoch/receipt/statistics consumers; test competing roots, pruning, event loss, Req/Resp recovery, persisted restart and rebuild before removing retained-view adapters.
3. Qualify ordinary and delegated/sponsored workload services, paymaster ABI and production cutover against explicit source pins.
4. Extend CL skipped-slot/VC/remote-signer/redundant-BN and alternate-peer coverage beyond the profiles already measured.
5. Continue public binary layout/witness/key lifecycle and application-specific proof/native-rollup acceptance independently.

Keep each major function on a separate branch/PR, preserve unrelated dirty work, commit exact edited files and verify actual source/runtime identities. Retain failed attempts separately. Existing recovery/backup instructions remain in [the operator runbook](operations/glamsterdam-recovery.md); a new scale campaign declares its own limits rather than changing an existing measured budget.

[Current roadmap](ROADMAP.md) · [Progress](PROGRESS.md) · [Detailed findings](audits/merged-glamsterdam-20261122/README.md)
