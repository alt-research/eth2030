# Development checklist

- [x] Read and pin current phase0/Gloas FCR, safe-block and Engine contracts; retain existing frozen progress boundaries.
- [x] Reproduce mixed-root vote counting; repair root isolation and verify invalid/duplicate-vote and expiration behavior without inventing a second consensus authority.
- [x] Trace real Lighthouse BLS/fork-choice FCR through authenticated Engine safeBlockHash to native EL safe RPC and persisted recovery.
- [x] Exercise actual signed CL, a captured unconfirmed-block reorg and crash recovery; cover same-head safe changes, invalid checkpoint rejection and rollback through native HTTP. Retain source/CL-unit scope for optimistic exclusion rather than claiming a live injected optimistic fault.
- [x] Run affected unit/race/integration checks and builds; submit standalone master-target PR #155. Functional head 92392f25 passes hosted CI 45343841108. Final documentation-head checks are authoritative on the PR at delivery.

48h/larger-scale acceptance and upstream Lodestar approval remain user-deferred. Original historical waiter attribution remains unknown. No public fork parameter changes or EL master merge are authorized.
