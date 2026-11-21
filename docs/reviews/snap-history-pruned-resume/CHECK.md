# Snap history resume after BAL retention expiry

- [x] Reproduce a completed-prefix resume failure after legal EIP-7928 pruning.
- [x] Keep exact-boundary/in-window missing bytes, corrupt bytes and unfinalized-head expiry rejected.
- [x] Verify FileDB reopen, unchanged hash-linked body/state/finality authority and affected tests/race/vet/build.
- [x] Submit a separate master-targeted PR with exact source and validation.
