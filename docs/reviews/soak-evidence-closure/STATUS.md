## Current incident

The original run failed at2026-11-22T03:44:38Z after781 accepted samples /23545.536 observed seconds. CL syncing HTTP exceeded10s, then CL failed graceful30s stop and exited137 withoutOOM; EL exited0. Frozen databases and runtime/logs are retained privately. Lighthouse stall investigation is a separate source branch; this audit does not call the failed run a48h success or reset its record.

# Long-soak audit status

The existing48-hour run remains active at source2c0df12f; its immutable protocol/resources and failures are preserved. At the initial2026-11-22T03:15Z inspection it has run about6hours, not48. Native/Geth commitments agree; samples show noOOM or observation gap so far. Six bounded BLS windows pass their recorded live/offline checks. This review audits complete evidence and end-of-run closure rather than treating passed flags as sufficient.

Confirmed controller acceptance gaps: the operating loop checks elapsed time but does not enforce maximum sample gaps; the signed collector checks adjacent slot bounds but does not require each new proof chain to include the previously finalized anchor; cleanup return/state is not part of the operating passed decision. These are acceptance gaps, not a reproduced native execution divergence. The independent final gate must reject them while retaining the ongoing evidence and original duration.

Before the incident, independent prefix verification rechecked763 complete sample/wire groups,154 MPT proofs and seven fresh unchanged-reference BLS replays, with six cross-window finalized-anchor links and maximum sample gap30.779s. The generated Gloas/Fulu consumers were regenerated from the pinned source archive and match the live reference byte-for-byte.45 Python control tests and the native Go build pass. Four compiled guard-removal mutations are detected. A real claimed-balance mutation is rejected by the separately rebuilt Geth SDK, despite its old checked marker.

A separately retained diagnostic restart serves the complete frozen7872-block execution prefix: native/Geth commitments and parent links agree, with5 transactions and979200 gas. The stopped-observer setup attempt remains failed; no load-capacity result is inferred. The forensic BLS extension now authenticates the entire actual failed-prefix span64059–71906, while operating acceptance remains failed. Actual48h completion and successful daemon cleanup remain unmet gates.

The historical173-member public archive freshly verifies/replays every selected byte,154 account proofs, seven original BLS windows and a signed extension through the final accepted observed head71906. The live gate now selects the separately fixed CL57ccb466 executable3158ae6dc22db03e406b710bd9188ade72d9aad1c0d8cc4a493e4a0d09d6f142. Fresh lifecycle qualification and a newly declared48-hour run remain required; the old6h32m is not carried forward.

The live supervisor also handles the initial zero-sample / not-yet-signed transition without treating it as success: first sample must arrive within60s, first signed segment within1200s. Complete admission still requires the entire duration, signed span and clean exits. The old frozen archive retains its exact old auditor; the current live gate pins the fixed CL.

Round04 begins accepted operation at2026-11-22T05:11:01.060551+00:00, earliest48-hour deadline2026-11-24T05:11:01.060551+00:00 (Shanghai 2026-11-24T13:11:01.060551+08:00). Fixed-image smoke preserves2CPUs/4.096GB and10s/30s bounds, passes4096 concurrent reads,45s P2P loss with12.006s recovery and both clean exits.48 Python controls pass, including first-sample/first-signed-window transitions. Operating sampler, BLS collector and strict supervisor run independently from immutable source copies. Current snapshots are incomplete observations; the final gate still requires actual48h, complete signed span and verified cleanup.

The CL scheduling fix is separate fork PR4, targetingeth2030 under the prior explicit user authorization. Its actual release source is57ccb466; later suffixes are tests/docs/CI without production changes. The original blocked dependency cycle remains unobserved; completed duration requalification is still required.

## Source provenance follow-up

The auditor now also requires the exact frozen CL source declaration in the measured profile and both original container identities. Executable hashing remains independently enforced; this closes a provenance inconsistency rather than changing cryptographic acceptance. Wrong/missing source controls pass, along with the full 24 audit/supervisor unit tests. The updated sample and runtime checks were also replayed against the current actual observer prefix. No live controller or immutable deployed source was replaced. The active tuple remains EL 2c0df12f / CL 57ccb466, not the unqualified combined candidate.

## Consolidated progress snapshot

See [RESULTS_20261122.md](RESULTS_20261122.md) for current failed/running/passed boundaries, independently reviewable CL PRs4-11 and remaining exact-source/full-duration gates. The38 Python tests and native build passed; final tuple qualification is still pending.

## Authoritative round04 failure follow-up

Round04 failed at5h30m /658 samples; it is not active and not a48-hour pass. The signed prefix remains independently valid but the unsigned tail and runtime stall need closure. See the final follow-up in RESULTS_20261122.md for the125/129 final fallback failure, two causal lifecycle fixes, actual EF execution/skip counts and the pending native blocked-stack diagnosis. Original failed03/04 evidence is preserved, and no new accepted clock is started.
