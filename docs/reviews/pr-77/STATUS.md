# PR 77 tracking cleanup status

Original review head: `6c86693815e154a092e6321e08d16109c86e49a2`. Merge base for preserved records: `38ea9259901c7873f483752ea390962e26563648`. Target master: `98716284b049933888bcd68d64c1957004d759b2`. Shared tracking policy: `42a9ceb86dea2fe4a74f01fd5e2360d6f21af0e7`. The sections below preserve this branch snapshot, including any inherited feature records; their historical validation is not a new result for this cleanup. The exact original diff, including removed text, is in [legacy-tracking.patch](legacy-tracking.patch).

Cleanup validation passed on the merged source tree: four-package core/config, core/vm, core/execution and txpool tests, and `go build ./...` from `pkg/`. Source changes outside the reviewed conflict resolution match Git's automatic three-way merge. Publication targets the existing review branch without force; live GitHub verification follows publication. Draft status and dependency order are unchanged.

## Preserved feature tracking

## EIP-8282 native builder requests (2026-11-18)

Canonical contracts and mandatory Amsterdam type-3/type-4 extraction now compose with the existing snapshot and Engine commitment path. Explicit owned genesis allocation precedes roots; missing production predeploys invalidate blocks. Actual HTTP and separate real-BLS Lighthouse component consumers qualify request bytes and registration/exit gates. Full Go, tagged Core/EIP, targeted race/vet/build and failing guard mutations pass. See [verification](../../verification/glamsterdam-builder-requests-20261118.md). Final gas, live full-client/devnet and release gates remain open.

