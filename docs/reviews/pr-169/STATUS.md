# Hegota Group B status

The fifth independent review reproduced no additional implementation defect in the selected native routes. It closed a missing acceptance consumer across memory, persisted/reopened Trie and persisted/reopened Snap state: exact cold/warm gas, nested transient rollback, inter-Frame clearing, transaction-original storage after commit/reopen and identical canonical state/receipt/BAL commitments. The [fifth audit](FIFTH_AUDIT.md) maps all 12 tasks to their review outcomes. Earlier implementation corrections and original-source controls remain in the [fourth](FOURTH_AUDIT.md), [third](THIRD_AUDIT.md) and [second](SECOND_AUDIT.md) audits.

All 12 selected Group B tasks map to 36 criteria and 79 mandatory native test identities, with no required test skipped. The register is [`specs/hegota/group-b.json`](../../../specs/hegota/group-b.json), with source-bound qualification in [`evidence/hegota-b/verification.json`](../../../evidence/hegota-b/verification.json). Removing the new backend consumer from any of four affected criteria, or removing the EIP-1153 normative source, fails checker regression tests.

Local qualification at test/register revision `35c95f6ea19889e8d3e5be7f4a05bfc08f058e3a`; production Go implementation is unchanged from `eeddcc62`:

- Full Go: 23,765 named test/subtest passes, 0 failures, 124 explicit skips; all 240 packages completed.
- Focused race: 8,005 passes, 0 failures, 74 explicit skips; all 36 packages completed. All 79 mandatory B tests passed.
- Three primary native backends agree on both committed roots and canonical receipt/BAL bytes for two signed transactions, including reopening Trie/Snap between transactions. Independent gas/transient values are specified in the test and audit.
- The earlier reconstruction control was rerun against current test sources: restoring only the old BAL wrapper from `2a092e43` makes the three earlier regression parents fail, with 15 failed test/subtest events. Current sources pass. This is not presented as a new fifth-audit production finding.
- Build/vet and native/wrapper CLI builds passed. Historical native/Geth differential baseline: 12/12 passed, no skips.
- Python: 75 tests passed in normal and optimized modes; base/isolated codec and semantic oracles passed.
- A requalified: 7 tasks / 26 criteria and all 62 mandatory native test identities. Full/race skips remain individually inventoried.

No production execution logic, public transaction/receipt wire format, storage schema, activation switch or migration is changed in this round. Prior consensus corrections retain their documented research-history compatibility boundary. Witness-only stateless execution is not admitted as a node backend by this review.

EIP-8250 D01 remains open: its three-fee example conflicts with its nonce-only replacement instruction and current 8141 two-field total budget. HGT-017 completes the development plan's isolated state/vector allowance, with no public composed decoder, activation, pool/import path or database migration. EIP-8272 uses the exact ordinary contract. Public-pool extensions remain Group C.

[PR #169](https://github.com/alt-research/eth2030/pull/169) is the publication target. Previous head `eeddcc62` passed 10 checks before the new backend consumer was added; those results do not qualify the new test/register revision. All 10 triggered checks passed on updated test/register head `b649750d` ([main CI](https://github.com/alt-research/eth2030/actions/runs/45744370015), [snap2](https://github.com/alt-research/eth2030/actions/runs/45744370014)). The evidence pins this tested revision; final documentation-only descendant checks are reported separately on PR #169.
