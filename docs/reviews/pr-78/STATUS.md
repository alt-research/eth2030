## Master integration after PRs 66–77 (2026-11-19)

Original head: `95d363d6f6a2fecb4efd1b4b9ef2614ec45fe2b7`. Merged master: `f0b37671200dc7e8060006e1c86feb6b51356c69`. The only text conflicts are the async and stored return paths in `EngineBackend.GetPayloadV5`. Both now retain master's `BuildBlobsBundleV2` construction and error propagation, while returning PR 78's explicitly named `LegacyGlamsterdamPayloadResponse`. The formal Osaka HTTP method remains `EngineAPI.GetPayloadV5` with the V3 payload shape and fork checks; Amsterdam remains V6. No interface is renamed back or aliased across fork versions.

Authority remains the existing Engine specification pin `68d76a5b1c09dff023782d97eec6ce87a6f6505e` ([Osaka](https://github.com/ethereum/execution-apis/blob/68d76a5b1c09dff023782d97eec6ce87a6f6505e/src/engine/osaka.md), [Amsterdam](https://github.com/ethereum/execution-apis/blob/68d76a5b1c09dff023782d97eec6ce87a6f6505e/src/engine/amsterdam.md)). The two local reference documents were read in full for payload version, slot/BAL, cell-proof bundle and fork-boundary requirements.

`TestLegacyPayloadV5RetainsBlobBundle` exercises both return paths with a real KZG commitment and 128 cell proofs, checks unchanged blob/commitment/proof bytes and the supplied slot/block hash, and rejects publication with a missing sidecar. This is a response-construction unit test with prepared pending results; existing actual HTTP regressions qualify the canonical Osaka/Amsterdam consumer separately. A controlled mutation replaces both bundle builders with empty bundle objects. All four cases then fail on the intended missing-data/proof assertions; production source is restored before the full validation run. Mutation output: `/tmp/eth2030-pr78-bundle-mutation.log`.

Final validation passed with Go `1.26.11` from `pkg/`:

```sh
GOTOOLCHAIN=go1.26.11 GOFLAGS=-p=4 go test ./... -timeout=15m -count=1
GOTOOLCHAIN=go1.26.11 GOFLAGS=-p=4 go test -count=1 -tags integration ./core ./p2p/... ./eiptest
GOTOOLCHAIN=go1.26.11 GOFLAGS=-p=4 go test -race -count=1 ./core/config ./engine/... ./node/... ./eiptest
GOTOOLCHAIN=go1.26.11 GOFLAGS=-p=4 go vet ./...
GOTOOLCHAIN=go1.26.11 GOFLAGS=-p=4 go build ./...
```

The full default run passes 211 tested packages; race passes 23 tested packages. Go 1.26.11 `gofmt -l .` is empty. Logs: `/tmp/eth2030-pr78-{full,integration,race}.log`. The existing actual HTTP schema/fork regression and the zero/nondefault/maximum-slot tests pass together with the new legacy-adapter bundle regression. Scoped whitespace checks pass; unchanged upstream tracking archives preserve their original patch bytes. Root CHECK/STATUS/AGENTS are unchanged from master. Only the two combined conflict hunks and the dedicated regression differ from automatic source merging. Publication will update the existing PR branch without force or upstream merge, followed by exact-head, mergeability and CI verification.

# PR 78 tracking cleanup status

Original review head: `9ad2d9d61bd05885bc7c6703d8df9b9a1228fec7`. Merge base for preserved records: `38ea9259901c7873f483752ea390962e26563648`. Target master: `98716284b049933888bcd68d64c1957004d759b2`. Shared tracking policy: `42a9ceb86dea2fe4a74f01fd5e2360d6f21af0e7`. The sections below preserve this branch snapshot, including any inherited feature records; their historical validation is not a new result for this cleanup. The exact original diff, including removed text, is in [legacy-tracking.patch](legacy-tracking.patch).

Cleanup validation passed on the merged source tree: four-package core/config, core/vm, core/execution and txpool tests, and `go build ./...` from `pkg/`. Source changes outside the reviewed conflict resolution match Git's automatic three-way merge. Publication targets the existing review branch without force; live GitHub verification follows publication. Draft status and dependency order are unchanged.

## Preserved feature tracking

## Canonical schedule and Engine wire boundaries (2026-11-18)

Reject incoherent canonical genesis schedules in actual startup, add Osaka to fork identity and fix asymmetric activated-fork comparisons. Production Engine versions now enforce their timestamp windows; Osaka getPayloadV5 has its own correct shape. Actual HTTP tests cover required fields, DATA/QUANTITY bounds, zero values, malformed BAL, capabilities and forkchoice adoption before attribute errors. Full Go/tagged tests, affected race/vet/build and four failing behavioral mutations qualify this scope. See [verification](../../verification/glamsterdam-fork-schema-20261118.md). Persisted compatibility and broader program/release gates remain open.

