# Hegota Group B status

The fourth independent clause audit found and fixed an account-reconstruction BAL omission: prefunded CREATE/CREATE2 destinations captured balance zero after account reset instead of their transaction-original balance. This could publish false net-zero balance changes after success, constructor revert, enclosing Frame failure or atomic-batch rollback. The [fourth audit](FOURTH_AUDIT.md) records normative clauses, all 12 task outcomes and original-source reproduction. Earlier [third](THIRD_AUDIT.md) and [second](SECOND_AUDIT.md) audits retain their own findings and evidence; green predecessor CI did not prove absence of this omission.

All 12 selected Group B tasks now map to 36 criteria and 78 mandatory native test identities, with no required test skipped. The register is [`specs/hegota/group-b.json`](../../../specs/hegota/group-b.json), with current-source qualification in [`evidence/hegota-b/verification.json`](../../../evidence/hegota-b/verification.json). The checker requires the new lifecycle, rollback and real build/import consumers.

Local qualification at functional revision `52a84ba38b62ae6aea35a9815143d75a5d04773f`:

- Full Go: 23,761 named test/subtest passes, 0 failures, 124 explicit skips; all 240 packages completed.
- Focused race: 8,001 passes, 0 failures, 74 explicit skips; all 36 packages completed. All 78 required B tests passed.
- Restoring only the old BAL wrapper from `2a092e43` causes all three new parent regressions to fail, with 15 failed test/subtest events across execution and block packages. Corrected sources pass.
- Eight prefunded CREATE/CREATE2 call bodies agree with unchanged upstream Geth v1.17.10 Amsterdam state/gas. Earlier delegated-call and RETURN memory comparisons also pass in the full suite. These do not claim independent Frame-envelope or BAL-producer equivalence.
- Build/vet and native/wrapper CLI builds passed. Historical native/Geth differential baseline: 12/12 passed, no skips.
- Python: 75 tests passed in normal and optimized modes; base/isolated codec and semantic oracles passed.
- A requalified on current shared VM sources: 7 tasks / 26 criteria and all 62 mandatory native test identities. Full/race skips remain individually inventoried.

Public transaction/receipt wire and storage schemas are unchanged. Research history containing incorrectly computed Frame gas/BAL commitments may require rebuilding from a valid checkpoint; roots are not rewritten. No public-chain database migration is performed.

EIP-8250 D01 remains open: its three-fee example conflicts with its nonce-only replacement instruction and current 8141 two-field total budget. HGT-017 is verified under the development plan's isolated state/vector allowance, with no public composed decoder, activation switch, pool/import path or database migration. EIP-8272 uses the exact ordinary contract. Public-pool extensions remain Group C.

[PR #169](https://github.com/alt-research/eth2030/pull/169) remains the publication target. Previous head `2a092e43` passed 10 checks but contains the reproduced reconstruction defect. All 10 triggered checks passed on corrected source head `afe87e20` ([main CI](https://github.com/alt-research/eth2030/actions/runs/45741616254), [snap2](https://github.com/alt-research/eth2030/actions/runs/45741616252)). The evidence records that immutable tested head; final documentation-only descendant checks are reported separately on PR #169.

## Fifth audit in progress

A new independent review is checking every Group B task, including native state backends, context and serialization boundaries beyond the previous regression register. Findings and current-source qualification will be recorded before completion.
