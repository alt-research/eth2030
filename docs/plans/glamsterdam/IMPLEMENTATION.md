# Glamsterdam implementation tracking

Implementation begins from fetched `origin/master` at `38ea9259`, not the older audit checkout. The original checkout and unrelated dirty submodule/generated directories are preserved. The working branch is `feat/glamsterdam-baseline` in an isolated worktree. The source plan is imported from `6f60df69`; historical audit outputs remain dated observations, not current-mainline failures.

## Revalidation

- Mainline already includes native BAL and slot header encoding (`db9f384b` and related changes). The old diagnostic now reports different BAL commitments produce different RLP/hash and native Header has SlotNumber.
- Mainline already propagates header slot into execution and persists/serves canonical hex/RLP historical Engine bodies (`e0451092`). The old body-converter-only probe cannot prove omission in the production adapter and is not treated as such.
- Mainline fixed the stale tagged BAL expectation (`49030842`). Tagged regressions are rerun on the actual baseline.
- Remaining observed source gaps include 3,533-epoch BAL retention, old 7954 size limits, the invented 0x12 factory path, old gas constants/refund/reservoir semantics, eth/68-only registration and the access-list placeholder.
- GL-001 remains partial: [baseline inputs](implementation-baseline.json) freeze the audit scope and live-rechecked pending choices. Exact compatible fixtures/client artifacts and open protocol decisions are release inputs, not invented defaults.

## Delivery sequence

| Feature review | Scope | State |
| --- | --- | --- |
| [PR #64](https://github.com/alt-research/eth2030/pull/64) — baseline | Import the complete plan/evidence and reconcile newer mainline fixes | Submitted; release-input closure remains open |
| Canonical header/slot qualification | Verify merged code and add only missing consumer/independent regression coverage | Pending |
| [PR #65](https://github.com/alt-research/eth2030/pull/65) — EIP-7954 | Published code/initcode bounds and transaction/pool boundaries | Submitted; final fork/model qualification remains open |
| [PR #67](https://github.com/alt-research/eth2030/pull/67) — EIP-7997 | Canonical runtime/address/nonce and no fork-time mutation | Submitted; actual generated genesis/control remains open |
| [PR #66](https://github.com/alt-research/eth2030/pull/66) — BAL retention | Published slot-based window, body preservation and conservative fallback | Submitted; complete cold-sync/physical-GC gates remain open |
| [PR #68](https://github.com/alt-research/eth2030/pull/68) — RPC discovery | Actual HTTP/backend discovery, historical/reverted/creation cases and non-mutation | Submitted; final gas/schema qualification remains open |
| Gas/state execution | Selected coherent intrinsic/runtime/state/refund/floor model; upstream choice gate retained | Pending |
| eth/69–71 feature review | Negotiated protocols, canonical wire, correlated historical consumers and encrypted-socket checks | Implemented on separate branch; review publication in progress |
| eth/72, snap/2 and CL lifecycle | Full remaining protocols, actual CL/builder and recovery acceptance | Pending |

Each major change receives its own branch and PR; no unrelated changes are staged. Final task completion requires the acceptance evidence in DEVELOPMENT_PLAN.md. A submitted PR is not a merged or release-qualified feature.

## Verified branch evidence

Feature branches are based independently on mainline `38ea9259`; no feature was merged upstream by this implementation run. Each has its own CHECK/STATUS entry and verification document. Current reviews retain pre-fix failures, meaningful behavioral assertions and scoped builds. The eth/69-71 consumer hash-check mutation fails tamper/reordering cases as intended. A combined local integration run and full release matrix have not yet passed and are not inferred from separate branch tests.

Upstream PRs #12581/#12582 and #5770/#5820 were rechecked live and remain unmerged at this pass. Their concrete heads are frozen in implementation-baseline.json. No open alternative is silently selected as mainnet consensus. Independent work continues while those final semantic choices remain unresolved.
