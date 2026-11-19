# PR 81 tracking cleanup status

Original review head: `6ce3eceeef539e69d5f2a20428aa7f88e651a661`. Merge base for preserved records: `38ea9259901c7873f483752ea390962e26563648`. Target master: `98716284b049933888bcd68d64c1957004d759b2`. Shared tracking policy: `42a9ceb86dea2fe4a74f01fd5e2360d6f21af0e7`. The sections below preserve this branch snapshot, including any inherited feature records; their historical validation is not a new result for this cleanup. The exact original diff, including removed text, is in [legacy-tracking.patch](legacy-tracking.patch).

Cleanup validation passed on the merged source tree: four-package core/config, core/vm, core/execution and txpool tests, and `go build ./...` from `pkg/`. Source changes outside the reviewed conflict resolution match Git's automatic three-way merge. Publication targets the existing review branch without force; live GitHub verification follows publication. Draft status and dependency order are unchanged.

## Preserved feature tracking

## Unchanged storage account pivot proof repair (2026-11-19)

Reproduced `storage account not verified` over real encrypted Node sync when an unchanged account still has partial storage after a pivot move. The verified storage consumer now obtains an exact account inclusion proof against the fresh pivot before trusting its storage root. Seven encrypted Node scenarios pass, including damaged-proof rejection for the unchanged account; two added compiling mutations are detected. Full Go, tagged Core/EIP, affected race/vet and build pass. This closes the reproduced cache-binding gap without waiving root authentication. Independent-client/resource and final release gates remain open.


## Snap version 2 socket lifecycle follow up (2026-11-19)

Explicitly stop the test-owned P2P servers before Node/database restart and on every scenario exit. Five encrypted Node cases pass again under race, with affected vet and build. This makes the test resource ownership match the manually started component setup; production protocol behavior is unchanged.

## snap version 2 state catch up (2026-11-19)

The production node negotiates snap/2 and snap/1, preserving range/code schemas and rejecting removed trie-node messages on version 2. Raw positional BAL responses distinguish missing 0x80 from available c0; the actual sender and receiver use the custom codec. Header commitments and canonical BAL structure are verified before staged state rebasing.

CL-selected moving pivots reuse partial account/storage progress. Reorg recovery restores old-only entries through exact inclusion/absence proofs and applies ordered new-fork diffs; missing orphan BALs discard partial state and restart bulk download. Final roots and CL authority gate adoption. Five real encrypted Node scenarios and an unmodified Geth SDK packet consumer qualify cursor preservation, rollback/deletion, restart and invalid-proof rejection. Seven compiling guard mutations are detected. Full Go, tagged Core/EIP, affected race/vet and build pass. [Evidence](../../verification/glamsterdam-snap2-20261119.md). Full independent-client cold sync, mainnet resource limits, peer reliability policy, eth/72, final gas and release gates remain open.

## EIP-8282 native builder requests (2026-11-18)

Canonical contracts and mandatory Amsterdam type-3/type-4 extraction now compose with the existing snapshot and Engine commitment path. Explicit owned genesis allocation precedes roots; missing production predeploys invalidate blocks. Actual HTTP and separate real-BLS Lighthouse component consumers qualify request bytes and registration/exit gates. Full Go, tagged Core/EIP, targeted race/vet/build and failing guard mutations pass. See [verification](../../verification/glamsterdam-builder-requests-20261118.md). Final gas, live full-client/devnet and release gates remain open.

## Negotiated eth/69-71 and historical consumers (2026-11-18)

Branch `feat/glamsterdam-eth71` registers eth/68/69/70/71 on the production node. Implements seven-field range status without TD for 69+, version-correct bloom-free receipts, nonblocking queued range updates and canonical raw-list BAL wire encoding. Independent golden bytes expose the old double-wrapped BAL response.

Correlated historical fetchers paginate receipts, bound decoding by requested block transaction/gas limits, verify the complete receipt root and rebuild persistent metadata only for validated blocks. BAL replies are matched to peer/request/hash, checked against headers before any restore, and reject tamper/reordering/replay/unsolicited data. Unavailable markers preserve body availability and are not trusted as arbitrary BALs.

Independent Geth RLPx/codec tests negotiate all four versions over real encrypted sockets and check status plus actual receipt/BAL formats. Real-chain eiptests validate restored receipt metadata and tampered BAL preservation; a disabled consumer hash-check mutation fails the intended tests. Affected eth/P2P/chain/node/eiptest suites, targeted race, full build and vet passed. Full independent-client/devnet, eth/72/snap/2 and complete sync/recovery qualification remain separate program gates. [Evidence](../../verification/glamsterdam-eth71-20261118.md).

