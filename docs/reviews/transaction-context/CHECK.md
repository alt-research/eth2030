# Ordinary transaction context isolation

- [ ] Trace actual Uniswap sequence gas divergence to transaction-local refund/transient lifetime using EIP-2200/3529/1153.
- [ ] Reproduce consecutive signed transactions without a state commit and compare independent roots, receipts and persistent effects.
- [ ] Reset only transaction-local context at the common message boundary; retain child-frame state-gas/refund rollback semantics.
- [ ] Run affected/full/integration/race/vet/build and complete application replay, then publish an independent production correction.
