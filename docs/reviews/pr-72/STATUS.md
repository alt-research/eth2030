# PR 72 tracking cleanup status

Original review head: `8cf3f9afde17acabeaffc2884580ae7ba3bb55bf`. Merge base for preserved records: `38ea9259901c7873f483752ea390962e26563648`. Target master: `98716284b049933888bcd68d64c1957004d759b2`. Shared tracking policy: `42a9ceb86dea2fe4a74f01fd5e2360d6f21af0e7`. The sections below preserve this branch snapshot, including any inherited feature records; their historical validation is not a new result for this cleanup. The exact original diff, including removed text, is in [legacy-tracking.patch](legacy-tracking.patch).

Cleanup validation passed on the merged source tree: four-package core/config, core/vm, core/execution and txpool tests, and `go build ./...` from `pkg/`. The same four packages also passed race and vet checks. Source changes outside the reviewed conflict resolution match Git's automatic three-way merge. Publication targets the existing review branch without force; live GitHub verification follows publication. Draft status and dependency order are unchanged. Initcode rejection remains before any BAL seeding; valid execution retains seeding and reconciliation.

## Preserved feature tracking

## Glamsterdam EIP-8246 execution and BAL (2026-11-18)

Implemented balance-preserving destruction, transaction-end account clearing, journaled native creations and rollback, correct CREATE2 creator nonce/prechecks, and final-state BAL reconciliation. Native and canonical Frame tests cover late credits, CREATE2 reuse, existing contracts, fork boundaries and four state backends. Affected suites, targeted race/vet and the local build pass. See [verification evidence](../../verification/glamsterdam-selfdestruct-20261118.md). Independent-client and final gas qualification remain open.

