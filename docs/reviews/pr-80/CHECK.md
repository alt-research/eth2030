## PRs 80–87 merge and CI repair, dependency order (2026-11-19)

- [x] Inventory all eight open reviews, exact heads, current master and original CI failures; preserve Draft 83.
- [x] Repair and qualify PR 80 against master without importing unrelated unmerged features.
- [ ] Repair PR 81 separately, then reconcile existing dependencies in 82 and 84.
- [ ] Repair Draft 83 and its dependent RPC/simulation/tracing reviews 85–87 individually.
- [ ] Verify exact published heads, no conflicts and all required CI for every review; keep true functional integration boundaries explicit.

## PR 80 current master integration

- [x] Resolve conflicts without weakening fork/configuration or blob validation.
- [x] Fix the observed CI cause and run formatting, full default tests, integration, affected race/vet and build.
- [x] Prepare a non-forced update of the existing branch; require target CI verification before batch completion.

# PR 80 tracking cleanup

- [x] Preserve original tracking diff and changed sections from `12e22aac2f7853092d8270beb644caaf1883f94d`.
- [x] Synchronize shared trackers and merge current master without dropping feature behavior.
- [x] Verify merged source, local build and affected tests.
- [x] Prepare the existing PR branch for fast-forward publication; require live conflict-state verification before reporting completion.

## Preserved feature tracking

## Complete ordinary blob sidecar validation (2026-11-18)

- [x] Read pinned EIP-4844/7594 networking and proof requirements.
- [x] Reproduce invalid ordinary-cell-proof admission through real node HTTP.
- [x] Validate complete sidecar binding/proofs and preserve Frame regression behavior.
- [x] Require full data for pool-based builders and return genuine legacy blob proofs.
- [x] Qualify actual wire/admission/Engine consumers, affected checks/build and regression mutations.
- [x] Publish the isolated blob-validation PR (PR 76).

## Fork-aware blob capacity, fees and configuration (2026-11-18)

- [x] Base on PR 79 configuration authority and compose PR 76 full-data/proof consumers; retain the 21-blob race timeout as evidence until the current wrapper is qualified.

- [x] Read EIPs 7840/7892/7691/7918 and trace actual builder/import/pool/VM/RPC consumers.
- [x] Reproduce hardcoded capacity/fee and ignored configuration defects against independent reference inputs.
- [x] Wire one selected schedule through canonical block admission, fee calculation, payloads and RPC.
- [x] Qualify actual signed blob traffic, fork transitions, invalid blocks and configuration/restart boundaries.
- [x] Run affected/full builds/tests and publish the isolated feature review.

## Persisted fork configuration and restart authority (2026-11-18)

- [x] Trace actual Node startup before history replay and reuse the existing persisted config key.
- [x] Detect incompatible active schedules and immutable parameter changes before state/metadata adoption.
- [x] Qualify future updates, either-side activation, missing/corrupt metadata and failed persistence.
- [x] Verify actual node restart preserves heads/state/configuration on rejection and does not mutate network globals.
- [x] Run affected/full tests, build and race/vet; publish a dedicated PR.

