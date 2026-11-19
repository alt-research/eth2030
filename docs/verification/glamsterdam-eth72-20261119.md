# Sparse blob relay verification

Date: 2026-11-19. Branch: `feat/glamsterdam-eth72`. Tested production source: `d72a8d35dd2bea6bcf6794c30efcd88c9968ec00`, including the snap/2 test socket-lifecycle follow-up. The foundation is the submitted feature composition at `1efbd03a`; upstream master remains separate. A dedicated review compares the sparse relay changes against that foundation.

## Normative inputs and discussion decisions

[EIP 8070](https://github.com/ethereum/EIPs/blob/3db38a3725105cb67b8804ac3e147967113bc3ca/EIPS/eip-8070.md) defines provider/sampler behavior, custody updates, blob elision and Engine cell readback. [EIP 7594](https://github.com/ethereum/EIPs/blob/3db38a3725105cb67b8804ac3e147967113bc3ca/EIPS/eip-7594.md) defines the version-one wrapper, commitment binding, complete cell-proof matrix and 64-cell reconstruction threshold. Both complete EIPs were read before implementation.

The current [devp2p eth specification](https://github.com/ethereum/devp2p/blob/88f1c80800d622cc3a542a2125bbe33cc18c9f9c/caps/eth.md) resolves two wire ambiguities. Its file blob is `a23dc1725c1cd0e3b1319e0181a0124a577173c1`.

- The EIP table omits GetCells/Cells request IDs. Formal devp2p and [merged Geth PR 35428](https://github.com/ethereum/go-ethereum/pull/35428), reviewed on July 28, require flat correlated fields. Embedded nested request structs are incompatible.
- [Issue 290](https://github.com/ethereum/devp2p/issues/290) and [merged PR 291](https://github.com/ethereum/devp2p/pull/291), merged October 30, correct the earlier prose to blob-major ordering. The implemented position is `blob * popcount(mask) + requested-index-position`, with indices ascending inside each blob. Two-blob tests distinguish this from index-major order.

These are merged interoperability inputs, separate from the still-unresolved execution-gas alternatives. Unmodified Geth v1.17.10 codecs and API data structures provide independent consumers.

## Production path

The actual Node advertises eth/68 through eth/72 and negotiates the highest shared version. Version 72 reserves 22 message codes, adds GetCells 0x14/Cells 0x15 and the four-field announcement. Legacy versions retain their schemas. The old standalone nonce-announcement helper no longer claims that it defines eth/72; the historical experimental witness inventory remains outside production negotiation.

Pooled type-3 responses carry the signed payload, wrapper version one, raw empty list c0 for blobs, commitments and all cell proofs. The receiver rejects 80, full-data wrappers, noncanonical metadata and unsupported versions on this route. Legacy complete sidecars are converted to genuine cell proofs for version-72 metadata without mutating the sender's transaction. Nonblob transactions retain their existing encoding. The sender emits a nil mask when no blob is announced; the receiver tolerates the fixed-width ignored nonblob mask used by SDK peers.

A separate bounded staging store verifies canonical signed sender recovery, low-S signatures, chain/economic/nonce rules, commitment-to-versioned-hash binding and metadata dimensions. Ordinary pool admission still requires complete authenticated blob data. No sparse entry appears in executable pending contents. The store bounds metadata/cells to 256 entries and 64 MiB, expires entries after 20 minutes, clones external buffers and authenticates an entire cell group before publishing any of it. Cell proofs use the production ceremony batch verifier. At 64 verified indices per blob, native erasure recovery reconstructs the original blobs; the ordinary full-sidecar validator is the final promotion gate.

Each unknown blob transaction receives a stable process-local 15 percent provider decision. Samplers wait for two distinct provider announcements, fetch current custody indices from overlapping peers and request one random additional noncustody index when sampling a provider. Announcements group each transaction by its actual available mask, excluding sampling noise from custody advertisements. Providers request all available full data and announce all-ones only after complete promotion. Builders switch permanently to eager fetching after the first actual build request; eager nonprovider fetches prioritize 64-index reconstruction and can use sampler or provider availability.

Requests bind response kind, peer, request ID, transaction order and requested mask. Pending requests are bounded, expire, retry through live peer lifecycle workers and are cleared on disconnect. Partial/empty replies remain valid; unrequested, reordered, replayed, expired and dimensionally invalid responses fail. Cell requests accept up to 64 hashes; the server may answer one available transaction within the 2 MiB soft budget. Metadata requests accept up to 256 hashes with a bounded response. Complete-group truncation preserves common masks and all blobs for every returned index.

Actual Engine forkchoice V4/V5 consumes an exact 16-byte custody bitarray. Null and unchanged custody values have no blob-state effect while forkchoice processing continues. Updated custody drives subsequent worker requests and announcements; retained cells outside a contracted set are not advertised as custody. The backend enables permanent eager mode before asynchronous building, whose pool remains full-data only.

`engine_getBlobsV4` accepts at most 128 canonical versioned hashes and an exact 16-byte mask. It returns requested indices in ascending order, canonical hexadecimal DATA and matching proofs. Unknown blobs are null rows, while missing cells in a known sparse row are null positions, matching the SDK API consumer. Pending and future-nonce ordinary entries and verified staged cells are available. Pre-Osaka calls return null. This route does not claim historical cache coverage after pool eviction.

## Consumer verification

- Independent SDK packet tests verify flat GetCells/Cells fields and bitmap width, version-one metadata, raw c0 elision and nonempty multi-index ordering. Encrypted SDK RLPx sessions negotiate versions 68–72 and exercise actual status, receipts, BAL and version-72 cell dispatch.
- The production sparse store verifies two blobs, rejects a corrupted second-blob cell without partial publication, reconstructs from 64 even-numbered indices and compares both original blobs byte-for-byte. Commitment, version, proof width, signature, expiry and buffer ownership cases are checked.
- Three real encrypted ETH peers backed by native full-data pools qualify the sampler lifecycle. One provider yields no cell request; two yield exactly eight custody cells plus one extra. Expansion to sixteen custody cells fetches the delta without entering the ordinary pool. Eager mode then reconstructs complete signed data and removes duplicate staged storage. A fixed 10,000-hash control checks the 15 percent role policy.
- Two actual Nodes qualify HTTP submission, encrypted metadata/cell propagation, ordinary pool promotion, exact Engine cells 0/127 and independently verified proofs. A fresh receiving Node reconnects and reacquires authenticated data from the source pool. Unknown rows and malformed masks are checked through HTTP. Manually started servers stop before Node/database reopening; the original failed harness restart log is retained.
- Forkchoice consumer tests verify custody forwarding, identical/null behavior and continued FCU processing. Input tests reject short/oversized masks, cell-width aliases, unknown hash forms, excessive request counts and unrequested/expired/replayed response data.

Eight compiling behavioral mutations are detected: missing two-provider threshold; missing random extra; bypassed cell-proof verification; non-elided metadata; ignored peer identity; unrequested mask acceptance; omitted custody forwarding; omitted Engine cell projection. The first threshold fixture waited less than the actual retry period; extending it across that period makes premature sampling observable. All mutation edits are restored. [Exact results](glamsterdam-eth72-results-20261119.json).

## Reproducible checks

Run from `pkg`:

```sh
go test -count=1 ./...
go test -tags integration -count=1 ./core/... ./eiptest
go test -race -count=1 ./eth ./txpool/... ./engine/... ./node/... ./crypto/bls ./core/types
go vet ./eth ./txpool/... ./engine/... ./node/... ./crypto/bls ./core/types
go build ./...
```

All 212 tested Go packages, 27 tagged Core/EIP packages and 36 affected race-tested packages pass; affected vet and local build pass. Exact outcomes are recorded in the result JSON. Local complete logs use `/tmp/glamsterdam-eth72-final-{all,tagged,race}.log`; SDK/Node/sampler/mutation logs use the same prefix.

## Remaining program qualification

This qualifies real native Node propagation/reacquisition, a native-pool sampler network, independent SDK codecs and Engine consumers. It does not establish a full independent Geth node blob-gossip campaign, actual CL custody reconnect over a live multiclient devnet, production peer fairness/rate performance, long-lived sparse WAL recovery, mainnet memory/throughput or the final execution-gas/release matrix. Sparse staging intentionally restarts empty; the demonstrated recovery is authenticated peer reacquisition. Replacement policy is rechecked by the existing full pool on promotion. Those program gates remain explicit rather than being inferred from this feature PR.
