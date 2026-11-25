# Status

Active closure against master `5fedb7e5733fbef69767a711115ef9925e7bd769`. The original dirty checkout is preserved. Large implementations use independent branches. No EL master merge has been performed.

| Item | Current evidence | Remaining |
| --- | --- | --- |
| G01 | PR #163, clean functional executable `a73dc6a6225dfdc7e370f01bfba931b94f20feb4`: Osaka 17,441 cases/19,875 blocks and Amsterdam 26,349/32,745 pass, all final heads/full roots match. Six executed mutations caught. Nine exact-head CI checks pass at documentation head `bf7b24dc63cf5f9c5c5351ddfcb7179d75dae320`. | Merge/review only; not merged. |
| G02 | Complete current reference union has 12065 cases. All9344 SSZ,196 Gloas FCR,134 affected historical FCR,92 reward and645 gossip cases pass retained clean-source replays. Full composed qualification is now in Lighthouse PR23; real native canonical light-client defects are separately repaired in PR22. | Require all12195 full-node/compatibility receipts, broad affected tests, node build and latest hosted CI; four client/store-role vectors remain explicitly unrun. |
| G03 | Prior dated qualification retained as evidence, not reused as a new pass. | Fresh independent compatible mixed-peer fallback/scoring, custody/reconnect and BAL recovery qualification. |
| G04 | PR #161: obsolete compatibility helpers deprecated, source authority/pointers corrected, runtime token streams unchanged; all 10 CI checks pass at `209686f189cf8f9b716cd9e975f35bb69853c88b`. | Merge/review only; not merged. |
| G05 | Both detailed reference and unchanged client enforce the common prefix precondition before existing-builder top-up. Existing upstream EIPs #12572 explicitly proposes the same clarification. Local compatibility contract is documented with paired actual/reference receipts. | Upstream prose amendment is still open/unmerged; no unilateral consensus change. |

The new current-vector adapter exposed two additional Gloas runtime admission gaps: consensus-failed roots are treated as unknown, and proposer preferences cannot resolve a valid dependency hidden before finality by fork choice. Repairs are isolated in a Lighthouse runtime branch with real independently generated signed blocks, outer-signature poisoning controls and current-vector consumers. They remain in progress; adapter/setup failures are kept separate from client defects.

Independent RLP corrections are published in PR #162 (`99045706520bdc336b743d422e6ec55ac51015f9`, all nine CI checks pass). PR #160 remains ready, unmerged, all 14 checks passed at `0d47b9cb35a05c3c5251471b6ebeb3b06cfeebc2`. All 24 simulated merge orders of the exact #160–163 heads are conflict-free and produce one tree. Full composed Go tests are being run separately.

48h/larger-scale acceptance and Lodestar upstream approval remain excluded, not passed. Public activation is a network rollout decision. Original-04 / the 5.5-hour actual waiting-chain attribution remains UNKNOWN.
