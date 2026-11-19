# Protocol transfer-log qualification

Base: `38ea9259`; branch: `feat/glamsterdam-transfer-logs`. EIP-7708 pinned to EIPs `3db38a3725105cb67b8804ac3e147967113bc3ca`; the older local draft includes burns and was not modified. Canonical Amsterdam now enables transfer logs without waiting for the separate legacy private gas-repricing timestamp. Its compatibility activation is retained, without a new runtime switch.

Consensus execution no longer emits base-fee burn events or SELFDESTRUCT-to-self burn events. The shared transfer helper suppresses self/zero transfers. Existing actual CALL/CREATE/CREATE2 value-transfer ordering is retained. A legacy analytics burn helper remains available but is not called by these consensus paths.

Native tests prove:

- Nonzero base fees and tips do not add logs to zero-value or self transfers; genuine transfers emit one system Transfer event with the correct amount.
- The exact canonical Amsterdam boundary changes emission, without activating private repricing.
- Contract creation with endowment emits the system transfer before a constructor LOG0; no fee event follows it.
- A reverted value transfer leaves a failed receipt with no transfer/fee logs.
- SELFDESTRUCT to an existing self beneficiary emits no fictitious burn and keeps its balance.

A controlled restoration of the baseline production files makes the new integration tests fail for extra fee logs, inactive Amsterdam logs and creation log count/order. Production files were restored before the passing run. Older post-Amsterdam tests expecting no transfer events were updated to check the required protocol events and nonzero matching blooms.

Nonzero logs also exposed an existing eth/69+ decode bug: CreateBloom only ORs existing receipt blooms and cannot initialize a new decoded receipt's bloom. Shared PR #70 replaces it with LogsBloom and adds a typed-receipt exact consensus-byte roundtrip. That fix is included here so pagination/root consumers retain the commitment.

Passed from pkg:

```sh
go test -count=1 ./core/... ./p2p/... ./engine/... ./node/... ./rpc/... ./eiptest
go build ./...
go vet ./core/config ./core/execution ./core/vm ./p2p/ethproto
```

This does not close full EIP-8246 account finalization, the final 8037/8038 economic model or independent-client/devnet qualification. Correcting logs changes receipt/block hashes compared with obsolete experimental Amsterdam semantics: qualify on a fresh chain or a separately documented migration, not by silently rewriting old persisted history. Official unscheduled public fork times are unchanged.
