# Engine cell serving availability

- [x] Read current EIP-8070, Amsterdam Engine API and PR 12701 discussion.
- [x] Reproduce null-vs-missing-array and null-cell/proof violations through production backend.
- [x] Connect actual sync/shutdown/pool availability and preserve healthy positional responses.
- [x] Verify 128-hash support, sync/recovery, real native RPC/KZG path, race and native build.
- [x] Commit and submit master-target PR 151; initial head CI passed and expanded final head is rechecked on the PR.

- [x] Extend shape/readiness and Osaka gating consistently to V2/V3.
- [x] Reproduce and remove implicit legacy-proof upgrades; serve retained authenticated PeerDAS proofs.
- [x] Verify independent SDK V2/V3 decoding, 128-repeat ownership and all affected race suites.
