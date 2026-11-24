# Development checklist

- [ ] Audit the canonical guest statement, all producer/verifier/caller paths and real proof backend choices against primary specifications.
- [ ] Reproduce zero/forged proof acceptance; replace hash-shaped placeholders with a real execution proof and trusted verifier.
- [ ] Bind program, input, output, resource bounds and guest machine state; reject forged receipts, mismatched statements and malformed encodings.
- [ ] Integrate actual node execute/prove/verify consumers with lifecycle, cancellation, limits and restart behavior.
- [ ] Run real local proof generation and independent verification plus tamper controls, affected suites and node builds.
- [ ] Submit a separate master-target PR with exact proof-system/source identities and honest validation boundaries.

48h/larger-scale acceptance and upstream Lodestar approval remain user-deferred. Original historical waiter attribution remains unknown. No public fork parameter changes or EL master merge are authorized.
