# Real native commit-stall recovery control

- [x] Keep original04 root attribution open and preserve the unchanged long diagnostic.
- [x] Assess the single delayed-response control narrowly: native execution already succeeded and a parallel lookup imported the envelope before the delayed gossip error.
- [x] Introduce only an internal instance-local database opener for test construction; normal New continues opening the same FileDB.
- [x] Use an explicit diagnostic test binary to hold one fully checked epoch-boundary block immediately before atomic native commit while retaining the import mutex.
- [x] Preserve all payload/hash/state/BAL/signature checks and original8s Engine timeouts/resources; publish exact armed/blocked/released/aborted evidence.
- [x] Test the opener failure and a bounded real native commit barrier, then verify the ordinary native package/build suites.
- [x] In a separate owned control, release the deliberate hold and assess both CL health and independent canonical execution progress; capture both sides before cleanup.
- [x] Submit independently to master and check merge/CI compatibility. A simulated hold is not the original native holder.
