# EIP-7843 slot QUANTITY interoperability

- [x] Compare EIP-7843 and the Engine QUANTITY rules.
- [x] Reproduce valid uppercase hex rejection in memory and actual native Engine import.
- [x] Accept uppercase hex digits while retaining prefix, minimal encoding and uint64 bounds.
- [x] Verify persisted header/hash, RPC SLOTNUM and restart, plus affected race/build checks.
- [ ] Submit an independent master-target PR and verify CI/conflicts.
