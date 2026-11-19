## Master integration after PRs 66–78 (2026-11-19)

Original head: `86cbc4fa250c546967a09f48adbf02b08fca116a`. Target master: `6b16bdf8ef58ef9f8b67bb16b44dbbb40fcd8d59`. Three conflict hunks are reconciled: keep PR 79's BPO1/BPO2 registry entries, keep its 24-entry schedule assertion, and retain master's Osaka activation in the owned minimal genesis fixture. Osaka appears once in the merged registry. Root CHECK/STATUS/AGENTS remain identical to master.

The local EIP-7892 reference was read in full. BPO activation timestamps remain registered configuration authority, and optional BPOs can interleave with later named forks. The automatically merged canonical schedule validation still requires Shanghai/Cancun/Prague/Osaka predecessors for Amsterdam, while BPO ordering is validated separately. Node startup retains canonical validation and PR 79's persisted-history compatibility check before blockchain replay/adoption; neither guard is bypassed.

The existing targeted configuration and Node startup/restart regressions pass after reconciliation. They cover the full schedule registry, BPO ordering and compatibility, actual persisted authority, canonical predecessor rejection and supplied genesis loading. No new protocol policy or migration rule is introduced. Comparison with Git's automatic merge tree confirms only the three reviewed conflict selections differ from automatic source merging.

Final local validation passed using Go `1.26.11` from `pkg/`:

```sh
GOTOOLCHAIN=go1.26.11 GOFLAGS=-p=4 go test ./... -timeout=15m -count=1
GOTOOLCHAIN=go1.26.11 GOFLAGS=-p=4 go test -count=1 -tags integration ./core ./p2p/... ./eiptest
GOTOOLCHAIN=go1.26.11 GOFLAGS=-p=4 go test -race -count=1 ./core/config ./node/... ./eiptest
GOTOOLCHAIN=go1.26.11 GOFLAGS=-p=4 go vet ./...
GOTOOLCHAIN=go1.26.11 GOFLAGS=-p=4 go build ./...
```

The full default run passes 211 tested packages; race passes seven packages. The Go 1.26.11 formatting check is empty. Logs: `/tmp/eth2030-pr79-{full,integration,race}.log`. This includes the complete existing Node/configuration suites, actual signed HTTP/restart checks and canonical Engine schema regressions. Scoped whitespace checks pass; unchanged incoming tracking archives retain their original patch bytes. Publication will fast-forward the existing PR branch without force or upstream merge. Exact-head, live mergeability and target CI verification are required before completion.

# PR 79 tracking cleanup status

Original review head: `901ec06331bd5290c733adf23c10d9e670dabd2c`. Merge base for preserved records: `38ea9259901c7873f483752ea390962e26563648`. Target master: `98716284b049933888bcd68d64c1957004d759b2`. Shared tracking policy: `42a9ceb86dea2fe4a74f01fd5e2360d6f21af0e7`. The sections below preserve this branch snapshot, including any inherited feature records; their historical validation is not a new result for this cleanup. The exact original diff, including removed text, is in [legacy-tracking.patch](legacy-tracking.patch).

Cleanup validation passed on the merged source tree: four-package core/config, core/vm, core/execution and txpool tests, and `go build ./...` from `pkg/`. Source changes outside the reviewed conflict resolution match Git's automatic three-way merge. Publication targets the existing review branch without force; live GitHub verification follows publication. Draft status and dependency order are unchanged.

## Preserved feature tracking

## Persisted configuration recovery (2026-11-18)

Actual Node startup now checks persisted runtime configuration before history replay, rejects active or immutable changes and retains compatible future updates. Runtime overrides do not mutate network globals. Real signed HTTP/restart evidence preserves head/state/config bytes on rejection and verifies successful future updates; missing/corrupt authority is explicit. Full Go, affected race/vet/build and four failing behavioral mutations pass. See [verification](../../verification/glamsterdam-config-recovery-20261118.md). Final gas, blob schedules, unknown legacy migration and release gates remain open.

