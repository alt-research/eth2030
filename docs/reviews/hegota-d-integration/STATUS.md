# Group D status

Development begins from merged PR #170 at master
`81e63f586642a1406d56db9fa7cbf00941feb7bf`. The working tree was clean and
the branch was created directly from the fetched master, without replaying the
merged Group C commits. Specification and duplication review is in progress.
The first real primitive is implemented: the pinned leanSPHINCS verifier passes
all 26 independent EthLib verification vectors mirrored by pinned leanVM,
including malformed inputs, key/message changes and tampering in every layer.
Valid signatures use exactly 515 Keccak-f permutations; all 7,740 truncations
are rejected before hashing. Full Group D integration/release qualification is
not claimed yet.
