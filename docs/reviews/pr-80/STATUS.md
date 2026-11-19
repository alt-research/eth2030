# PR 80 tracking cleanup status

Original review head: `12e22aac2f7853092d8270beb644caaf1883f94d`. Merge base for preserved records: `38ea9259901c7873f483752ea390962e26563648`. Target master: `98716284b049933888bcd68d64c1957004d759b2`. Shared tracking policy: `42a9ceb86dea2fe4a74f01fd5e2360d6f21af0e7`. The sections below preserve this branch snapshot, including any inherited feature records; their historical validation is not a new result for this cleanup. The exact original diff, including removed text, is in [legacy-tracking.patch](legacy-tracking.patch).

Cleanup validation is pending. Publication must be a fast-forward update of the existing review branch. Draft status and dependency order are unchanged.

## Preserved feature tracking

## Qualified blob schedule and settlement (2026-11-18)

The current version-1 signed HTTP scenario passes at 7/10/21 blobs through producer, independent receiver, receipts/RPC and fresh-node replay. Canonical per-fork capacity, pricing, fee burn/affordability, opcode context and persisted parameter authority agree with independent Geth formulas. Six behavioral mutations fail; full Go/tagged, affected race/vet/build pass after composing PR 76 and PR 79. The earlier legacy-wrapper race timeout is retained as failed evidence, not erased. See [verification](../../verification/glamsterdam-blob-schedule-20261118.md). Final gas, sparse relay/state sync, CL/devnet and release gates remain open.

## Ordinary blob proof validation and Engine publication (2026-11-18)

Reproduced and repaired invalid ordinary cell-proof admission through real HTTP. Preserve wrapper versions and pooled sidecars, authenticate KZG data, wait for complete pool data, derive genuine legacy proofs and expose canonical BlobAndProofV1 DATA. Shared payload collectors no longer fabricate zero proofs or publish missing data. Journal insert/rotation and fresh-node replay preserve authenticated sidecars. Frame checks remain qualified. Affected suites, targeted race/vet and build pass; four controlled mutations fail as intended. See [verification](../../verification/glamsterdam-blob-validation-20261118.md). Complete eth/72 sparse relay and release gates remain open.

## Blob schedule implementation in progress (2026-11-18)

Fork-aware block capacity, pricing, ordinary fee settlement, opcode/receipt/RPC consumers and metadata history guards are implemented. Signed HTTP producer/receiver controls pass at 7/10/21 blobs. The full module and tagged suites pass; the 21-blob race run hits the unchanged Engine timeout in the legacy proof-generation route. Compose the submitted PR 76 proof path and qualify current version-1 wrappers before completion. Final gas/release gates remain open.

## Persisted configuration recovery (2026-11-18)

Actual Node startup now checks persisted runtime configuration before history replay, rejects active or immutable changes and retains compatible future updates. Runtime overrides do not mutate network globals. Real signed HTTP/restart evidence preserves head/state/config bytes on rejection and verifies successful future updates; missing/corrupt authority is explicit. Full Go, affected race/vet/build and four failing behavioral mutations pass. See [verification](../../verification/glamsterdam-config-recovery-20261118.md). Final gas, blob schedules, unknown legacy migration and release gates remain open.

