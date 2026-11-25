# Hash-selected RPC execution contexts

- [x] Read EIP-1898 and trace the actual HTTP call/estimate/access-list/trace state and header selectors.
- [x] Reproduce the real eth_call blockHash-object rejection, then cover canonical, orphan, unknown and historical selections with independent commitments.
- [x] Bind execution, ancestry and overrides to the selected exact header and enforce requireCanonical without selecting a different block.
- [x] Run affected/full/integration/race/vet/build checks; publish this independently reviewable correction.
- [ ] Roll the qualified candidate into owned live nodes and complete the original real application sequence with exact historical block queries.
