# Bounded FileDB residency and recovery

- [x] Identify full-database resident values and whole-WAL byte retention at open.
- [x] Reproduce residency growth and record existing byte/snapshot/failure semantics.
- [x] Preserve the exact data/WAL format while using a bounded value cache, streaming directory scanning and WAL intent offsets.
- [x] Keep committed-prefix recovery, atomic batches, read errors, empty values, iterator snapshots and failed-write fencing strict.
- [x] Measure declared profiles: 1000x4KiB and10000x64KiB values plus repeated-key WAL history; reopening budget300s and RSS ceiling1GiB per isolated profile. Fixture preparation is separate from open/recovery timing.
- [x] Pass rawdb/state/chain/node regressions/race, full suite/vet/build and cumulative PR merge checks.

Large metadata cardinality, full trie authentication and true multi-day soak remain separately measured boundaries; no shortened test is represented as a multi-day pass.

- [ ] Complete current-head CI, composed-source verification and review/merge-order publication.
