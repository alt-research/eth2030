# Historical EIP-7702 authorization conformance (R01)

Review: [PR 88](https://github.com/alt-research/eth2030/pull/88).

Native pre-Amsterdam authorization handling now warms a recovered authority before checking its code and nonce, rejects the reserved maximum nonce before recovery, skips authorities with ordinary code, and clears delegation for a zero target. The working Amsterdam authorization executor is unchanged. This follows EIP-7702 steps 1–9, including the order of skipped-tuple side effects.

The signed skipped-nonce regression previously charged 48,605 gas against Geth's 46,105. The actual public pool → Engine HTTP producer → second native Node importer now charges 46,105 and matches the full independently executed Geth v1.17.10 state root `0xe140e22463021b1009a0eec15a73fdadd640ab0dafa586fe0e349abf0ececd0a`, including standard history/beacon/withdrawal/consolidation predeploy execution. The authority nonce/code remain unchanged. Helper regressions cover wrong nonce, ordinary code, maximum nonce, chain/signature rejection, duplicate tuples, and clear delegation; these exposed failures before the correction.

Validation: full `go test ./...` (212 passing packages), `go test -tags integration ./core/...` (26 passing packages), race tests for `core/eips`, `core/execution`, `cmd/glamsterdam-compare`, `node`, full `go vet ./...`, and `go build ./...`. The unchanged independent execution corpus matches 42/42 Amsterdam cases; all ten Osaka authorization cases also match. Six of the other Osaka direct-transition controls still differ because they exceed the historical transaction cap; these remain R02/R04 outer-consumer coverage, not an assertion of full Osaka conformance.

The compact signed evidence below retains both fork inputs/results for the ten authorization scenarios. Its dirty flag records capture before the source commit. This review closes the concrete R01 transition bug; official fixtures, deployed applications, independent full-process clients and final protocol/release selection remain separate obligations.

Normative input: [EIP-7702](https://github.com/ethereum/EIPs/blob/3db38a3725105cb67b8804ac3e147967113bc3ca/EIPS/eip-7702.md). Selected LIFO gas policy remains unchanged.
