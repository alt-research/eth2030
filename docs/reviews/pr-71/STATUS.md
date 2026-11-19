# PR 71 tracking cleanup status

Original review head: `4d82484fe510a2fbea71db3a0be59fb8cc219d9c`. Merge base for preserved records: `38ea9259901c7873f483752ea390962e26563648`. Target master: `98716284b049933888bcd68d64c1957004d759b2`. Shared tracking policy: `42a9ceb86dea2fe4a74f01fd5e2360d6f21af0e7`. The sections below preserve this branch snapshot, including any inherited feature records; their historical validation is not a new result for this cleanup. The exact original diff, including removed text, is in [legacy-tracking.patch](legacy-tracking.patch).

Cleanup validation passed on the merged source tree: four-package core/config, core/vm, core/execution and txpool tests, and `go build ./...` from `pkg/`. The same four packages also passed race and vet checks. Source changes outside the reviewed conflict resolution match Git's automatic three-way merge. Publication targets the existing review branch without force; live GitHub verification follows publication. Draft status and dependency order are unchanged. The adjacent rule conflict retains Amsterdam ETH-log activation and the independently gated EIP-7954 helper.

## Preserved feature tracking

## Glamsterdam protocol ETH transfer logs (2026-11-18)

Branch `feat/glamsterdam-transfer-logs` activates 7708 at canonical Amsterdam independently of the private gas timestamp and removes extra fee-burn and self-beneficiary burn events. Specified transfers keep the ERC20-compatible system event; same-address/zero-value transfers are suppressed. Existing CREATE/CREATE2 endowment emission is retained and verified before constructor logs, with failed-call rollback preserved.

Native regressions reproduce baseline extra fees, missing Amsterdam activation and incorrect creation log counts; after correction, fork-boundary, fees/self/zero-value, ordering and rollback cases pass. The new logs exposed the modern P2P decoder zero-bloom bug, fixed in shared review #70 and included here. Affected core/P2P/Engine/node/RPC/eiptest suites, full build and targeted vet passed. Full 8246 destruction/finalization and the remaining gas model are separate gates; use a fresh experimental chain to qualify corrected historical fork semantics.

