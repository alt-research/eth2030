# PR 69 tracking cleanup status

Original review head: `5ab6ce83f58f02e1531bc017c23a81975cfaa470`. Merge base for preserved records: `38ea9259901c7873f483752ea390962e26563648`. Target master: `98716284b049933888bcd68d64c1957004d759b2`. Shared tracking policy: `42a9ceb86dea2fe4a74f01fd5e2360d6f21af0e7`. The sections below preserve this branch snapshot, including any inherited feature records; their historical validation is not a new result for this cleanup. The exact original diff, including removed text, is in [legacy-tracking.patch](legacy-tracking.patch).

Cleanup validation passed on the merged source tree: four-package core/config, core/vm, core/execution and txpool tests, and `go build ./...` from `pkg/`. Source changes outside the reviewed conflict resolution match Git's automatic three-way merge. Publication targets the existing review branch without force; live GitHub verification follows publication. Draft status and dependency order are unchanged.

## Preserved feature tracking

## Negotiated eth/69-71 and historical consumers (2026-11-18)

Branch `feat/glamsterdam-eth71` registers eth/68/69/70/71 on the production node. Implements seven-field range status without TD for 69+, version-correct bloom-free receipts, nonblocking queued range updates and canonical raw-list BAL wire encoding. Independent golden bytes expose the old double-wrapped BAL response.

Correlated historical fetchers paginate receipts, bound decoding by requested block transaction/gas limits, verify the complete receipt root and rebuild persistent metadata only for validated blocks. BAL replies are matched to peer/request/hash, checked against headers before any restore, and reject tamper/reordering/replay/unsolicited data. Unavailable markers preserve body availability and are not trusted as arbitrary BALs.

Independent Geth RLPx/codec tests negotiate all four versions over real encrypted sockets and check status plus actual receipt/BAL formats. Real-chain eiptests validate restored receipt metadata and tampered BAL preservation; a disabled consumer hash-check mutation fails the intended tests. Affected eth/P2P/chain/node/eiptest suites, targeted race, full build and vet passed. Full independent-client/devnet, eth/72/snap/2 and complete sync/recovery qualification remain separate program gates. [Evidence](../../verification/glamsterdam-eth71-20261118.md).

