## Master integration after PR 71 (2026-11-19)

Starting PR head: `7d60edb7f3680a2d5eaf0c0edca253491b83f080`. Merged master: `e90422b4f6a3bdddb6ef124b4b8dd51cd7e8d57a`, which includes reviews 66–71. The only unresolved source hunk was in `opSelfdestruct`: retain the EIP-8246 creation/destruction journal and balance-preserving finalization, while removing the obsolete conditional Burn event as required by merged PR 71. Root CHECK/STATUS and the shared tracking policy remain identical to master.

Authority remains EIPs `3db38a3725105cb67b8804ac3e147967113bc3ca`: [EIP-7708](https://github.com/ethereum/EIPs/blob/3db38a3725105cb67b8804ac3e147967113bc3ca/EIPS/eip-7708.md) logs real transfers to a different beneficiary and excludes burn events; [EIP-8246](https://github.com/ethereum/EIPs/blob/3db38a3725105cb67b8804ac3e147967113bc3ca/EIPS/eip-8246.md) preserves balances while clearing same-transaction-created accounts at finalization. Local older draft references are unchanged.

Keeping the old PR-side conflict hunk makes `TestEIP7708SelfdestructToSelfEmitsNoLog` fail with one unexpected log. Removing it passes that regression. Native transaction finalization coverage now also checks exact receipt event count, emitter, topics, amount and ordering together with persisted final account state. Added prefunded and zero-endowment beneficiary cases distinguish creation endowment from the larger SELFDESTRUCT transfer. Test fixtures install the actual transaction log context, matching ProcessWithBAL, before reading receipt logs.

Final validation passed from `pkg/`:

```sh
go test -count=1 ./bal/... ./core/... ./engine/... ./node/... ./p2p/... ./rpc/... ./eiptest
go test -race -count=1 ./bal ./core/vm ./core/execution ./core/state
go vet ./bal ./core/config ./core/vm ./core/execution ./core/state
go build ./...
```

The source comparison confirms that instructions.go differs from the automatic three-way merge only by choosing master's empty replacement for the obsolete Burn-event conflict hunk. Native receipt/final-state regression assertions are the only additional source change beyond the merged parents. Live exact-head and GitHub mergeability verification follows the fast-forward publication. This updates only the existing PR 72 branch without force or upstream merge; original unrelated checkouts remain untouched.

# PR 72 tracking cleanup status

Original review head: `8cf3f9afde17acabeaffc2884580ae7ba3bb55bf`. Merge base for preserved records: `38ea9259901c7873f483752ea390962e26563648`. Target master: `98716284b049933888bcd68d64c1957004d759b2`. Shared tracking policy: `42a9ceb86dea2fe4a74f01fd5e2360d6f21af0e7`. The sections below preserve this branch snapshot, including any inherited feature records; their historical validation is not a new result for this cleanup. The exact original diff, including removed text, is in [legacy-tracking.patch](legacy-tracking.patch).

Cleanup validation passed on the merged source tree: four-package core/config, core/vm, core/execution and txpool tests, and `go build ./...` from `pkg/`. The same four packages also passed race and vet checks. Source changes outside the reviewed conflict resolution match Git's automatic three-way merge. Publication targets the existing review branch without force; live GitHub verification follows publication. Draft status and dependency order are unchanged. Initcode rejection remains before any BAL seeding; valid execution retains seeding and reconciliation.

## Preserved feature tracking

## Glamsterdam EIP-8246 execution and BAL (2026-11-18)

Implemented balance-preserving destruction, transaction-end account clearing, journaled native creations and rollback, correct CREATE2 creator nonce/prechecks, and final-state BAL reconciliation. Native and canonical Frame tests cover late credits, CREATE2 reuse, existing contracts, fork boundaries and four state backends. Affected suites, targeted race/vet and the local build pass. See [verification evidence](../../verification/glamsterdam-selfdestruct-20261118.md). Independent-client and final gas qualification remain open.

