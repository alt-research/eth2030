# Authenticated snap history and BAL backfill

- [x] Observe fresh synchronized snap process missing an in-window original body/BAL served by independent Geth and original native process.
- [x] Read exact selected EIP-7928/8159 and current chain/peer/snap consumers.
- [x] Persist only finalized hash-linked authenticated historical bodies/BALs, never execution-valid markers or pruned state.
- [x] Implement bounded, cancellable, resumable normal-process backfill with existing peers/protocols.
- [x] Verify restart, corruption, interruption, canonical/orphan and in-window availability controls.
- [x] Reproduce the original actual-process request with exact original bytes, plus strict pre-pivot state pruning.
- [x] Run affected/full/race/vet/build and publish a separate master-targeted feature PR.
