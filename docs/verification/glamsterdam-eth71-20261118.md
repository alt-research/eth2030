# Negotiated historical data protocols

Base: `38ea9259`; branch: `feat/glamsterdam-eth71`. Inputs: local EIP-7642/7975/8159 reference texts, current published EIPs and devp2p `caps/eth.md` blob `a23dc1725c1cd0e3b1319e0181a0124a577173c1`. EIP-7928 canonical scalar BAL encoding remains the header commitment. No reference files or runtime feature switches were changed.

## Delivered behavior

- Production node advertises eth/68,69,70,71 and lets the existing RLPx capability negotiator choose the highest common version. Legacy 68 status remains six fields; 69+ uses seven-field available-range status without TD. Invalid ranges are rejected.
- Legacy 68 receipts retain consensus bloom encoding. Version 69 uses explicit transaction type and no bloom; 70/71 add pagination context. Shared modern receipt codecs preserve the existing canonical private Frame receipt case. Unknown receipt data stops the response instead of inventing an empty known page.
- Canonical-head updates queue bounded range notices without blocking FCU on a slow network writer. All peer writes are serialized; the notifier exits with peer lifecycle.
- BAL packet entries are raw RLP lists. The independent golden packet [1,[[]]] is c301c1c0; the baseline emitted c401c281c0 (a byte-string wrapper). Decoding rejects wrapped/trailing data and more than the bounded requested item count.
- Received BALs require outstanding peer/request metadata and corresponding local block headers. All nonempty entries are hash-verified before any restoration; replay, reordered and tampered packets fail. c0 can mean unavailable, or a valid empty BAL when that exact hash is committed.
- FetchBlockAccessList and FetchReceipts use the actual message loop and correlated response channels. Receipt continuation carries the correct block hash/offset; counts, per-transaction bytes and block totals are bounded, and complete lists must match the receipt trie root before storage. Decoding uses known request bounds before constructing receipt/log objects. Late/unsolicited receipt replies cannot populate storage.
- Receipt restoration and BAL restoration require an execution-validated local block and its commitment. Receipt cache eviction makes readback rebuild transaction/log metadata from persisted canonical records. No unexecuted block is promoted to validated by historical input.

The historical experimental AllCapabilities helper is not the production advertisement; its witness-shaped 72 entry is not EIP-8070. Actual 72/snap2 support remains separate work.

## Verification

- True encrypted TCP/RLPx sessions using independent Geth crypto, RLPx and RLP codecs negotiate each of 68/69/70/71. Tests validate actual seven/six-field status, nonempty receipt schemas and raw BAL unavailable encoding.
- Immutable receipt/BAL golden encodings test bytes, not merely local round trips.
- Correlated BAL consumer tests check successful storage, full-packet validation before writes, tamper, reordering, replay, unsolicited and unavailable input. Historical fetch runs both live message loops.
- Receipt fetch tests follow two pages and reject a changed root, no-progress pagination, too-large receipt and extra pages. A native eiptest commits a transfer and verifies persistent receipt restoration, derived metadata and rejection of bad counts/roots.
- Restoring the missing consumer hash-check behavior in a temporary mutation makes tamper/reordering tests fail; the production check was restored. The earlier wire golden test failed on the baseline for the expected wrapping difference.

Passed from pkg:

```sh
go test -count=1 ./eth ./core/chain ./p2p/... ./node/... ./eiptest
go test -race -count=1 ./eth ./p2p/ethproto ./core/chain ./eiptest -run 'TestNegotiatedEthVersions|TestCorrelatedBAL|TestEmptyBAL|TestValidatedBAL|TestBlockAccessListsCanonical|TestBALCodec|TestHistoricalBALFetch|TestReceiptConsumer|TestReceivedReceipts|TestReceiptDecoder'
go build ./...
go vet ./eth ./p2p/... ./core/chain ./node/backend
```

This is real-socket codec interoperability, not a full independent execution-client node/devnet campaign. No fresh cold sync, physical history pruning or entire 14,299-epoch run was performed. Those remain GL-043/061/062 qualification. Existing private Frame codec tests ran, but final public gas/schema compatibility remains under the program's versioned specification gates.
