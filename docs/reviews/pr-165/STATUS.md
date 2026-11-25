# Glamsterdam conformance delivery status

All confirmed G01–G05 development repairs and their finite qualification are
complete. EL #160–165 target master; no EL merge is performed. Lighthouse #18–23 are
actually merged into eth2030, ending at `e021d1bb17c16579432aa402441c82ed375a93ef`. Each merge is guarded by
the qualified source SHA and successful latest CI, with only the defined
checkpoint-sync skip. Exact receipts are in `LIGHTHOUSE_DELIVERY.json`.

| Item | Completed result | Delivery evidence |
| --- | --- | --- |
| G01 | 43,790 native signed-block cases; 52,620 block checks, 45,221 accepted and 7,399 expected rejections; all final headers/state/BAL roots agree. Six executed behavioral mutations caught. | EL #160/#162/#163; EL_CODE_CI.json. |
| G02 | 12,195 strict current-source/full-node and historical compatibility cases pass locally and hosted, with 320 complete consumer jobs. Native admission, progressive bitmaps, canonical light-client cache/store and actual worker notifications are repaired. | Lighthouse #18/#19/#20/#22/#23; HOSTED_REFERENCE_QUALIFICATION.json. |
| G03 | Actual encrypted stock-Geth BAL recovery and Rust/JS peer fallback, reverse delivery, reconnect, metadata and scoring; delivered bytes pass real BLS/KZG consumers. | EL #164 and Lighthouse #21; LOCAL_CHECKPOINT.json. |
| G04 | Retired obsolete gas authority; runtime token streams unchanged; current merged LIFO preserved. Proposals #12581/#12582 are closed without merge. | EL #161; normative-source receipt in LOCAL_CHECKPOINT.json. |
| G05 | Detailed current consensus/client/reference agree on the common builder-prefix precondition, including top-ups; CL stake/registration authority is preserved. | Paired actual/reference probes in EL #160; prose clarification #12572 remains external, not a code blocker. |

The combined clean functional source c93c64e5a passes all eight local release stages,
including 403 beacon-chain / 4 client / 69 state-processing tests, both explicitly
executed external peer/consumer tests, 24 native network integration tests,
Clippy and a real-blst production node build. Feature-specific negative and recovery controls remain archived in their
owning reviews.

`DELIVERY.md` states the precise code/validation scope. Four light-client
client/store-role vectors are explicitly unrun outside full-node serving;
synthetic-signature and SSZ fixtures are separately labelled, not real BLS.
All current source identities and every uploaded case record are verified.

The final documentation publication gate rechecks its CI, all published EL
heads/targets and cumulative merge compatibility after this commit. Its outcome
is recorded in PR #165 metadata to avoid a self-referential commit hash.

Original-04 / the historic 5.5-hour actual wait-chain cause remains UNKNOWN.
48h/scale and Lodestar upstream approval remain excluded, not passed. Public
activation is a separate network decision; EIP-8025/full Ethereum STF guest
work stays later. Historical dated checklists are not new active-code gaps.
