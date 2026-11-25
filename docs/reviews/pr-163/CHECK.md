# Native blockchain fixture conformance

- [x] Read official encoded-block fixture format, EIP-7928 BAL, EIP-7843 header, EIP-7685/EIP-8282 request and EIP-4895 execution contracts.
- [x] Import signed encoded blocks through the native Blockchain and compare independent header, BAL bytes, persisted BAL, receipts, full state and canonical head.
- [x] Preserve expected invalid cases; fail unknown exceptions, panics, missing metadata and unsupported networks instead of silently skipping.
- [x] Reproduce and fix six execution/header defects; retain seven unchanged official regressions and negative reference controls.
- [x] Execute six restored behavioral mutations; require assertion failures rather than compile failures.
- [x] Pass the full Go suite and both node builds.
- [x] Finish final affected race/vet checks and rerun both complete corpora from the committed executable.
- [ ] Publish an independent master-target PR and verify exact-head CI and cumulative merge compatibility.
