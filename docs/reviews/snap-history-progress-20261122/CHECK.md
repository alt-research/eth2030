# Historical prefix validation and import progress

- [x] Preserve original incidents and the failed mature replay with live waiting stacks.
- [x] Read EIP-3675 atomic CL forkchoice/finalized authority and EIP-7928 history commitment requirements.
- [x] Compile a deterministic barrier regression: slow saved-prefix reads must not serialize unrelated validated-tip execution or FCU.
- [x] Move only the immutable finalized-prefix traversal outside the import critical section; retain all header/body/BAL/canonical checks.
- [x] Preserve cancellation, corruption rejection and absence of pre-pivot execution authority.
- [x] Repeat package/race tests, build the native executable, then rerun the same real mature-state replay.
- [x] Submit a separate PR to master and check integration with the independent replay/profile PR.

Observed failure: replay0676f992, block77623 waits on Blockchain.insertMu while Node.backfillSnapHistory -> SnapHistoryCursor hashes and authenticates the saved prefix. The Engine8s gate failed with noOOM; original failed03/04 attribution remains limited until their live condition is also captured.
