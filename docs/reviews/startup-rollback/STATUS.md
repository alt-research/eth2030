# Transactional startup implementation

Source starts at PR126 head43002a5d plus shared task tracking. All configured P2P/RPC/Engine endpoints and enabled WS/metrics endpoints are required. Listener reservation precedes worker/service publication. A failed Start releases all node-owned resources and closes Wait; the failed instance is terminal, consistent with Stop closing its database. Retry uses New on the same stopped datadir/configuration. No consensus fork/validation behavior is changed.

Functional source 6c8c8b12bc9acf0907ac468068a984c306dbb098 passes 219 fresh package suites,108 official RPC requests, affected race suites, full vet and native build. All six ordinary merge orders with PR126 and the sibling feature produce the same tree without conflicts. Exact current-head CI and PR metadata are checked after publication. The inherited PR126/shared tracking ancestry is explicit; all reviews target master.
