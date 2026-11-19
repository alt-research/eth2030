# RPC execution request cancellation

- [x] Trace real HTTP/single/batch dispatch and native simulation consumers before choosing an implementation.
- [ ] Preserve JSON-RPC/EIP-1474 wire fields and EIP-1898 exact block/state semantics; carry only local transport context.
- [ ] Propagate cancellation through actual native call/estimate/access-list/simulation/trace execution without copying shared API locks or weakening work/gas limits.
- [ ] Verify actual HTTP disconnect and batch cancellation, concurrent request isolation, pre-cancelled execution and unchanged committed state; require executed mutation failures.
- [ ] Run affected/full/race/integration/vet/build gates, publish the separate review and qualify the composed runtime before pressure acceptance.
