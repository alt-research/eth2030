# Fast native import qualification checklist

- [x] Preserve failed03/04 and the active long diagnostic; use a new branch and disposable replay data.
- [x] Read EIP-3675 and the frozen Engine Paris payload-validation/forkchoice requirements.
- [ ] Add a bounded real native replay control that checks independent RLP/header identity, full execution, BAL, state roots, FCU and persistent restart.
- [ ] Exercise repeated payloads and concurrent historical state reads without slot sleeps or relaxed deadlines.
- [ ] Replay the actual empty block77623 and its successors against a private mature-state clone.
- [ ] Record measured runtime, exact source/input identity, block count and explicit qualification limits.
- [ ] Document the fast PR gate and the final background soak gate separately; never relabel accelerated time as48-hour uptime.
- [ ] Run build, meaningful regressions and race checks; submit a separate PR against master.

The native stall root remains unknown until a real waiting stack or a deterministic causal regression establishes it. Passing a replay alone does not resolve that claim.
