# Native transaction validation completion

- [x] Read EIP-2/155/2681/2718/4844/7702 constraints and trace actual pool, decoding and import signature consumers.
- [x] Add independent signed regressions for high-S/parity/chain width, mandatory recipients, nonce overflow and empty authorizations.
- [x] Fix validation by default without changing generic ecrecover high-S behavior or earlier-fork signature rules.
- [x] Prove actual Node admission and correctly bound Engine rejection, including unchanged head/state on failures.
- [x] Run affected/full/integration/race/vet/build checks, reconcile complete official fixtures and publish a dedicated PR.
