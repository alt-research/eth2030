# PR 70 tracking cleanup

- [x] Preserve original tracking diff and changed sections from `86a3322133a00d0ba36928decedc859299c9abc9`.
- [x] Synchronize shared trackers and merge current master without dropping feature behavior.
- [ ] Verify merged source, local build and affected tests.
- [ ] Publish and verify GitHub conflict state (reported separately after push).

## Preserved feature tracking

## Modern receipt bloom restoration

- [x] Reproduce nonzero-log consensus receipt roundtrip failure.
- [x] Recompute omitted wire blooms from decoded logs, not from an empty receipt bloom.
- [x] Verify codecs/consumers/build and prepare the scoped fix review.

