# Exact historical state readback after cache eviction

- [x] Read EIP-1898/EIP-1186 and trace the real retained-header historical failure after independent archival replay.
- [x] Reproduce cold historical canonical and orphan state reads, forged same-root headers, complete proofs and restart behavior through actual Node consumers.
- [x] Resolve every native retained header by its exact validated block and reuse existing authenticated state replay/checkpoints.
- [x] Run affected/full/integration/race/vet/build and meaningful controls; publish this independent correction.
- [ ] Compose and roll the qualified source into the owned live nodes, then resume the original exact historical application sequence.
