# Hegota Group B status

The third clause audit found and corrected two additional consensus-visible omissions: APPROVE incorrectly expanded memory for empty returns, and static SSTORE recorded a forbidden storage read before failing. The [third audit](THIRD_AUDIT.md) records exact normative clauses, original-source reproduction and all 12 task review outcomes. The [second audit](SECOND_AUDIT.md) retains the earlier four dispatch/BAL defect families and their separate evidence. Neither prior completion labels nor green CI are treated as proof of absence of omissions.

All 12 selected Group B tasks now map to 36 criteria and 75 mandatory native test identities, with no required test skipped. The register is [`specs/hegota/group-b.json`](../../../specs/hegota/group-b.json), with current-source evidence in [`evidence/hegota-b/verification.json`](../../../evidence/hegota-b/verification.json). Both new APPROVE consumers, the static SSTORE consumer and the EIP-214 source are mandatory in the checker.

Local qualification at functional revision `f68ba648ed0448650cf67c73df46d6b253ff9679`:

- Full Go: 23,746 named test/subtest passes, 0 failures, 124 explicit skips; all 240 packages completed.
- Focused race: 7,986 passes, 0 failures, 74 explicit skips; all 36 packages completed. No mandatory B test skipped.
- Restoring the two original VM files from `c9f847e0` causes all three new parent regressions to fail (nine failed test/subtest events); corrected sources pass.
- Ten APPROVE memory cases agree with unchanged upstream Geth v1.17.10 Amsterdam RETURN on gas, returned bytes and validity, excluding the Frame entry. This does not claim independent Frame-envelope equivalence.
- Build/vet and native/wrapper CLI builds passed. Historical native/Geth differential baseline: 12/12 passed, no skips.
- Python: 75 tests passed in normal and optimized modes; base/isolated codec and semantic oracles passed.
- A requalified on current shared VM sources: 7 tasks / 26 criteria and all 62 mandatory native test identities. Full/race skips remain individually inventoried.

Public transaction/receipt wire and storage schemas are unchanged. Research histories containing previously incorrect Frame gas or BAL commitments can require rebuilding from a valid checkpoint; roots are not rewritten. No public-chain migration is performed.

EIP-8250 D01 remains open: its three-fee example conflicts with its nonce-only replacement instruction and the current 8141 two-field total budget. HGT-017 is verified under the development plan's isolated state/vector allowance. No public composed decoder, activation, pool/import path or database migration is added. EIP-8272 uses the exact ordinary contract. Public-pool extensions remain Group C.

[PR #169](https://github.com/alt-research/eth2030/pull/169) is the publication target. The previous head `c9f847e0` passed 10 checks but contains the two newly reproduced defects. All 10 triggered checks passed on corrected source head `46f57ce4` ([main CI](https://github.com/alt-research/eth2030/actions/runs/45693637673), [snap2](https://github.com/alt-research/eth2030/actions/runs/45693637672)). The evidence records that immutable tested head; final documentation-only descendant checks are reported separately on PR #169.
