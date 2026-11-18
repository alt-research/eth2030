# Glamsterdam slot commitment and independent header verification

[PR 74](https://github.com/alt-research/eth2030/pull/74), branch `feat/glamsterdam-slot-commitment`, based on `origin/master` `38ea9259`.

The requirements follow [EIP-7843](https://github.com/ethereum/EIPs/blob/3db38a3725105cb67b8804ac3e147967113bc3ca/EIPS/eip-7843.md) and [EIP-7928](https://github.com/ethereum/EIPs/blob/3db38a3725105cb67b8804ac3e147967113bc3ca/EIPS/eip-7928.md). Reference encoder/decoder: the existing `github.com/ethereum/go-ethereum v1.17.10` module, with its independent Header type, generated RLP encoder and block-hash implementation.

## Findings and changes

The canonical native header codec already encodes BAL and slot correctly; no replacement codec is introduced. Independent decoding/re-encoding now verifies BAL/slot fields and block hashes for slot zero, 42 and uint64 maximum. Independently changing either commitment changes the block hash.

Production node Engine FCU and newPayload treated a supplied slot zero as absence. Their shared build call now uses an explicit optional slot pointer and V4 preserves zero; V3 continues supplying no slot. Import retains an explicitly encoded zero. The memory Engine V4 builder and V5 importer also omitted slots; both now populate the actual header, matching their payload reports. The standalone V4 attribute validator no longer rejects zero. CL assignment and chain-level input/schema qualification remain separate from accepting the uint64 domain.

## Actual consumer evidence

The new HTTP test drives a real node-backed builder and independent native receiver from a common genesis. For slots 0, 42 and uint64 maximum it submits a real signed transaction executing SLOTNUM, reads back the stored value, decodes the resulting header with go-ethereum, compares its hash to the Engine payload hash, and checks slot/hash in public block RPC. ByHashV2 and ByRangeV2 historical bodies and the built payload's BAL DATA must hash to that independently decoded header's BAL commitment. Slots are deliberately unrelated to timestamp and block number.

A separate test builds and imports actual memory-Engine payloads, checks independent header hashes and reads back the accepted slot. Restoring the three production files to `38ea9259` makes the HTTP zero-slot assertion and every memory-backend slot assertion fail. The local controlled baseline log is `/tmp/glamsterdam-slot-baseline-failures.log`; production sources were restored before passing verification.

Validation:

```sh
cd pkg
go test -count=1 ./core/types ./eiptest ./engine/... ./node/...
go test -race -count=1 ./engine ./eiptest -run 'TestMemoryEngineSlotCommitment|TestEngineSlotAndBALIndependentReadback'
go vet ./core/types ./engine/... ./node/backend ./eiptest
go build ./...
```

These checks pass. This is independent codec/hash qualification plus two native execution peers, not full independent-client execution or CL gossip qualification. Missing-field/schema permutations, all asynchronous/custom builder paths and final gas-model parity remain release tasks. Historical-body persistence/restart qualification also uses the existing program evidence; this test does not restart a process.
