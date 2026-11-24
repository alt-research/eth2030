# EIP-7843 slot QUANTITY interoperability

- [x] Compare EIP-7843 and the Engine QUANTITY rules.
- [x] Reproduce valid uppercase hex rejection in memory and actual native Engine import.
- [x] Accept uppercase hex digits while retaining prefix, minimal encoding and uint64 bounds.
- [x] Verify persisted header/hash, RPC SLOTNUM and restart, plus affected race/build checks.
- [x] Submit independent master-target PR160.
- Exact-head hosted checks and mergeability are final delivery gates on PR160.

## Whole-upgrade review

- [x] Freeze official core/networking/informational scope and current EIP sources.
- [x] Review active EL/CL paths separately from obsolete experimental helpers.
- [x] Execute 220 Go package suites, 64 broad race suites and both complete selected native EELS corpora.
- [x] Execute CL types/state-processing/fork-choice tests and captured-state checks without fake crypto.
- [x] Independently compare slashed proposer selection, unchanged lookahead and churn/weak-subjectivity boundaries.
- [x] Retain the reproduced pre-fix native Engine rejection and meaningful invalid encodings.
- [x] Document all twenty EIPs and explicit remaining qualification/maintenance gaps.

- [x] Diagnose and retain the actual proof-CI installer rate-limit failure; supply the existing read-only workflow token without changing pins or acceptance.
