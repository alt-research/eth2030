# Adopted fast-confirmation contract

Authority: consensus-specs d2021d2719235f0df4726a1d330502cb15aa8ea3, phase0/fast-confirmation.md and gloas/fast-confirmation.md; the standard Engine forkchoice contract. The full source was read before implementation.

Production confirmation belongs to the CL: authenticated BLS attestations, committee assignments, active unslashed balances, competing forks, optimistic exclusion, FFG conditions and restart rules. Lighthouse supplies its FCR result through Engine safeBlockHash. In Gloas the safe execution hash is the parent payload hash of the confirmed beacon block. The native EL validates head ancestry and exposes the selected block through ordinary safe-tag RPC. It can accept same-head confidence advances/retractions; finalized remains monotonic. EL startup conservatively resets safe to finalized until a fresh CL update, matching Geth.

The older Go FastConfirmTracker is a preauthenticated, single-candidate utility, not a second Ethereum consensus algorithm. It now owns its configuration, rejects out-of-range indices before allocating state, and rejects foreign-root votes before consuming a vote or incrementing support. No locally supplied index/registry is promoted into production validator authority.

The native HTTP regression builds/imports four real payloads, advances/retracts safe under the same head, rejects an unknown checkpoint atomically and verifies conservative restart plus authoritative reannouncement. The separate companion FCR durability PR persists announcements with compatible fork choice, with real signed restart/failure controls.

Process qualification uses a fresh two-node, 256-validator private Gloas tuple at six-second slots. prepare writes a standalone JSON args file reusing canonical genesis contracts. observe checks actual FCR SSE roots against the Gloas parent bid and native safe RPC, abrupt primary-CL restart, peer convergence and optional real proving with independent follower verification. reference_fast_confirmation_window.py uses unmodified d2021d27 reference code and real BLS for every signed category in its declared window. Failed windows require new output directories; no passing replay is substituted for an original incident attribution.

This is an out-of-protocol confirmation service under the specification's synchrony/Byzantine assumptions. It is not finality and does not create a public activation schedule, an actual 48h/scale pass or external maintainer approval. The original-04 waiter remains unknown.
