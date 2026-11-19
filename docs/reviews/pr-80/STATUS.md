## Master/CI repair in the PR 80–87 batch (2026-11-19)

Original head: `c357d2a13f31e3d04397993aabf23d3892ad945c`. Merged master: `4f659fd096fe850b4a2dddaf906b05a853c59631` (through PR 79). The sole text conflict appends two independent mock methods: retain both the blob fork query `IsOsaka` and master's `CreateAccessList` fixture consumer. No production validation is removed.

The failed original CI run 44477321231 names only `core/gas/blob_context.go` in gofmt output; later vet/default tests were skipped. Formatting that file preserves every non-whitespace character. EIPs 7840/7892/7918 and the feature's pinned configuration/fee authority remain unchanged. Shared root trackers/instructions match master.

Go 1.26.11 validation passed: full default suite (`GOFLAGS=-p=4 go test ./... -timeout=15m -count=1`), integration (`./core ./p2p/... ./eiptest`), race (`./core/config ./core/gas ./core/execution ./engine ./node ./eiptest ./rpc/ethapi`), full vet, formatting and build. This includes the existing real signed blob-schedule/fee/restart scenario. Logs are `/tmp/eth2030-pr80-{full,race,integration}.log`; original CI failure is `/tmp/eth2030-pr80-initial-ci.log`.

The existing branch is prepared for fast-forward publication. Verify exact remote head and fresh target CI before completion; approval and upstream merging are separate. Later reviews will receive repairs only along already documented functional dependencies, preserving Draft 83 and avoiding importing it into earlier reviews.

# PR 80 tracking cleanup status

Original review head: `12e22aac2f7853092d8270beb644caaf1883f94d`. Merge base for preserved records: `38ea9259901c7873f483752ea390962e26563648`. Target master: `98716284b049933888bcd68d64c1957004d759b2`. Shared tracking policy: `42a9ceb86dea2fe4a74f01fd5e2360d6f21af0e7`. The sections below preserve this branch snapshot, including any inherited feature records; their historical validation is not a new result for this cleanup. The exact original diff, including removed text, is in [legacy-tracking.patch](legacy-tracking.patch).

Cleanup validation passed on the merged source tree: four-package core/config, core/vm, core/execution and txpool tests, and `go build ./...` from `pkg/`. Source changes outside the reviewed conflict resolution match Git's automatic three-way merge. Publication targets the existing review branch without force; live GitHub verification follows publication. Draft status and dependency order are unchanged.

## Preserved feature tracking

## Qualified blob schedule and settlement (2026-11-18)

The current version-1 signed HTTP scenario passes at 7/10/21 blobs through producer, independent receiver, receipts/RPC and fresh-node replay. Canonical per-fork capacity, pricing, fee burn/affordability, opcode context and persisted parameter authority agree with independent Geth formulas. Six behavioral mutations fail; full Go/tagged, affected race/vet/build pass after composing PR 76 and PR 79. The earlier legacy-wrapper race timeout is retained as failed evidence, not erased. See [verification](../../verification/glamsterdam-blob-schedule-20261118.md). Final gas, sparse relay/state sync, CL/devnet and release gates remain open.

## Ordinary blob proof validation and Engine publication (2026-11-18)

Reproduced and repaired invalid ordinary cell-proof admission through real HTTP. Preserve wrapper versions and pooled sidecars, authenticate KZG data, wait for complete pool data, derive genuine legacy proofs and expose canonical BlobAndProofV1 DATA. Shared payload collectors no longer fabricate zero proofs or publish missing data. Journal insert/rotation and fresh-node replay preserve authenticated sidecars. Frame checks remain qualified. Affected suites, targeted race/vet and build pass; four controlled mutations fail as intended. See [verification](../../verification/glamsterdam-blob-validation-20261118.md). Complete eth/72 sparse relay and release gates remain open.

## Blob schedule implementation in progress (2026-11-18)

Fork-aware block capacity, pricing, ordinary fee settlement, opcode/receipt/RPC consumers and metadata history guards are implemented. Signed HTTP producer/receiver controls pass at 7/10/21 blobs. The full module and tagged suites pass; the 21-blob race run hits the unchanged Engine timeout in the legacy proof-generation route. Compose the submitted PR 76 proof path and qualify current version-1 wrappers before completion. Final gas/release gates remain open.

## Persisted configuration recovery (2026-11-18)

Actual Node startup now checks persisted runtime configuration before history replay, rejects active or immutable changes and retains compatible future updates. Runtime overrides do not mutate network globals. Real signed HTTP/restart evidence preserves head/state/config bytes on rejection and verifies successful future updates; missing/corrupt authority is explicit. Full Go, affected race/vet/build and four failing behavioral mutations pass. See [verification](../../verification/glamsterdam-config-recovery-20261118.md). Final gas, blob schedules, unknown legacy migration and release gates remain open.

