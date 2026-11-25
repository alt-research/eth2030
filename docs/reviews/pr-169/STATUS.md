# Hegota Group B status

All 12 Group B tasks are verified within the selected execution profiles: 36 criteria and 65 mandatory native test identities, with no required test skipped. The clause/file/test register is [`specs/hegota/group-b.json`](../../../specs/hegota/group-b.json), and the current-source result is [`evidence/hegota-b/verification.json`](../../../evidence/hegota-b/verification.json). See [the specification review](SPEC_REVIEW.md) for implementation and scope details.

The work reuses the canonical seven-field transaction and shared native VM/executor/build/import/BAL/persistence paths. It adds actual isolated keyed-nonce execution, canonical key/signing/pricing vectors, protocol nonce state/BAL and introspection, ordinary recent-root contract helpers and native deployment/slot/window/overwrite/reorg tests. APPROVE nonce overflow exceptionally halts; failed canonical frame receipts reject surviving state gas/logs. New matrices cover signatures, default accounts, introspection, scopes and corrupted block commitments.

EIP-8250's nonce-only delta profile remains isolated: D01 is open because the current 8250 three-fee example conflicts with its preservation instruction and current 8141 two-fee total budget. There is no public decoder, activation switch, pool/import route or public database migration for that profile. HGT-017 is verified under the plan's controlled-profile allowance. EIP-8272 executes the verbatim ordinary contract; Group C public-pool policy is separate.

Local qualification at functional revision `24d2553a`:

- Full Go: 23,692 named test/subtest passes, 0 failures, 124 skips; 240 completed packages (221 pass, 19 skip/no-tests).
- Focused race: 7,932 named passes, 0 failures, 74 skips; 36 completed packages (34 pass, 2 skip/no-tests).
- All 65 mandatory Group B native test identities passed, including real keyless deployment, CREATE, nonce rollback/retry, P256/default-code and native canonical fork-switch cases.
- Build and vet passed. Both native and wrapper CLI binaries were built and identified. The historical native/Geth baseline passed all 12 cases with no skips.
- Python regression suite: 73 tests pass in normal and optimized modes. Base/isolated codec and semantic oracles pass. Wrong budget, consumed nonce and root-age expectations each cause the native consumer to fail as intended.
- A was requalified: 7 tasks / 26 criteria and all 62 required test identities pass with current shared-runtime hashes. Full and race skip inventories remain explicit in `evidence/hegota-a/skipped-tests.json`.

The first full run exposed an existing checkpoint ETA test clock-tick flake. The test now explicitly establishes elapsed time; a fresh complete run passed. No production checkpoint behavior changed.

Publication and exact-head hosted CI checks are pending. No public composed Hegota release, independent-client Frames equivalence or Group C–J completion is claimed.
