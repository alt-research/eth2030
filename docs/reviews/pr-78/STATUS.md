# PR 78 tracking cleanup status

Original review head: `9ad2d9d61bd05885bc7c6703d8df9b9a1228fec7`. Merge base for preserved records: `38ea9259901c7873f483752ea390962e26563648`. Target master: `98716284b049933888bcd68d64c1957004d759b2`. Shared tracking policy: `42a9ceb86dea2fe4a74f01fd5e2360d6f21af0e7`. The sections below preserve this branch snapshot, including any inherited feature records; their historical validation is not a new result for this cleanup. The exact original diff, including removed text, is in [legacy-tracking.patch](legacy-tracking.patch).

Cleanup validation passed on the merged source tree: four-package core/config, core/vm, core/execution and txpool tests, and `go build ./...` from `pkg/`. Source changes outside the reviewed conflict resolution match Git's automatic three-way merge. Publication targets the existing review branch without force; live GitHub verification follows publication. Draft status and dependency order are unchanged.

## Preserved feature tracking

## Canonical schedule and Engine wire boundaries (2026-11-18)

Reject incoherent canonical genesis schedules in actual startup, add Osaka to fork identity and fix asymmetric activated-fork comparisons. Production Engine versions now enforce their timestamp windows; Osaka getPayloadV5 has its own correct shape. Actual HTTP tests cover required fields, DATA/QUANTITY bounds, zero values, malformed BAL, capabilities and forkchoice adoption before attribute errors. Full Go/tagged tests, affected race/vet/build and four failing behavioral mutations qualify this scope. See [verification](../../verification/glamsterdam-fork-schema-20261118.md). Persisted compatibility and broader program/release gates remain open.

