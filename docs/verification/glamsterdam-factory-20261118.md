# Canonical deterministic factory

Base: origin/master `38ea9259`; branch: `feat/glamsterdam-factory`. Normative input: EIP-7997 at EIPs `3db38a3725105cb67b8804ac3e147967113bc3ca`. The runtime was extracted byte-for-byte from that specification (69 bytes), not reconstructed from the obsolete local draft.

The existing ApplyEIP7997 helper now explicitly constructs canonical genesis/test state at 0x4e59b44847b379578588920cA78FbF26c0B4956C with nonce 1. Block execution and both builders no longer check for or inject factory code at activation. Existing networks satisfy the requirement through an ordinary deployment; new private genesis allocations should use this exact account. The program does not silently rewrite supplied genesis roots or persisted chain state.

Native EVM tests execute the factory's real CREATE2 code, calculate the expected address independently, require the specified 20-byte return data, check deployed code/value, and reject collision/short-salt input. A processor regression proves an empty fork-active block does not inject either the canonical factory or the obsolete 0x12 contract. These assertions failed before the implementation and passed after it.

Canonical Ethereum BLS remains 0x0b..0x11. The legacy private layout retains its 0x12 map precompile; it no longer collides with an invented factory there. This does not activate legacy BLS on standard networks.

Passed from pkg:

```sh
go test -count=1 ./core/vm ./core/eips ./core/execution ./core/block ./engine/... ./node/... ./eiptest
go build ./...
go vet ./core/eips ./core/execution ./core/block
```

No new live devnet or independent-client deployment run is claimed. Operator-generated genesis inputs and full cross-client activation remain integration acceptance work; absence of the factory is not repaired by consensus execution.
