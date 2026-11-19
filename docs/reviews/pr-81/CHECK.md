## Sequential master/CI repair (2026-11-19)

- [x] Inspect original all-green CI and compare the exact branch with current master.
- [ ] Merge current master without importing unrelated unmerged blob or gas features.
- [ ] Verify source composition, real snap recovery tests, full default suite, integration, race/vet, formatting and build.
- [ ] Publish the existing branch without force and require fresh exact-head CI verification.

# PR 81 tracking cleanup

- [x] Preserve original tracking diff and changed sections from `6ce3eceeef539e69d5f2a20428aa7f88e651a661`.
- [x] Synchronize shared trackers and merge current master without dropping feature behavior.
- [x] Verify merged source, local build and affected tests.
- [x] Prepare the existing PR branch for fast-forward publication; require live conflict-state verification before reporting completion.

## Preserved feature tracking

## Glamsterdam eth/69-71 production protocols

## snap version 2 unchanged storage account proof binding (2026-11-19)

- [x] Reproduce a moving pivot with partial storage whose account body/storage root is unchanged.
- [x] Authenticate that account against the fresh pivot before using its storage root.
- [x] Reject a damaged proof for that unchanged account without checkpoint/adoption.
- [x] Qualify all encrypted Node scenarios, affected race/vet/build and update PR 81.

## snap/2 BAL state catch-up (2026-11-18)

- [x] Pin EIP-8189/7928 and trace real snap peer, state scheduler/writer and CL target adoption.
- [x] Negotiate snap/2 separately, retain unchanged range/code packets and reject removed trie-node messages.
- [x] Serve and authenticate ordered raw BAL responses with missing entries, budgets and cancellation.
- [x] Rebase downloaded state and outstanding ranges through verified BALs, including moving pivots and reorg handling.
- [x] Qualify actual Node sync/restart and independent wire/root consumers; run builds/tests and publish the separate PR.

- [x] Read EIP-7975/8159 and current devp2p status/message schemas; inspect the actual node registration.
- [x] Add independent canonical BAL-wire, versioned handshake and correlated tamper tests.
- [x] Register negotiated versions, implement status/range and version-aware receipt serving.
- [x] Correlate and validate received BALs before restoring verified historical data.
- [x] Verify affected network/chain/node/consumer paths and race/build; prepare the separate feature review.

## EIP-8282 builder execution requests (2026-11-18)

- [x] Read the complete pinned EIP, canonical bytecode assets and selected CL processing.
- [x] Add exact runtime/constructor assets and request types/SSZ encoding with provenance.
- [x] Wire mandatory Amsterdam system queues, rollback and commitment validation.
- [x] Supply explicit pre-fork deployment/genesis fixtures without fork-time code injection.
- [x] Qualify fee/queue/endian/cap/failure and actual Engine/CL consumer paths.
- [x] Run affected builds/tests and publish a dedicated feature PR.

