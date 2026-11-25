# Fast native import qualification checklist

- [x] Preserve failed03/04 and the active long diagnostic; use a new branch and disposable replay data.
- [x] Read EIP-3675 and the frozen Engine Paris payload-validation/forkchoice requirements.
- [x] Add a bounded real native replay control that checks independent RLP/header identity, full execution, BAL, state roots, FCU and persistent restart.
- [x] Exercise repeated payloads and concurrent historical state reads without slot sleeps or relaxed deadlines.
- [x] Replay the actual empty block77623 and its successors against a private mature-state clone.
- [x] Record measured runtime, exact source/input identity, block count and explicit qualification limits.
- [x] Document the fast PR gate and the final background soak gate separately; never relabel accelerated time as48-hour uptime.
- [x] Run build, meaningful regressions and race checks; submit a separate PR against master.

The native stall root remains unknown until a real waiting stack or a deterministic causal regression establishes it. Passing a replay alone does not resolve that claim.
