# EIP-7495 completion checklist

- [x] Read current EIP-7495 and EIP-7916; preserve original dirty checkout and read-only refs.
- [x] Retain executable pre-fix gindex, illegal-schema and ineffective-test observations.
- [x] Add immutable validated ProgressiveContainer schemas and reuse canonical Container codecs.
- [x] Generate and verify field proofs; repair shared progressive-list gindex prefixing.
- [x] Use checked root construction in existing execution SSZ consumers; preserve their reference roots/bytes.
- [x] Generate independent roots/bytes/proofs and invalid-schema vectors, including boundary and nested layouts.
- [x] Qualify captured actual CL progressive state against the new Go API and existing CL consumers.
- [x] Require meaningful rejection tests and executed Merkle/index/layout mutations.
- [ ] Run affected/full Go tests, race, builds, required CI and site progress guards.
- [ ] Update the implementation matrix, current roadmap and homepage only after acceptance.
- [ ] Submit an independent master-target PR, relocate tracking to pr-number and check final CI/conflicts.

No EIP-8025 adoption, public activation parameter, 48h/scale verdict or historical original-04 root-cause closure is implied.
