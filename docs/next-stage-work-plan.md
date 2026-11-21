# ETH2030 next-stage development plan

Updated November 22, 2026 after all submitted EL PRs through 125 merged at `9a094830`. The selected merged LIFO / signed Gloas development target and its 33 original tasks are complete; the [current audit](audits/merged-glamsterdam-20261122/README.md) identifies the following work instead of re-opening old completed acceptance cases.

| Order | Work | Concrete completion criteria |
| --- | --- | --- |
| 1 · Review | Corrective PRs127/128/129/130 | Preserve separately reviewable Builder retirement, startup rollback bounded FileDB implementation and restored snap progress. All targetmaster; use ordinary ancestry-preserving merges after126. Verify the composed package tree and exact-head CI before release attribution. |
| 2 · P1 qualification | Scale and soak (GAP-03) | Complete actual172,800 seconds on the final declared runtime. Require non-optimistic CL, progressing finalized execution, original Geth root/hash agreement, independent MPT consumers and fixed resource/restart budgets. Existing synthetic value/WAL and private million-file restart results do not close this duration. Metadata/cardinality and complete root/ancestry authentication still grow. |
| 3 · Upstream | Lodestar10693 (GAP-02) | Maintainer lockfile review and full hosted unit/spec/e2e/browser CI, then upstream merge/dependency integration. Keep the local15-case/process evidence separate. |
| 4 · Reserved review | Public protocol/model/release (GAP-04) | User approves the compatible gas/Engine/consensus/CL/fixture target and any public activation schedule. Source merge does not provide this approval. |
| 5 · Conditional coverage | Corpus extension (GAP-05) | Add concrete adopted fork/vector formats or uncovered consumer cases, retaining exact counts, negative controls and native versus Geth-backed scope. |

GAP-01 and GAP-06 are corrected in source, pending PR merge. Compatibility follows the user's explicit decision: retire unsupported Engine extensions and keep the existing standard CL/EIP-8282 route. The initial audit remains an immutable historical observation. See [delivery/evidence and merge handoff](plans/glamsterdam/REMAINING_CODE_CLOSURE_20261122.md).

Independent work can proceed while upstream/public approvals remain pending. No public release date or whole-fork percentage is inferred from merged PR count.

## Separate later-fork and service sequence

1. Select an independent source with the same seven-field Frame/Bogotá envelope, nonce/gas/receipt/Engine contract. Keyed-nonce experiments require a separately versioned profile. Verify actual state/receipt roots, signed verdicts and restart, not just a same-implementation pair.
2. Integrate Dora's upstream-shaped SSZ decoder into real epoch/receipt/statistics consumers; test competing roots, pruning, event loss, Req/Resp recovery, persisted restart and rebuild before removing retained-view adapters.
3. Qualify ordinary and delegated/sponsored workload services, paymaster ABI and production cutover against explicit source pins.
4. Extend CL skipped-slot/VC/remote-signer/redundant-BN and alternate-peer coverage beyond the profiles already measured.
5. Continue public binary layout/witness/key lifecycle and application-specific proof/native-rollup acceptance independently.

Keep each major function on a separate branch/PR, preserve unrelated dirty work, commit exact edited files and verify actual source/runtime identities. Retain failed attempts separately. Existing recovery/backup instructions remain in [the operator runbook](operations/glamsterdam-recovery.md); a new scale campaign declares its own limits rather than changing an existing measured budget.

[Current roadmap](ROADMAP.md) · [Progress](PROGRESS.md) · [Detailed findings](audits/merged-glamsterdam-20261122/README.md)
