# Preserve Engine storage failure classification

- [x] Reproduce actual post-WAL materialization error followed by incorrect SYNCING in an isolated native process.
- [ ] Preserve storage errors before cached/unknown-head and validated-parent shortcuts.
- [ ] Add actual FileDB fence regressions with healthy unknown-head controls.
- [ ] Rerun actual process reorg/finality/fault/SIGKILL recovery without relaxing assertions.
- [ ] Run affected full/race/vet/build checks and publish a separate master-targeted PR.
