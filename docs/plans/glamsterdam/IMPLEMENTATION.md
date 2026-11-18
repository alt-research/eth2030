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
| Baseline | Import the complete plan/evidence and reconcile newer mainline fixes | In progress |
| Canonical header/slot qualification | Verify merged code and add only missing consumer/independent regression coverage | Pending |
| EIP-7954 limits | Enforce current code/initcode size bounds with meaningful execution/admission tests | Pending |
| EIP-7997 factory | Canonical deployment/genesis route, runtime/address/nonce; no fork-time mutation | Pending |
| BAL serving/retention | Qualify merged body wire/storage and fix remaining time/window/consumer gaps | Pending |
| RPC access discovery | Replace placeholder through the actual backend and returned-list gas semantics | Pending |
| Gas/state execution | Selected coherent intrinsic/runtime/state/refund/floor model; upstream choice gate retained | Pending |
| Network and CL lifecycle | Negotiated protocols, sync, actual CL/builder and recovery acceptance | Pending |

Each major change receives its own branch and PR; no unrelated changes are staged. Final task completion requires the acceptance evidence in DEVELOPMENT_PLAN.md. A submitted PR is not a merged or release-qualified feature.
