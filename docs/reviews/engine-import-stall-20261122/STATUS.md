# Repeated Engine import stall

Round04 failed at2026-11-22T10:42:17Z after658 samples /19836.586 seconds. The CL syncing request exceeded10s. NoOOM occurred and both daemons later exited0. Six BLS windows/five links remain valid for the signed prefix; the full observed span and48-hour duration are not qualified.

Native block77622 was accepted at10:41:20.796. Block77623 timed out at the Engine8s deadline, but acceptance was logged only at10:43:41.793, immediately after native SIGTERM at10:43:41.649. The reference block hash is0x5601d44c4aa53f33ed3d3834692fa12ca74ea989cfa350643952cfded66073d3, with zero transactions and gasUsed0. This excludes an ordinary transaction execution loop as an explanation, but does not establish which lock, I/O operation or background dependency blocked processing. No blocked stack was captured before the automatic shutdown.

The native node imports net/http/pprof for its default mux, but serves metrics using a separate mux with only /metrics and /debug/vars. Add the standard goroutine profile to that existing optional listener so a diagnostic controller can capture the actual wait before stopping a failed owned experiment. This is observability, not the functional root-cause fix. Original03/04 artifacts and the original validator/producer fleet remain unchanged.
