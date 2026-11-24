# Development checklist

- [x] Read and pin current phase0/Gloas FCR, safe-block and Engine contracts; retain existing frozen progress boundaries.
- [x] Reproduce mixed-root vote counting; repair root isolation, invalid/duplicate votes and expiration without inventing a second consensus authority.
- [x] Trace real Lighthouse BLS/fork-choice FCR through authenticated Engine safeBlockHash to native EL safe RPC and persisted recovery.
- [ ] Exercise actual signed CL, competing branches, same-head safe changes, optimistic/invalid exclusion, rollback and restart consumers.
- [ ] Run affected unit/race/integration checks and builds; submit a standalone master-target PR and check exact-head CI.

48h/larger-scale acceptance and upstream Lodestar approval remain user-deferred. Original historical waiter attribution remains unknown. No public fork parameter changes or EL master merge are authorized.
