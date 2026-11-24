# Development checklist

- [x] Audit the canonical guest statement, all producer/verifier/caller paths and real proof backend choices against primary specifications.
- [x] Reproduce zero/forged proof acceptance; replace hash-shaped placeholders with a real execution proof and trusted verifier.
- [x] Bind program, input, output, resource bounds and guest machine state; reject forged receipts, mismatched statements and malformed encodings.
- [x] Integrate actual node execute/prove/verify consumers with lifecycle, cancellation, limits and restart behavior.
- [x] Run real local proof generation and independent verification plus tamper controls, affected suites and node builds.
- [x] Submit a separate master-target PR with exact proof-system/source identities and honest validation boundaries.

48h/larger-scale acceptance and upstream Lodestar approval remain user-deferred. Original historical waiter attribution remains unknown. No public fork parameter changes or EL master merge are authorized.

The complete final-image local acceptance passed all eleven required producer/verifier/native-node cases without skips. Initial hosted CPU proving exceeded the initial five-minute budget; the bounded fifteen-minute budget and independent-family CI matrix retain the same fixed-image, real-proof, failure and cancellation assertions. Exact final hosted status remains attached to PR #156 at delivery.
