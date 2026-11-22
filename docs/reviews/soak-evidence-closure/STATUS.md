# Long-soak audit status

The existing48-hour run remains active at source2c0df12f; its immutable protocol/resources and failures are preserved. At the initial2026-11-22T03:15Z inspection it has run about6hours, not48. Native/Geth commitments agree; samples show noOOM or observation gap so far. Six bounded BLS windows pass their recorded live/offline checks. This review audits complete evidence and end-of-run closure rather than treating passed flags as sufficient.

Confirmed controller acceptance gaps: the operating loop checks elapsed time but does not enforce maximum sample gaps; the signed collector checks adjacent slot bounds but does not require each new proof chain to include the previously finalized anchor; cleanup return/state is not part of the operating passed decision. These are acceptance gaps, not a reproduced native execution divergence. The independent final gate must reject them while retaining the ongoing evidence and original duration.
