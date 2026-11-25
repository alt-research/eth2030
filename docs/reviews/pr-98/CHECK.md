# Real application compatibility qualification

- [x] Read pinned merged EIP-8037 LIFO, EIP-2780 and EIP-8038 behavior and preserve the selected model.
- [x] Pin independently published wallet, router/factory and token artifacts with licenses and archive integrity.
- [x] Execute actual signed deployment and multi-transaction application workflows through native and independent transitions; compare complete roots, logs, gas/payment, state effects and failure boundaries.
- [x] Exercise gas-limited delegation and failure handling with funded reservoir and nested rollback, retaining actual GAS metering limitations.
- [x] Qualify affected tests, race/vet/build and publish a dedicated application PR with reproducible evidence and remaining cross-process boundaries.


## PR 98 CI Engine projection correction (2026-11-20)

- [x] Inspect the exact failed native-node CI log and trace full-pool/staging/Engine consumers against selected EIP-8070 getBlobs semantics.
- [x] Reproduce sparse staging shadowing authenticated full cells with deterministic state and real KZG proof assertions.
- [x] Preserve available cells/proofs when overlapping full and partial entries refer to the same versioned hash; keep missing cells null and existing network/restart expectations unchanged.
- [x] Verify targeted red/green, ten repeated original native-node tests, affected race/full tests, vet and local build; publish only PR 98 and require remote-head CI verification before reporting completion.
