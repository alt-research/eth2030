# Status

G01 implementation is published in [PR #163](https://github.com/alt-research/eth2030/pull/163) against master `5fedb7e5733fbef69767a711115ef9925e7bd769`; final committed-source receipts pass; remote CI is being checked. This work does not close G02 (current CL vectors), G03 (independent live interoperability), or original-04 waiting-chain attribution.

The native runner imports 43,790 official cases containing 52,620 encoded blocks: Osaka 17,441/19,875 and Amsterdam 26,349/32,745. Both complete final-source runs pass at clean executable revision `a73dc6a6225dfdc7e370f01bfba931b94f20feb4`; see `VALIDATION.json`. Eight blocks per corpus have independently proven multiple invalid fields and reject at another permitted check; receipts flag them explicitly and do not assert that the declared signature branch ran.

The real import path found and fixed BAL bookkeeping addresses, zero-amount withdrawal account cleanup, zero-length EXTCODECOPY account access, uint256 BLOCKHASH truncation, erroneous EL rejection of opaque request output, and missing mandatory Amsterdam slot headers. Existing test constructors now supply explicit slot zero. The low-level block constructor retains zero as its omitted-slot default; Engine callers continue to validate and supply the actual CL slot, without deriving it from timestamps.

Seven unmodified official fixture subsets and their upstream source metadata are retained in `pkg/core/eftest/testdata`. The six executed Go-overlay mutations each produced an assertion failure in the intended native block-import regression. Header/post-state/receipt/BAL/unknown-exception negative controls remain fatal. `MUTATIONS.json` records the executed controls. Independent RLP framing and declared-length fixes are isolated in PR #162.

## Reproduction

Use Go 1.26.11 from `pkg`. Download the official [tests@v21.0.2 archive](https://github.com/ethereum/execution-specs/releases/tag/tests%40v21.0.2), require SHA-256 `e2fa7ea2ea516459f9ab92284b2927892f318f4e1f78ca802cb05a6c3b479ed9`, and extract `fixtures/blockchain_tests/for_osaka` and `for_amsterdam`.

Build with compiled source stamps (required on toolchains that omit Go VCS build settings):

```sh
revision=$(git rev-parse HEAD)
go build -buildvcs=true -ldflags "-X main.sourceRevision=$revision -X main.sourceModified=false" -o /tmp/native-blockchain-tests ./cmd/native-blockchain-tests
/tmp/native-blockchain-tests -fork Osaka /path/to/fixtures/blockchain_tests/for_osaka > osaka-blockchain.json
/tmp/native-blockchain-tests -fork Amsterdam /path/to/fixtures/blockchain_tests/for_amsterdam > amsterdam-blockchain.json
```

Only stamp `false` from a verified clean checkout. The executable records compiled identity, not the run directory's Git HEAD. CI requires exact case/block counts, zero errors/failures, unique cases, final canonical heads and full state roots. Unsupported fork transitions fail explicitly; they are outside these two selected exact-fork corpora.

## Normative sources

- [EIP-7928](https://github.com/ethereum/EIPs/blob/d27448e08f60513856bbe228ea7ce04f156209dc/EIPS/eip-7928.md)
- [EIP-7843](https://github.com/ethereum/EIPs/blob/d27448e08f60513856bbe228ea7ce04f156209dc/EIPS/eip-7843.md)
- [EIP-7685](https://github.com/ethereum/EIPs/blob/d27448e08f60513856bbe228ea7ce04f156209dc/EIPS/eip-7685.md)
- [EIP-8282](https://github.com/ethereum/EIPs/blob/d27448e08f60513856bbe228ea7ce04f156209dc/EIPS/eip-8282.md)
- [EIP-4895](https://github.com/ethereum/EIPs/blob/d27448e08f60513856bbe228ea7ce04f156209dc/EIPS/eip-4895.md)
