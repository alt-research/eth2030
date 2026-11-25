# Hegota Group B status

The second clause-level audit found and corrected four defect families missed by the first qualification: direct-precompile BAL entries, delegated-code gas/forwarding, implicit SSTORE reads on post-state failure and SELFDESTRUCT beneficiary access boundaries. The [second audit](SECOND_AUDIT.md) maps every Group B task to its review outcome and normative basis.

All 12 Group B tasks are verified within the selected execution profiles: 36 criteria and 72 mandatory native test identities, with no required test skipped. The current-source register is [`specs/hegota/group-b.json`](../../../specs/hegota/group-b.json), and its result is [`evidence/hegota-b/verification.json`](../../../evidence/hegota-b/verification.json). Prior qualification at `95d4be4e` contained these defects and is retained as predecessor evidence, not proof of their absence.

The correction reuses the existing delegated-code resolver before EIP-150 forwarding for all four CALL variants, preserves parent warming through child revert, and records exact BAL accesses only after their applicable pre-state sentries. Eight cold delegated call bodies agree with the unchanged upstream Geth v1.17.10 Amsterdam VM on gas and output. This is not an independent Frame-envelope oracle.

Local qualification at functional revision `157d9886`:

- Full Go: 23,729 named test/subtest passes, 0 failures, 124 explicit skips; all 240 packages completed.
- Focused race: 7,969 passes, 0 failures, 74 explicit skips; all 36 packages completed.
- All 72 mandatory B test identities passed. Restoring the four original VM files from `95d4be4e` causes all six new parent regression tests to fail (29 failed test/subtest events); current sources pass.
- Build/vet and both native/wrapper CLI builds passed. Historical native/Geth differential baseline: 12/12 passed, no skips.
- Python: 75 regression tests passed in normal and optimized modes; base/isolated codec and semantic oracles passed.
- A requalified on current shared VM sources: 7 tasks / 26 criteria and all 62 mandatory test identities. Full/race skip inventories remain explicit.

Public transaction/receipt wire and storage schemas are unchanged. Research histories containing the previously miscomputed Frame commitments can require rebuilding from a valid checkpoint; roots are not silently rewritten. No public-chain migration is performed.

EIP-8250 D01 remains open because its current three-fee example conflicts with its nonce-only replacement instruction and current 8141 total fee budget. HGT-017 is verified under the plan's isolated state/vector profile allowance; no public decoder, activation, pool/import path or public database migration is added. EIP-8272 uses the verbatim ordinary contract. Public-pool extension work remains Group C.

[PR #169](https://github.com/alt-research/eth2030/pull/169) remains the publication target. All 10 triggered checks passed on the corrected source head `0e633951` ([main CI](https://github.com/alt-research/eth2030/actions/runs/45689792178), [snap2](https://github.com/alt-research/eth2030/actions/runs/45689792174)). The committed evidence records that immutable tested revision; final documentation-only descendant checks are reported separately on PR #169. No public composed Hegota release, independent-client Frame equivalence or Group C–J completion is claimed.
