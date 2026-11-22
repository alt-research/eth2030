# Long-soak evidence and closure

- [x] Preserve the running source2c0df12f pair and all previous failed attempts.
- [x] Inspect actual wall-clock duration, complete samples/proofs, processes and six captured BLS windows.
- [ ] Independently regenerate the pinned consensus consumer and replay every completed window.
- [ ] Reproduce missing cadence, cross-window anchoring and final-cleanup acceptance guards with meaningful negative controls.
- [ ] Add a fail-closed audit/supervisor without restarting or relabelling the existing run.
- [ ] Verify local tests/build, publish a separate master-targeted PR and reconcile exact heads/CI/merge compatibility.
- [ ] Require actual172800 seconds and complete contiguous finality coverage before final acceptance.
