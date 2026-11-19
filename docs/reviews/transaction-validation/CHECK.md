# Native transaction validation completion

- [ ] Read EIP-2/155/2681/2718/4844/7702 constraints and trace actual pool, decoding and import signature consumers.
- [ ] Add independent signed regressions for high-S/parity/chain width, mandatory recipients, nonce overflow and empty authorizations.
- [ ] Fix validation by default without changing generic ecrecover high-S behavior or earlier-fork signature rules.
- [ ] Prove actual Node admission and correctly bound Engine rejection, including unchanged head/state on failures.
- [ ] Run affected/full/integration/race/vet/build checks, reconcile complete official fixtures and publish a dedicated PR.
