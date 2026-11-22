# Fast native import qualification status

Implemented an always-running16-block native/Engine/SDK proof regression and an explicit mature-snapshot replay test with a resource-bounded driver. The driver was exercised against a fresh immutable failed04 archive clone. The old source failed the8s gate at actual block77623 and retained the waiting stack. With the separate1243adc1 history-lock fix, all256 real blocks77623–77878, forced execution, repeated Engine calls, FCU, concurrent SDK proofs and durable restart passed in295.296s total /232.538s execution. NoOOM; clean exit0.

Fresh independent real-BLS replay of the complete original observed signed span73639–80256 passed7 linked windows in9.277s. It preserves failed04 as a failed operating result. The normal16-block test passed in2.488s; composed race repetitions5/5 passed12.556s. See FAST_ACCEPTANCE.md and RESULTS.json for commands, pins and qualification boundaries.

Original04 history completion occurred before the accepted run, so its multi-hour stall is still unresolved. The unchanged6h45m diagnosis continues independently. No actual48-hour pass is claimed. Full native suite/vet and current PR conflict/CI checks are tracked in the final handoff.
