# Consensus fixture replay

Run `go test ./...` and `go run . testdata/heze-a05a56de` from this directory.
This separate module pins the released ethpandaops Go client `v0.1.11`; it
does not add an explorer SDK to the execution client's dependencies.

The fixture is the real finalized Heze state at slot 192 from the coded
paymaster/blob control. Its manifest retains the Beacon header, runtime
revisions, spec response and the validator keys and balances from the matching
Beacon JSON response. The SSZ was independently fetched by the header's state
root and compared with the archived bytes before the enclave was removed.

The released Heze schema rejects these bytes: EIP-8015 removed four Eth1
fields that Lighthouse `a05a56de` still retains (92 fixed bytes). The local
replay type uses upstream's Gloas state field order and progressive-list tags,
with upstream's Heze execution bid. It therefore also adopts the current
execution requests root, PTC window and relocated latest execution hash,
which the March local codec lacks. Its field and Merkle organization follow
the upstream SDK. It is confined to the explicitly pinned experimental schema.

Replay tries the released codec first, then the compatibility type. Acceptance
requires exact SSZ byte roundtrip, the original Beacon state root, and every
validator identity and balance matching the captured JSON. Negative tests
change state bytes, header authority, validator data, the SSZ preset and the
execution requests root. Successful decoding alone is insufficient.

This qualifies a codec candidate. It does not wire Dora's epoch consumer,
qualify Dora restart/retention on that candidate, or retire the local service
fork. Those require a separately reviewed service migration. The upstream
schema remains the default candidate for a CL that implements EIP-8015.

Reference: [upstream state](https://github.com/ethpandaops/go-eth2-client/blob/v0.1.11/spec/gloas/beaconstate.go),
[upstream Heze bid](https://github.com/ethpandaops/go-eth2-client/blob/v0.1.11/spec/heze/executionpayloadbid.go),
[paired CL state](https://github.com/jiayaoqijia/lighthouse/blob/a05a56de03471aeffcdf3dbadd965af9da6462bf/consensus/types/src/state/beacon_state.rs).
