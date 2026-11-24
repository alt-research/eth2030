# Native RLP framing and length bounds

- [x] Reproduce acceptance of trailing block bytes and overflow of declared RLP lengths without allocating large payloads.
- [x] Require exactly one complete encoded block and bound lengths before integer addition.
- [x] Run negative regressions, affected/full decoder tests, race and both builds.
- [ ] Submit independent master-target PR; verify exact-head CI and cumulative compatibility.
