# Fixed historical acceptance corpus

`legacy-add11.json` and `cancun-add11.json` are unmodified Ethereum Foundation
legacytests files at commit `1f581b8ccdc4c63acf5f2c5c1b155c690c32a8eb`:

- `Constantinople/GeneralStateTests/stExample/add11.json` (7 cases)
- `Cancun/GeneralStateTests/stExample/add11.json` (5 cases)

Source URLs, SHA-256 and exact counts are in `manifest.json`. The upstream
legacytests repository dedicates its test data to the public domain under CC0:
https://github.com/ethereum/legacytests/blob/1f581b8ccdc4c63acf5f2c5c1b155c690c32a8eb/LICENSE

`rejected-upstream-zero-root.json` is the unmodified geth v1.17.1
`cmd/evm/testdata/statetest.json`, commit
`16783c167c4be5e6675fd8de0d1b762c88d6232f`. It is **excluded from acceptance**
because its root and logs oracles are zero. It demonstrates why discovery of an
upstream fixture alone is not sufficient evidence of conformance. Geth source:
https://github.com/ethereum/go-ethereum/blob/16783c167c4be5e6675fd8de0d1b762c88d6232f/cmd/evm/testdata/statetest.json
Geth license: GNU LGPL v3 or later for library code / GPL v3 or later for commands;
https://github.com/ethereum/go-ethereum/blob/16783c167c4be5e6675fd8de0d1b762c88d6232f/COPYING

Run from `pkg/`: `go run ./cmd/eth2030-baseline`. Missing inputs, changed counts,
unsupported fields/forks, zero roots/logs, wildcard exceptions and backend
mismatches return nonzero. This tiny corpus is not full historical-fork or
Prague/Glamsterdam/Hegota coverage. New required vectors need explicit manifest
and source review; the runner never writes its own expected results.
