# Transactional startup implementation

Source starts at PR126 head43002a5d plus shared task tracking. All configured P2P/RPC/Engine endpoints and enabled WS/metrics endpoints are required. Listener reservation precedes worker/service publication. A failed Start releases all node-owned resources and closes Wait; the failed instance is terminal, consistent with Stop closing its database. Retry uses New on the same stopped datadir/configuration. No consensus fork/validation behavior is changed.
