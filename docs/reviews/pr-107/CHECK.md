# WebSocket connection execution lifetime

- [x] Trace the actual Node Gorilla WebSocket reader and in-memory HTTP dispatch; read the primary single-reader/single-writer contract.
- [ ] Keep reading remote close events while native execution runs, forward connection context and bound queued frames without adding runtime switches.
- [ ] Exercise real TCP WebSocket disconnect, queued frames, oversized frames and independent connections against the actual native EVM; require a restored executed mutation failure.
- [ ] Run affected/race/vet/build checks and publish this separate transport review on top of PR 106.
