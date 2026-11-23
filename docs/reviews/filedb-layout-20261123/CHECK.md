# FileDB directory capacity repair

- [x] Bind actual ENOSPC to target directory inode and kernel htree-level limit; verify free blocks/inodes and preserve source.
- [x] Share per-operation/path helpers with existing persistence/metrics PRs to avoid merge conflicts.
- [x] Add default sharded storage with legacy reads, correct overwrite/delete/restart and crash recovery.
- [x] Protect old-binary rollback before any new-format writes; preserve committed WAL through interrupted upgrade.
- [x] Exercise legacy/mixed data, failures, directory distribution, full builds/race checks and actual source bytes.
- [x] Submit independent master PR; qualify cumulative runtime/BLS and update progress without historical overclaims.
