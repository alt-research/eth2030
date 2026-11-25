# Restored snap progress

A snap-configured node can reopen authenticated non-genesis execution state while its next CL-nominated finalized hash is still unknown. Both node pivot callbacks then decline bootstrap. The syncer nevertheless previously disabled its retained ancestry cursor merely because a pivot callback existed. Advancing targets and bounded-cache eviction could repeatedly discard completed ancestry work.

The worker now uses the existing published CurrentBlock capability to distinguish empty-chain bootstrap from continuation. Existing state follows the complete linked ancestry and ordinary execution-validation path. Genesis bootstrap still observes the exact nominated finalized hash, retains the existing guarded adoption and does not gain validation authority from downloaded bytes. EIP-3675 fork-choice events remain the sole publication authority; no head/finality/hash/root rule changes. The complete local EIP-3675 was read; Engine remains on the selected68d76a5 contract.

The deterministic restored-chain regression is red on the original production bytes and green with the fix, including cache eviction and eight target replacements. A separate positive control requires the nominated genesis pivot callback. Twenty repeats, complete node/beacon tests, affected race, vet and native build pass. Earlier inadequate fixture attempts are retained separately and are not counted as behavioral evidence. Exact-head full-suite checks and actual resumed-network qualification are recorded in the follow-up evidence.

The original bcf9 preflight stayed execution-optimistic with published EL head54573+64 while its CL advanced. It was interrupted for this source correction, not accepted as a soak or declared to have exceeded its3600s budget. Its original setup error, logs and clean final stop remain retained. This source correction does not retrospectively qualify that run.

This branch composes the independent Builder/startup/FileDB prerequisites and targets master. Merge126/127/128/129 first with ancestry preserved; this review contributes only sync/beacon progress logic, regressions and its own evidence. No EL merge or public release is performed.

Exact production/test source2c0df12ff4c89d6407c5f27c8de8ea56517d7a4e passes219 fresh packages, full vet and native build (EXACT_CHECKS.json). The Go1.26.11 final soak executable SHA256 is4e0f7a54765ce0992d0477d65565fcb41c27a8bac6525bb6f2497721939a9043. Only this new tuple is used by the corrected independent observer; catchup and actual48h acceptance remain pending.
