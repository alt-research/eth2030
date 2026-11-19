# Verified Merkle state history

- [x] Retain the actual 15-second historical getter failure and trace ordinary-chain fallback to replay from genesis after state-cache eviction.
- [ ] Persist immutable native MPT nodes and code by their actual cryptographic commitments, with complete root publication atomic with validated block data.
- [ ] Open an original historical root through the existing native trie/snapshot primitives, independently verify every loaded account/storage/code commitment and preserve exact header/branch context.
- [ ] Retain authenticated replay compatibility and provide a bounded, cancellable upgrade path for existing databases without modifying forkchoice authority.
- [ ] Test restart/orphans/eviction, corruption/missing nodes, interrupted writes and independent account/storage proofs; run affected/full/race/vet/build checks.
- [ ] Publish the separate review and rerun original actual historical consumers on both new and upgraded processes.
