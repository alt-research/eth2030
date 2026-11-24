# Independent encrypted snap/2 BAL qualification

Completed locally against master `5fedb7e5733fbef69767a711115ef9925e7bd769`. Two independently implemented Geth v1.17.10 servers run the stock eth and snap protocol handlers, real eth status/fork negotiation, RLPx TCP transport and production BAL database/query path. Both activate Amsterdam at zero and disable later Bogota research. The native P2P server negotiates both protocols; Node.snap2Clients admits the clients only after its real eth peer registry accepts status. No peer is manually registered or admitted through a test bypass.

Geth independently encodes and validates three distinct nonempty BALs, each 2,700,006 bytes. They are preseeded as retained archival data; this test does not claim Geth executed their blocks. The real soft response budget forces truncated prefixes. An unavailable first entry stays at position zero before a valid second entry. The actual Node.fetchSnapBALPath recovers all three correct headers/lists across the two peers, rejects a different header commitment and an unavailable required list, and recovers from the corrupt provider using the other peer. Final Amsterdam execution is below one second; the race run passes in 10.54 seconds.

Both complete affected race suites, final Amsterdam-targeted race, affected vet and both node builds pass. Two executed Go overlays prove that ignoring the header commitment or stopping after the first truncated prefix triggers the intended assertion failures, not compilation failures. The wire recorder preserves unchanged raw request/response RLP as gzip and decoded positional receipts. All 18 retained request/response artifacts are authenticated by size and SHA-256. CI uploads these artifacts for the exact checked-out commit; artifact output is a test-only setting, not a node runtime flag.

`VALIDATION.json` and `MUTATIONS.json` retain the checks. Native full block import is separately qualified by PR #163; whole-node moving/canonical pivot campaigns remain separate evidence. No result here attributes or closes original-04 / the 5.5-hour stall.

## Reproduce

From `pkg`, use Go 1.26.11:

```sh
ETH2030_INTEROP_OUTPUT=/tmp/snap2-interop go test -race ./node -run TestIndependentGethSnap2PositionalTruncationAndNativeRecovery -count=1 -timeout=90s -v
```

The test-only recorder writes compressed actual RLP and `exchanges.json` below the selected output directory. Omit the setting to use automatically cleaned test temporary directories. The test assertions always run.

Normative input: [EIP-8159](https://github.com/ethereum/EIPs/blob/d27448e08f60513856bbe228ea7ce04f156209dc/EIPS/eip-8159.md), [devp2p snap/2](https://github.com/ethereum/devp2p/blob/master/caps/snap.md), and the exact Geth v1.17.10 module pinned by go.mod/go.sum. The BAL shape/unavailable/commitment rules and snap-specific message IDs are qualified separately.
