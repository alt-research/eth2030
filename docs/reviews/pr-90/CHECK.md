## EIP-161 transaction-boundary empty-account finalization (2026-11-19)

- [x] Read EIP-161 and reproduce native roots retaining touched empty precompile accounts.
- [x] Implement reversible touched-account tracking and fork-aware finalization across native state consumers.
- [x] Preserve untouched empty accounts, pre-fork behavior, failed-call rollback, durable state and simulation behavior.
- [x] Verify independent signed/official roots, affected/full/integration/race/vet/build checks and publish a separate PR.

