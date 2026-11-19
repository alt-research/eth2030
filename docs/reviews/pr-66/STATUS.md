# PR 66 tracking cleanup status

Original review head: `632853ed13073d6b061e0f6fb0e3d7a3fc38ab03`. Merge base for preserved records: `38ea9259901c7873f483752ea390962e26563648`. Target master: `98716284b049933888bcd68d64c1957004d759b2`. Shared tracking policy: `42a9ceb86dea2fe4a74f01fd5e2360d6f21af0e7`. The sections below preserve this branch snapshot, including any inherited feature records; their historical validation is not a new result for this cleanup. The exact original diff, including removed text, is in [legacy-tracking.patch](legacy-tracking.patch).

Cleanup validation passed on the merged source tree: four-package core/config, core/vm, core/execution and txpool tests, and `go build ./...` from `pkg/`. Source changes outside the reviewed conflict resolution match Git's automatic three-way merge. Publication targets the existing review branch without force; live GitHub verification follows publication. Draft status and dependency order are unchanged.

## Preserved feature tracking

## Historical empty BAL cache recovery (2026-11-18)

The bounded native Engine backend now regenerates the canonical empty BAL only when its header commitment matches, even without a cached BAL object. Both historical body APIs retain the execution body; nonempty/unknown commitments stay unavailable. This extends PR 66's slot-based retention qualification.

## Glamsterdam BAL availability window (2026-11-18)

Branch `feat/glamsterdam-bal-retention` updates EIP-7928 availability to 14,299 epochs. The bounded Engine backend uses actual header slots when available, with a conservative block-number fallback for legacy headers; both ByHash and ByRange preserve the body when BAL is unavailable. The production chain backend already persists validated canonical BALs on mainline and retains them beyond the minimum window.

Tests exercise the prior-window gap, exact cutoff, outside cutoff and missed slots through both Engine body APIs. A targeted mutation restoring 3,533 epochs fails the intended availability assertions. Affected storage/Engine/node/eiptest suites, build and targeted vet passed. This closes the obsolete-window subtask of GL-040; physical pruning, complete independent CL/recovery and full network qualification remain separate gates. [Evidence](../../verification/glamsterdam-bal-retention-20261118.md).

