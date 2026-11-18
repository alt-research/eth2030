# Snap version 2 state catch up verification

Date: 2026-11-19. Branch: `feat/glamsterdam-snap2`. This feature composes eth/69–71 PR 69 and the explicit Amsterdam genesis/request fixtures from PR 77. Both dependencies are submitted separately. No upstream branch or reference submodule was modified.

## Specification and production behavior

[EIP 8189](https://github.com/ethereum/EIPs/blob/3db38a3725105cb67b8804ac3e147967113bc3ca/EIPS/eip-8189.md) defines the independent snap/2 capability, messages 08/09, positional unavailable markers, byte budgets and moving/reorganized pivot recovery. [EIP 7928](https://github.com/ethereum/EIPs/blob/3db38a3725105cb67b8804ac3e147967113bc3ca/EIPS/eip-7928.md) defines the canonical account-grouped BAL commitment. The complete 8189 text and relevant 7928 source were read before implementation.

The actual Node registers both snap capabilities, negotiates the highest shared version and selects one for each synchronization attempt. Version 1 retains its trie healing path; version 2 rejects messages 06/07. Existing account, storage and bytecode messages retain their wire schemas. Local complete snapshots rebuild trie nodes without requesting removed messages.

Requests are limited to 64 hashes. Responses preserve order, use a 2 MiB soft budget and a 16 MiB hard message limit, with envelope space reserved. Missing BALs are raw RLP 80; an available empty BAL is raw c0. Tail truncation is permitted, and a first BAL may exceed the soft budget within the hard bound. The client correlates requests and verifies every available BAL against its exact header before returning it. Canonical decoding checks ordered unique accounts/slots, disjoint reads/writes, field widths, uint32 indices and header item budgets. Zero post-values remain deletions.

The scheduler resolves only CL-selected finalized targets. It validates both paths through the common ancestor, authenticates all old/new BALs and stages a copy of downloaded state. Exact new-root account proofs restore metadata or authenticated absence. Old-only storage entries are fetched by exact inclusion/absence proof, then new-fork post-values apply in block order. Completed storage roots must remain consistent. Partial range cursors survive the pivot update; storage queues deduplicate accounts. Snapshot, queues, pivot and proof adapter change only after the staged update succeeds. Missing required orphan BALs cause fresh bulk synchronization rather than reuse of unauthenticated partial state. Root reconstruction and the current CL target gate final adoption and persisted snap-pivot authority.

## Real consumers and regression evidence

`TestSnap2EncryptedExchangeWithIndependentSDKPackets` opens an encrypted TCP/RLPx session with unmodified Geth v1.17.10 crypto, RLPx/Snappy and snap packet codecs. It negotiates 2 over 1, decodes actual server missing/empty entries and checks disconnection for removed trie requests. The test exposed a real generic-encoder bug: helper round trips emitted c0 correctly, while the production sender wrapped it as 81c0. Explicit custom encoding/decoding in actual message paths fixes that failure.

`TestSnap2ActualMovingPivotReorgAndMissingOrphanRecovery` runs the real Node synchronization method over encrypted TCP between native nodes, using independently signed transactions and the actual chain state/proof backend. A serving-policy wrapper expires the old root only after verified progress and advances the receiving CL execution target. Its five scenarios are:

| Scenario | Required consumer result |
| --- | --- |
| Moving account ranges | Point-proof rebase occurs; final CL-selected root/head and fresh-node readback match |
| Moving partial storage | A new slot before the cursor is applied while the nonzero cursor and untouched suffix survive; no full storage restart |
| Reorganized partial storage | Old balance/storage effects are removed, slot zero restores its pre-fork value, old-only newly created slot is deleted, new-fork writes survive |
| Missing orphan BAL | Partial progress is discarded and fresh bulk synchronization adopts the new root; persisted authority survives restart |
| Invalid new point proof | Correct leaf values with a damaged point proof are rejected before adoption or snap-pivot publication |

Proof corruption is confined to the contract point query. Later bulk proofs stay valid, so their range checks cannot mask a missing point guard. The original broader corruption fixture failed this mutation-quality check and was narrowed before qualification.

Seven compiling mutations each fail an intended behavioral assertion: generic production wire encoding; omitted BAL commitment check; reset storage cursor; skipped old-only slot restoration; refused fresh restart for missing orphan data; bypassed point proof verification; accepted removed trie requests. Their source edits are restored. Exact outcomes are retained in [the result record](glamsterdam-snap2-results-20261119.json).

## Checks and remaining qualification

Run from `pkg`:

```sh
go test -count=1 ./...
go test -tags integration -count=1 ./core/... ./eiptest
go test -count=1 ./bal/... ./core/state/... ./p2p/snap/... ./sync/statesync/... ./sync/beacon/... ./node/...
go test -race -count=1 ./bal/... ./core/state/... ./p2p/snap/... ./sync/statesync/... ./sync/beacon/... ./node/...
go vet ./bal/... ./core/state/... ./p2p/snap/... ./sync/statesync/... ./sync/beacon/... ./node/...
go build ./...
```

Full tests and tagged tests pass. Affected checks were repeated after the isolated proof-fixture correction. Local logs are `/tmp/glamsterdam-snap2-all-4.log`, `-tagged-4.log`, `-final-affected.log`, `-final-race.log`, the seven mutation logs and the SDK/Node wire logs.

This qualifies native Node moving-pivot/reorg/restart behavior and independent SDK wire decoding. It does not establish full Geth cold-sync interoperability, physical retention/pruning over 14,299 epochs, mainnet-scale throughput/memory, a peer reputation/rate policy or complete CL/multiclient release acceptance. Snapshot assembly and staging still materialize state in memory. eth/72 sparse blob custody/provider/sampler/reconstruction and the selected execution gas model remain separate tasks.

## Explicit socket lifecycle follow up

The manually started P2P servers now have explicit test cleanup, and the receiver server is stopped before closing/reopening its Node database. This is required because the test does not invoke full Node.Start; Node.Stop's not-running path does not own an independently started server. All five real Node scenarios pass again under race, with affected vet and local build. The production protocol behavior is unchanged.
