# PR 78 tracking cleanup

- [x] Preserve original tracking diff and changed sections from `9ad2d9d61bd05885bc7c6703d8df9b9a1228fec7`.
- [x] Synchronize shared trackers and merge current master without dropping feature behavior.
- [x] Verify merged source, local build and affected tests.
- [x] Prepare the existing PR branch for fast-forward publication; require live conflict-state verification before reporting completion.

## Preserved feature tracking

## Canonical fork schedule and Engine version boundaries (2026-11-18)

- [x] Read the current Engine Amsterdam/Osaka/Prague/Cancun rules and trace actual startup/HTTP consumers.
- [x] Reject incoherent canonical fork schedules before genesis/state adoption; preserve explicit private extensions.
- [x] Enforce canonical Engine version/required-field boundaries with exact activation and presence cases.
- [x] Qualify actual HTTP failures and valid forkchoice adoption independently from attribute rejection.
- [x] Run affected/full tests, build, race/vet and publish the isolated review.

