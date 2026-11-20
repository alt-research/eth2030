# Master-target PR delivery audit (2026-11-20)

- [x] Freeze current master and all open/recently merged PR heads without touching the original dirty checkout.
- [x] Prepare a separate master-targeted follow-up containing only original PR 91 CLZ changes and review tracking; preserve original ancestry and verify affected tests/build.
- [x] Inspect every open PR target and recent merged heads for stranded delivery; preflight current-master and dependency-ordered merge conflicts.
- [x] Retarget open EL reviews directly to master, retain required dependency order and Draft status, and replace stale base-branch guidance in descriptions.
- [ ] Publish exact target/head/merge-state evidence and recheck the final remote heads, original dirty files and master; stop without merging or starting new feature work.

- [ ] Reconcile GitHub dirty flags with multiple-ancestor local merges: normally merge current master and updated prerequisites into affected review heads, preserving all original ancestry and exact predicted source trees; verify builds/affected tests and final GitHub mergeability.
