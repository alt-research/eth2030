# Simulation system deadline classification

- [x] Trace PR 90 CI failure through pre/post system-call timeout propagation and preserve actual deadline enforcement.
- [x] Reproduce wrapped context cancellation at system boundaries and distinguish genuine system errors.
- [x] Normalize cancellation before generic RPC error wrapping; preserve canonical state isolation.
- [x] Run affected/full/integration/race/vet/build checks, publish a dedicated fix and reconcile dependent CI.
