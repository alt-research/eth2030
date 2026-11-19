# PR 79 tracking cleanup

- [x] Preserve original tracking diff and changed sections from `901ec06331bd5290c733adf23c10d9e670dabd2c`.
- [x] Synchronize shared trackers and merge current master without dropping feature behavior.
- [ ] Verify merged source, local build and affected tests.
- [ ] Publish and verify GitHub conflict state (reported separately after push).

## Preserved feature tracking

## Persisted fork configuration and restart authority (2026-11-18)

- [x] Trace actual Node startup before history replay and reuse the existing persisted config key.
- [x] Detect incompatible active schedules and immutable parameter changes before state/metadata adoption.
- [x] Qualify future updates, either-side activation, missing/corrupt metadata and failed persistence.
- [x] Verify actual node restart preserves heads/state/configuration on rejection and does not mutate network globals.
- [x] Run affected/full tests, build and race/vet; publish a dedicated PR.

