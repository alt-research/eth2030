# Transactional node startup

- [x] Audit all startup/constructor resource owners and duplicate worker starts.
- [x] Reproduce occupied P2P/RPC/Engine/WS/metrics and post-worker failure without mocks of the listener.
- [x] Reserve required listeners before service publication, propagate failures and unwind every owner.
- [x] Make failed/stopped node instances terminal and prove fresh-instance datadir/port retry and idempotent Stop.
- [x] Pass functional/race/full-suite/vet/build and cumulative PR merge checks.
