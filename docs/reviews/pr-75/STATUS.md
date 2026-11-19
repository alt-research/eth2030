# PR 75 tracking cleanup status

Original review head: `3ee4dae928a86f1d378dc3ced2e31ce4a8e3bfd0`. Merge base for preserved records: `38ea9259901c7873f483752ea390962e26563648`. Target master: `98716284b049933888bcd68d64c1957004d759b2`. Shared tracking policy: `42a9ceb86dea2fe4a74f01fd5e2360d6f21af0e7`. The sections below preserve this branch snapshot, including any inherited feature records; their historical validation is not a new result for this cleanup. The exact original diff, including removed text, is in [legacy-tracking.patch](legacy-tracking.patch).

Cleanup validation is pending. Publication must be a fast-forward update of the existing review branch. Draft status and dependency order are unchanged.

## Preserved feature tracking

## Glamsterdam independent gas corpus (2026-11-18)

Added an offline signed-transaction differential command with 22 scenarios across Osaka controls and a pinned Amsterdam reference (44 comparisons). Reports preserve complete inputs, gas dimensions, validity versus included failure, logs and final state. The independent context explicitly includes CPSB; a targeted mutation proves its sanity check detects missing state creation pricing. Scoped tests/race/vet and local build pass. This is mismatch evidence, not release parity; see [verification](../../verification/glamsterdam-gas-corpus-20261118.md).

