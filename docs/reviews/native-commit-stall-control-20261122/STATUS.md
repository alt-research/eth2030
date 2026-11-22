# Native commit-stall fixture

The first original-source control delayed one response after real native execution had already completed. A parallel RPC lookup imported that envelope before the delayed gossip error, and the canonical EL advanced. This demonstrates recovery of that narrow response failure; it does not reproduce an uncommitted native block or explain the original5.5-hour stall.

This fixture will hold the actual native block batch before commit, after full execution/commitment checks and while insertMu remains owned. Peers and unrelated read/upcheck paths remain real. It is explicit test instrumentation, not a production option, and normal node construction stays unchanged. The original long-running source diagnostic remains separate and unmodified.
