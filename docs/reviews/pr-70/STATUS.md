# PR 70 tracking cleanup status

Original review head: `86a3322133a00d0ba36928decedc859299c9abc9`. Merge base for preserved records: `38ea9259901c7873f483752ea390962e26563648`. Target master: `98716284b049933888bcd68d64c1957004d759b2`. Shared tracking policy: `42a9ceb86dea2fe4a74f01fd5e2360d6f21af0e7`. The sections below preserve this branch snapshot, including any inherited feature records; their historical validation is not a new result for this cleanup. The exact original diff, including removed text, is in [legacy-tracking.patch](legacy-tracking.patch).

Cleanup validation is pending. Publication must be a fast-forward update of the existing review branch. Draft status and dependency order are unchanged.

## Preserved feature tracking

## Modern receipt bloom restoration (2026-11-18)

Branch `fix/eth-receipt-bloom` repairs a receipt-root interoperability defect: eth/69+ omits bloom on the wire, but the decoder OR-ed the newly constructed receipt's zero bloom instead of computing it from decoded logs. It now uses LogsBloom. A nonzero-log typed receipt roundtrip first failed on baseline and now preserves exact consensus bytes. p2p/ethproto, eth and eiptest suites plus full build passed. The fix is also required by the negotiated protocol and transfer-log feature reviews.

