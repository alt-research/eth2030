# Actual FCU durable-I/O wait fix

- [x] Read the spontaneous old-source failure stacks and retain all failed-run evidence without reclassifying controlled passes.
- [x] Verify the actual owner: ApplyForkchoice -> fileBatch.Write -> per-key fsync and later WAL fsync, retaining chain/database locks; newPayload and syncing wait behind those locks.
- [x] Read EIP-3675 and Engine Paris atomic forkchoice, validity and8s requirements.
- [x] Add a causal failing durable-publication regression with the actual FileDB/chain reader paths, including a cold reader and failure fencing.
- [x] Perform durable FileDB I/O before taking the chain publication lock; publish database/cache/chain state together only after successful persistence.
- [x] Allow committed cached reads during pending I/O without exposing partial writes or ignoring storage failure/close state.
- [x] Remove reader lock retention across database fallbacks and audit all affected mutation lock ordering.
- [x] Keep execution-sync progress reads independent from slow application while preserving target publication ordering.
- [x] Avoid rewriting unchanged forkchoice mappings/checkpoints; preserve durable WAL and state/hash validation.
- [x] Complete native build, targeted race and full regressions.
- [ ] Complete real mature-state/live/BLS revalidation; submit a separate master PR.

The spontaneous diagnosis failed before the original5.5h threshold and captured the real disk-wait chain. That establishes this reproduced failure mechanism. The earlier original04 has no stack and cannot be claimed independently proven identical from timing alone. Physical storage latency cannot be fixed by returning premature VALID or disabling fsync.
