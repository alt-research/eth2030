# Simulation system deadline classification

- [ ] Trace PR 90 CI failure through pre/post system-call timeout propagation and preserve actual deadline enforcement.
- [ ] Reproduce wrapped context cancellation at system boundaries and distinguish genuine system errors.
- [ ] Normalize cancellation before generic RPC error wrapping; preserve canonical state isolation.
- [ ] Run affected/full/integration/race/vet/build checks, publish a dedicated fix and reconcile dependent CI.
