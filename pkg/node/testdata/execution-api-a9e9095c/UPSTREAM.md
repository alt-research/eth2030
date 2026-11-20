# Official execution-API inputs

Source: ethereum/execution-apis at a9e9095c453d7b3c95b89abfcf46afb2eff4e1b7. The four method folders, genesis and chain bytes are unchanged upstream CC0 inputs. There are 107 files and 108 requests. Authenticated state snapshots are independently reconstructed from that exact chain through the unchanged pinned Geth SDK and fully rehashed through native snapshot validation. These supplied fixture pre-states qualify native HTTP/RPC/EVM consumers; they do not claim native historical PoW import.

Reproduction from `pkg`: `go run ./node/testdata/execution-api-a9e9095c/prepare.go -source ./node/testdata/execution-api-a9e9095c -output /path/to/new/snapshots`. Existing outputs are rejected. The SDK uses the same historical fake-seal fixture policy as the upstream chain generator; native fixture tests authenticate complete supplied state roots and exercise real HTTP/RPC/native EVM, while genuine native Node admission/import coverage remains separate.
