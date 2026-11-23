# FileDB directory capacity repair

- [x] Bind actual ENOSPC to target directory inode and kernel htree-level limit; verify free blocks/inodes and preserve source.
- [ ] Share per-operation/path helpers with existing persistence/metrics PRs to avoid merge conflicts.
- [ ] Add default sharded storage with legacy reads, correct overwrite/delete/restart and crash recovery.
- [ ] Protect old-binary rollback before any new-format writes; preserve committed WAL through interrupted upgrade.
- [ ] Exercise legacy/mixed data, failures, directory distribution, full builds/race checks and actual source bytes.
- [ ] Submit independent master PR; qualify cumulative runtime/BLS and update progress without historical overclaims.
