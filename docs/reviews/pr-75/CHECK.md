# PR 75 tracking cleanup

- [x] Preserve original tracking diff and changed sections from `3ee4dae928a86f1d378dc3ced2e31ce4a8e3bfd0`.
- [x] Synchronize shared trackers and merge current master without dropping feature behavior.
- [ ] Verify merged source, local build and affected tests.
- [ ] Publish and verify GitHub conflict state (reported separately after push).

## Preserved feature tracking

## Glamsterdam independent current-text gas corpus (2026-11-18)

- [x] Freeze signed transactions, complete pre-state, fork/config and reference-client version.
- [x] Execute identical inputs through native transaction-level execution and the unmodified Geth Amsterdam transition.
- [x] Record invalid versus included failure, gas dimensions, logs and exact final account/state results.
- [x] Verify pre-Amsterdam controls, archive candidate mismatches and build locally.
- [x] Publish the separate gas-corpus review (PR 75).

