# PR 76 tracking cleanup status

Original review head: `bc2f58506b637fa19e93a5d5f3aefc8f0c966686`. Merge base for preserved records: `38ea9259901c7873f483752ea390962e26563648`. Target master: `98716284b049933888bcd68d64c1957004d759b2`. Shared tracking policy: `42a9ceb86dea2fe4a74f01fd5e2360d6f21af0e7`. The sections below preserve this branch snapshot, including any inherited feature records; their historical validation is not a new result for this cleanup. The exact original diff, including removed text, is in [legacy-tracking.patch](legacy-tracking.patch).

Cleanup validation passed on the merged source tree: four-package core/config, core/vm, core/execution and txpool tests, and `go build ./...` from `pkg/`. Source changes outside the reviewed conflict resolution match Git's automatic three-way merge. Publication targets the existing review branch without force; live GitHub verification follows publication. Draft status and dependency order are unchanged.

## Preserved feature tracking

## Ordinary blob proof validation and Engine publication (2026-11-18)

Reproduced and repaired invalid ordinary cell-proof admission through real HTTP. Preserve wrapper versions and pooled sidecars, authenticate KZG data, wait for complete pool data, derive genuine legacy proofs and expose canonical BlobAndProofV1 DATA. Shared payload collectors no longer fabricate zero proofs or publish missing data. Journal insert/rotation and fresh-node replay preserve authenticated sidecars. Frame checks remain qualified. Affected suites, targeted race/vet and build pass; four controlled mutations fail as intended. See [verification](../../verification/glamsterdam-blob-validation-20261118.md). Complete eth/72 sparse relay and release gates remain open.

