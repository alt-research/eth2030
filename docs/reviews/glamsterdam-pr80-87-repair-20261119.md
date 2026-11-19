# Glamsterdam reviews 80–87: merge and CI repair

Date: 2026-11-19. Initial master: `4f659fd096fe850b4a2dddaf906b05a853c59631` (through review 79). Eight reviews were handled individually in dependency order. This is merge/CI repair; release qualification and upstream merge decisions remain separate.

## Original CI causes and retained behavior

Seven original primary jobs stopped at gofmt: 80 lists `core/gas/blob_context.go`; 82–87 list that file and `eiptest/header_slot_interop_test.go`. Review 81 originally had seven successful jobs. Original complete failed logs are `/tmp/eth2030-pr<number>-initial-ci.log`. The blob formatting repair preserves every non-whitespace character. The header diagnostic formatting was already fixed in merged review 74. Workflows, required jobs, test assertions and production timeouts are retained.

| Review | Individual reconciliation | Existing repaired dependency |
| --- | --- | --- |
| 80 | Retain both IsOsaka and access-list mock methods; format blob helper. | Current master through 79 |
| 81 | Clean automatic master merge; verify encrypted snap/reorg/restart and unchanged-account proofs. | Current master through 79 |
| 82 | Preserve initcode-before-BAL guards; resolve mock-method relocation once; import the real latest snap proof-binding fix. | 80, 81 |
| 84 | Preserve actual deployed history/beacon transitions and mandatory queues; processor conflict is whitespace. | 82 |
| 83 (Draft) | Preserve canonical EIP-7778 and selected merged LIFO; no alternative gas proposal or readiness change. | 84 |
| 85 | Preserve ordinary zero-policy and method-specific simulation wrappers before early validation. | 83 |
| 86 | Keep validation-aware nonce policy and request-owned ETH-transfer observation with consensus fork guards. | 85 |
| 87 | Keep transaction/frame tracing and exactly one SELFDESTRUCT transfer observation before its synthetic trace child. | 86 |

The last conflict is semantically significant: keeping both inline emission and recordETHTransfer would duplicate protocol logs. The trace branch already invokes the helper before the frame hook. Its single call is retained, alongside all balance/BAL/destruction behavior. No new protocol policy is introduced.

## Local validation

Every review passes full default tests with Go 1.26.11 and `GOFLAGS=-p=4`: `go test ./... -timeout=15m -count=1`, whole-module gofmt, `go vet ./...` and `go build ./...`. Each also passes integration and affected race suites at the scopes listed in the [machine-readable manifest](glamsterdam-pr80-87-repair-20261119.json). Local logs use `/tmp/eth2030-pr<number>-{full,integration,race}.log`. Existing real HTTP, independent codec/commitment, encrypted networking, state recovery and execution assertions remain in place.

| Review | Qualified implementation head before final batch documentation | Full packages | Race packages | Current-head CI |
| --- | --- | --- | --- | --- |
| [#80](https://github.com/alt-research/eth2030/pull/80) | `9272ee39391b` | 211 | 7 | [Checks](https://github.com/alt-research/eth2030/pull/80/checks) |
| [#81](https://github.com/alt-research/eth2030/pull/81) | `0c8fd49b93b7` | 211 | 4 | [Checks](https://github.com/alt-research/eth2030/pull/81/checks) |
| [#82](https://github.com/alt-research/eth2030/pull/82) | `80219cd40be7` | 212 | 36 | [Checks](https://github.com/alt-research/eth2030/pull/82/checks) |
| [#83](https://github.com/alt-research/eth2030/pull/83) | `a339283acb74` | 212 | 35 | [Checks](https://github.com/alt-research/eth2030/pull/83/checks) |
| [#84](https://github.com/alt-research/eth2030/pull/84) | `c2727d4831c0` | 212 | 11 | [Checks](https://github.com/alt-research/eth2030/pull/84/checks) |
| [#85](https://github.com/alt-research/eth2030/pull/85) | `c41e61d60743` | 212 | 22 | [Checks](https://github.com/alt-research/eth2030/pull/85/checks) |
| [#86](https://github.com/alt-research/eth2030/pull/86) | `9d7ecb18a0b8` | 212 | 23 | [Checks](https://github.com/alt-research/eth2030/pull/86/checks) |
| [#87](https://github.com/alt-research/eth2030/pull/87) | `0746dcba8461` | 212 | 23 | [Checks](https://github.com/alt-research/eth2030/pull/87/checks) |

The final tracing documentation commit preserves its qualified source tree. For remote completion, verify the exact current published head and all seven required checks through the linked endpoints after publication; local or old-head success is separate evidence. Earlier per-feature ledgers are publication snapshots; this final batch manifest and its closure ledger supersede their temporary pending batch status.

## Future merge order

`80 → 81 → 82 → 84 → 83 → 85 → 86 → 87`

A read-only dependency-ordered virtual merge of all eight repaired implementations is clean at every stage. Its final `pkg/` tree exactly matches the qualified PR 87 tree. The test creates only unreferenced local commit objects and updates no branch refs. Review 81 remains independent of unmerged 80; later branches receive fixes only through already documented feature dependencies. Shared root CHECK/STATUS/AGENTS are byte-identical across the batch.

Draft 83 remains a separate review gate. Merging a dependent top review includes its existing gas ancestry, so follow the dependency review rather than treating each head as an independent feature. The original dirty checkout and unrelated worktrees are preserved. No upstream PR is merged by this task.
