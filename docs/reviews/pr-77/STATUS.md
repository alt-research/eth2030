## Master integration after PRs 66–76 (2026-11-19)

Original PR head: `f720ab527f29876bbb16611fb3f640037620f570`. Target master: `56e8cd94eaa213e244a73f4ebd88f631a43b6ff2`. The only text conflict is in `TestReceiptSimpleTransferProtocolLog`: retain master's updated name and protocol-event/bloom assertions, and retain PR 77's `newBuilderRequestTestState()` fixture. This fixture installs the required builder contracts before execution. Root CHECK/STATUS/AGENTS remain identical to master; production sources match Git's automatic three-way merge.

The existing [pinned EIP-8282](https://github.com/ethereum/EIPs/blob/3db38a3725105cb67b8804ac3e147967113bc3ca/EIPS/eip-8282.md) requires both predeploys before activation and rejects blocks when either is missing. A controlled master-only fixture choice reproduces `request type 3: no code at 0x0000bff46984e3725691fa540a8c7589300d8282`. The combined fixture passes the actual post-Amsterdam receipt regression without disabling mandatory queue processing or reverting transfer-log expectations. No new protocol behavior is introduced by the manual conflict resolution.

The first full composition run also reproduced missing-predeploy failures in three newer master test fixtures: blob sidecar readiness, no factory injection at fork, and memory Engine slot commitment. Each now uses its existing package-local builder predeploy fixture. Test names, fork settings and assertions are retained, including rejection of missing contracts, absence of fork-time factory injection, blob readiness and the independent slot/header hash checks. These are test-only integration prerequisites; production validation is unchanged.

Final local validation passed with Go `1.26.11` from `pkg/`:

```sh
GOTOOLCHAIN=go1.26.11 GOFLAGS=-p=4 go test ./... -timeout=15m -count=1
GOTOOLCHAIN=go1.26.11 GOFLAGS=-p=4 go test -count=1 -tags integration ./core ./p2p/... ./eiptest
GOTOOLCHAIN=go1.26.11 GOFLAGS=-p=4 go test -race -count=1 ./core/block ./core/eips ./core/execution ./engine ./node ./eiptest
GOTOOLCHAIN=go1.26.11 GOFLAGS=-p=4 go vet ./...
GOTOOLCHAIN=go1.26.11 GOFLAGS=-p=4 go build ./...
```

The Go 1.26.11 formatting check is empty. The full default run passes 211 tested packages; the original integration run is unchanged by the three later test-only fixture repairs. Final full-test and race logs are `/tmp/eth2030-pr77-full-final.log` and `/tmp/eth2030-pr77-race-final.log`. Comparison with Git's automatic merge tree identifies exactly four manually reconciled test files and no production-source changes outside automatic merging. Scoped whitespace checks pass; pre-existing tracking archives imported unchanged from master retain their exact patch bytes. Publication will fast-forward the existing feature branch without upstream merge or changes to unrelated checkouts. Live exact-head and mergeability checks are required before reporting completion.

# PR 77 tracking cleanup status

Original review head: `6c86693815e154a092e6321e08d16109c86e49a2`. Merge base for preserved records: `38ea9259901c7873f483752ea390962e26563648`. Target master: `98716284b049933888bcd68d64c1957004d759b2`. Shared tracking policy: `42a9ceb86dea2fe4a74f01fd5e2360d6f21af0e7`. The sections below preserve this branch snapshot, including any inherited feature records; their historical validation is not a new result for this cleanup. The exact original diff, including removed text, is in [legacy-tracking.patch](legacy-tracking.patch).

Cleanup validation passed on the merged source tree: four-package core/config, core/vm, core/execution and txpool tests, and `go build ./...` from `pkg/`. Source changes outside the reviewed conflict resolution match Git's automatic three-way merge. Publication targets the existing review branch without force; live GitHub verification follows publication. Draft status and dependency order are unchanged.

## Preserved feature tracking

## EIP-8282 native builder requests (2026-11-18)

Canonical contracts and mandatory Amsterdam type-3/type-4 extraction now compose with the existing snapshot and Engine commitment path. Explicit owned genesis allocation precedes roots; missing production predeploys invalidate blocks. Actual HTTP and separate real-BLS Lighthouse component consumers qualify request bytes and registration/exit gates. Full Go, tagged Core/EIP, targeted race/vet/build and failing guard mutations pass. See [verification](../../verification/glamsterdam-builder-requests-20261118.md). Final gas, live full-client/devnet and release gates remain open.

