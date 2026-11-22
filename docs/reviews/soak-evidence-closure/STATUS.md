## Current incident

The original run failed at2026-11-22T03:44:38Z after781 accepted samples /23545.536 observed seconds. CL syncing HTTP exceeded10s, then CL failed graceful30s stop and exited137 withoutOOM; EL exited0. Frozen databases and runtime/logs are retained privately. Lighthouse stall investigation is a separate source branch; this audit does not call the failed run a48h success or reset its record.

# Long-soak audit status

The existing48-hour run remains active at source2c0df12f; its immutable protocol/resources and failures are preserved. At the initial2026-11-22T03:15Z inspection it has run about6hours, not48. Native/Geth commitments agree; samples show noOOM or observation gap so far. Six bounded BLS windows pass their recorded live/offline checks. This review audits complete evidence and end-of-run closure rather than treating passed flags as sufficient.

Confirmed controller acceptance gaps: the operating loop checks elapsed time but does not enforce maximum sample gaps; the signed collector checks adjacent slot bounds but does not require each new proof chain to include the previously finalized anchor; cleanup return/state is not part of the operating passed decision. These are acceptance gaps, not a reproduced native execution divergence. The independent final gate must reject them while retaining the ongoing evidence and original duration.

Before the incident, independent prefix verification rechecked763 complete sample/wire groups,154 MPT proofs and seven fresh unchanged-reference BLS replays, with six cross-window finalized-anchor links and maximum sample gap30.779s. The generated Gloas/Fulu consumers were regenerated from the pinned source archive and match the live reference byte-for-byte.45 Python control tests and the native Go build pass. Four compiled guard-removal mutations are detected. A real claimed-balance mutation is rejected by the separately rebuilt Geth SDK, despite its old checked marker.

A separately retained diagnostic restart serves the complete frozen7872-block execution prefix: native/Geth commitments and parent links agree, with5 transactions and979200 gas. The stopped-observer setup attempt remains failed; no load-capacity result is inferred. The forensic BLS extension now authenticates the entire actual failed-prefix span64059–71906, while operating acceptance remains failed. Actual48h completion and successful daemon cleanup remain unmet gates.

The historical173-member public archive freshly verifies/replays every selected byte,154 account proofs, seven original BLS windows and a signed extension through the final accepted observed head71906. The live gate now selects the separately fixed CL57ccb466 executable3158ae6dc22db03e406b710bd9188ade72d9aad1c0d8cc4a493e4a0d09d6f142. Fresh lifecycle qualification and a newly declared48-hour run remain required; the old6h32m is not carried forward.

The live supervisor also handles the initial zero-sample / not-yet-signed transition without treating it as success: first sample must arrive within60s, first signed segment within1200s. Complete admission still requires the entire duration, signed span and clean exits. The old frozen archive retains its exact old auditor; the current live gate pins the fixed CL.
