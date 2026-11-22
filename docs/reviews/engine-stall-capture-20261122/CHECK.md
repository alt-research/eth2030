# Original multi-hour stall evidence checklist

- [x] Keep original failed03/04 immutable; retain separate startup-lock regression and fix.
- [x] Check new30min observation and original6h45m diagnostic without relabeling partial time as completion.
- [x] Reproduce whether buffered Docker log reading can hide the only Engine timeout line while the process remains alive.
- [x] Replace that diagnostic read loop with bounded unbuffered reads, including burst/partial/EOF controls.
- [x] Capture native goroutines and optional CL native thread backtraces before any failed owned diagnostic is stopped.
- [x] Deploy the verified collector on the unchanged original diagnostic; retain script/runtime identities and every failure.
- [x] Recheck the30min operating result, fresh BLS replay, cleanup and sampler continuity after it actually finishes.
- [ ] Submit a separate master PR and check exact-head CI/merge compatibility.

A collector fix is not the original node root-cause fix. Only a captured waiting chain plus a causal regression can establish that claim. The original04 history worker finished before acceptance; do not attribute the original incident to PR135's startup critical section.
