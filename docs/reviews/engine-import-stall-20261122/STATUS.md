# Repeated Engine import stall

Round04 failed at2026-11-22T10:42:17Z after658 samples /19836.586 seconds. The CL syncing request exceeded10s. NoOOM occurred and both daemons later exited0. Six BLS windows/five links remain valid for the signed prefix; the full observed span and48-hour duration are not qualified.

Native block77622 was accepted at10:41:20.796. Block77623 timed out at the Engine8s deadline, but acceptance was logged only at10:43:41.793, immediately after native SIGTERM at10:43:41.649. The reference block hash is0x5601d44c4aa53f33ed3d3834692fa12ca74ea989cfa350643952cfded66073d3, with zero transactions and gasUsed0. This excludes an ordinary transaction execution loop as an explanation, but does not establish which lock, I/O operation or background dependency blocked processing. No blocked stack was captured before the automatic shutdown.

The native node imports net/http/pprof for its default mux, but serves metrics using a separate mux with only /metrics and /debug/vars. Add the standard goroutine profile to that existing optional listener so a diagnostic controller can capture the actual wait before stopping a failed owned experiment. This is observability, not the functional root-cause fix. Original03/04 artifacts and the original validator/producer fleet remain unchanged.

## Implemented diagnostic surface

Only the standard read-only `/debug/pprof/goroutine` profile is routed through the existing metrics listener. A real bound-node test first failed404, then captured an actual blocked marker goroutine after the route was added; the unrelated CPU-profile endpoint remains404. The startup rollback regression, targeted race control and native build passed. Diagnostic binary source19a3f1bb / SHA256bb3d8b4492a8b55963aabf455f3936a998309ff28e56c69424bc3c3968938bda. This adds no new listener/CLI toggle and does not alter payload/state/hash validation.

Both failed04 database archives are immutable and checksum-bound before cloning: CL670599168B /94f16aa84f5f3a12e4268a81730ca3eaf80e099f2d58de304bcd6b6c8adb698d; EL2324963840B /d247d5378422d23184c03f914226056142a3609a9825368570868e34c16abd21. An earlier host tar encountered unreadable log files and is retained as a failed attempt, not admitted. Original containers/data remain stopped and unchanged. An owned mature-state diagnostic clone is prepared to collect the actual native wait before cleanup; no new48-hour admission is claimed.
