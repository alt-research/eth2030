# Historical BAL availability qualification

Base: `38ea9259`; branch: `feat/glamsterdam-bal-retention`. Normative input: EIP-7928 at EIPs `3db38a3725105cb67b8804ac3e147967113bc3ca`, including merged PR #12347. Required minimum: 14,299 epochs, or 457,568 slots at 32 slots/epoch.

The bounded in-memory Engine now measures the availability window using beacon slot numbers where present. Legacy headers fall back to block distance, which retains extra data across missed slots rather than violating the minimum duration. Both hash/range consumers retain transaction/withdrawal bodies independently of BAL availability. The production persisted-chain consumer was already corrected on mainline and does not physically prune stored BALs here; retaining more than the minimum is allowed.

The consumer test constructs sparse header histories for old-window, exact-boundary, expired and missed-slot cases. It calls both Engine body APIs and checks canonical c0 BAL bytes or explicit unavailability. It is not a 14,299-epoch live run. Temporarily restoring the obsolete 3,533-epoch constant fails the old-window and minimum-boundary cases; the mutation was restored before committing.

Passed from pkg:

```sh
go test -count=1 ./core/rawdb ./core/chain ./engine/... ./node/backend ./eiptest
go build ./...
go vet ./core/rawdb ./engine/... ./node/backend
```

Existing validated BAL corruption/body-preservation and node restart coverage ran in the affected suites. No new physical-GC, cold-sync or independent-client campaign is claimed. Those remain broader GL-040/043/061/062 acceptance work.
