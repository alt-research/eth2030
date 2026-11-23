# Diagnose the repeated real Engine import stall

- [x] Retain the failed round04 evidence:658 samples,19836.586s, CL syncing timeout; noOOM; both clean exits.
- [x] Fetch and bind reference block77623: zero transactions, gasUsed0; actual native acceptance resumes immediately after SIGTERM.
- [x] Trace existing observability: pprof is imported but not wired through the dedicated metrics mux.
- [x] Expose the read-only standard goroutine profile on the already-enabled metrics listener; require an actual bound-node HTTP regression.
- [x] Archive the cleanly stopped failed04 databases before cloning or reuse.
- [ ] Reproduce on an owned isolated mature-state diagnostic pair, preserving2CPU/4096000000B and all cryptographic checks; collect blocked goroutines before shutdown.
- [ ] Establish the actual blocking dependency, add a causal regression and implement the functional fix separately from diagnosis if needed.
- [ ] Verify builds/appropriate tests and submit a master-targeted PR. Do not call diagnosis or any partial duration a48-hour pass.
