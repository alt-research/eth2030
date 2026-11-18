# RPC access-list discovery qualification

Base: `38ea9259`; branch: `feat/glamsterdam-access-list`. Requirements: EIP-2930 access items and merged execution-apis #950 (actual discovered list, not cheapest-list selection). No new runtime feature toggle or environment variable is introduced.

The actual node RPC adapter executes native transaction simulation on independent state copies with candidate access lists. It records state accesses independently of EVM snapshot rollback, sorts/deduplicates results and repeats until the returned list and metered execution agree. The run is bounded to 64 attempts and reports nonconvergence explicitly. Sender, recipient/creation target, warmed coinbase and active precompiles are excluded when they have no storage keys. Storage entries remain even for a prewarmed address. The exact cross-client exclusion policy still needs the selected API fixture qualification.

Production tests use eiptest's HTTP server over node/backend.RPCBackend, not a mocked discovery result:

- Storage read/write access discovery and stable key order; identical repeated output.
- Reverted child accesses survive rollback and reach the caller's response.
- Historical block 0 and current state produce different accessed slots after a committed write.
- Contract creation discovers its constructor's storage, without consuming the canonical sender nonce.
- Generation leaves canonical storage unchanged.
- Invalid hex/address/storage-key lengths and overflowing/malformed quantities return -32602.

The initial test failed with accessList=[] on the old placeholder. It passes after implementation. Mock backends implement only the explicit discovery capability for API-shape tests; a backend without capability returns an error instead of claiming an empty access result.

Passed from pkg:

```sh
go test -count=1 ./rpc/... ./node/... ./eiptest
go test -race -count=1 ./node/backend ./eiptest -run 'TestRPCActualAccessListDiscovery|TestRPCAccessList'
go build ./...
go vet ./rpc/... ./node/backend
```

The race command runs the new HTTP scenarios in eiptest; node/backend has no tests matching that filter. Gas uses the current native transaction model. This is not conformance of the still-pending final 8037/API gas model, expanded authorization/blob transaction arguments, all external exact-match fixtures or independent-client behavior. Those remain explicit program gates, not silent passes.
