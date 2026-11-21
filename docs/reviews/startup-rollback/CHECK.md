# Transactional node startup

- [x] Audit all startup/constructor resource owners and duplicate worker starts.
- [ ] Reproduce occupied P2P/RPC/Engine/WS/metrics and post-worker failure without mocks of the listener.
- [ ] Reserve required listeners before service publication, propagate failures and unwind every owner.
- [ ] Make failed/stopped node instances terminal and prove fresh-instance datadir/port retry and idempotent Stop.
- [ ] Pass functional/race/full-suite/vet/build and cumulative PR merge checks.
