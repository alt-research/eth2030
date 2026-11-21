# ETH2030 next-stage development plan

Updated November 22, 2026 after all submitted EL PRs through 125 merged at `9a094830`. The selected merged LIFO / signed Gloas development target and its 33 original tasks are complete; the [current audit](audits/merged-glamsterdam-20261122/README.md) identifies the following work instead of re-opening old completed acceptance cases.

| Order | Work | Concrete completion criteria |
| --- | --- | --- |
| 1 · P1 code | Legacy Builder Engine extensions (GAP-01) | Trace retained/unsupported methods to the selected protocol; retire unsupported prototypes or authenticate registration/bid signatures with the correct domain/registry key and finalized-deposit authority. Preserve signature/pubkey through header conversion. Real BLS positives plus zero/mutated/wrong-domain/wrong-key/retired-key rejections; rejected input must not mutate registry/auction. Separate functional PR. |
| 2 · Upstream | Lodestar 10693 (GAP-02) | Maintainer lockfile review and complete hosted unit/spec/e2e/browser CI, then upstream merge and dependency integration. Keep the final local/fork15-case/process evidence independent. |
| 3 · P1 qualification | Scale and soak (GAP-03) | Predeclare larger account/storage/history cardinalities and multi-day workloads. Retain disk/RSS/I/O/latency/restart measurements and independent roots/proofs. Correct index/audit growth only with original state/marker/finality guarantees. Do not relabel current20-minute qualification as multi-day capacity. |
| 4 · Reserved review | Public protocol/model/release (GAP-04) | User approves the compatible gas/Engine/consensus/CL/fixture target and any public activation schedule. Adopt a changed model only through explicit source/spec selection and affected requalification. |
| 5 · Conditional coverage | Corpus extension (GAP-05) | Add concrete newly adopted fork/vector formats or uncovered consumer cases while preserving exact input counts, negative controls and native versus Geth-backed scope. |

Independent work can proceed while upstream/public approvals remain pending. No public release date or whole-fork percentage is inferred from merged PR count.

## Separate later-fork and service sequence

1. Select an independent source with the same seven-field Frame/Bogotá envelope, nonce/gas/receipt/Engine contract. Keyed-nonce experiments require a separately versioned profile. Verify actual state/receipt roots, signed verdicts and restart, not just a same-implementation pair.
2. Integrate Dora's upstream-shaped SSZ decoder into real epoch/receipt/statistics consumers; test competing roots, pruning, event loss, Req/Resp recovery, persisted restart and rebuild before removing retained-view adapters.
3. Qualify ordinary and delegated/sponsored workload services, paymaster ABI and production cutover against explicit source pins.
4. Extend CL skipped-slot/VC/remote-signer/redundant-BN and alternate-peer coverage beyond the profiles already measured.
5. Continue public binary layout/witness/key lifecycle and application-specific proof/native-rollup acceptance independently.

Keep each major function on a separate branch/PR, preserve unrelated dirty work, commit exact edited files and verify actual source/runtime identities. Retain failed attempts separately. Existing recovery/backup instructions remain in [the operator runbook](operations/glamsterdam-recovery.md); a new scale campaign declares its own limits rather than changing an existing measured budget.

[Current roadmap](ROADMAP.md) · [Progress](PROGRESS.md) · [Detailed findings](audits/merged-glamsterdam-20261122/README.md)
