# Exact historical state readback after cache eviction

- [ ] Read EIP-1898/EIP-1186 and trace the real retained-header historical failure after independent archival replay.
- [ ] Reproduce cold historical canonical and orphan state reads, forged same-root headers, complete proofs and restart behavior through actual Node consumers.
- [ ] Resolve every native retained header by its exact validated block and reuse existing authenticated state replay/checkpoints.
- [ ] Run affected/full/integration/race/vet/build, meaningful controls and publish this independent correction.
- [ ] Compose and roll the qualified source into the owned live nodes, then resume the original exact historical application sequence.
