# Ordinary transaction context isolation

- [x] Trace actual Uniswap sequence gas divergence to transaction-local refund/transient lifetime using EIP-2200/3529/1153.
- [x] Reproduce consecutive signed transactions without a state commit and compare independent roots, receipts and persistent effects.
- [x] Reset only transaction-local context at the common message boundary; retain child-frame state-gas/refund rollback semantics.
- [x] Run affected/full/integration/race/vet/build and complete application replay, then publish an independent production correction.
