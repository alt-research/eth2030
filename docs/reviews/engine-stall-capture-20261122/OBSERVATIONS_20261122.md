# Original04 follow-up observations

## Completed bounded observation

The new native source d1415df3 (history-lock fix plus profiling) and CL3782df1 completed1815.433s after readiness. Both daemons stopped with exit0, noOOM and unchanged2CPU/4096000000B limits. All60 ready samples agreed with independent Geth finalized hash/state root; the maximum ready-sample gap was31.171s. This is a bounded observation, not an actual48-hour result. During cold catch-up the native cgroup recorded824 memory.max events and a peak4096012288B; these are preserved, not described as absence of memory pressure.

The first independent real-BLS live capture and fresh offline replay authenticated sampled slots84217–84812. Inspecting actual stopped-node logs showed10 additional imported slots. An unchanged captured signed path already contained them, so a fresh frozen-reference replay over the explicitly extended forensic interval authenticated84217–84822:606 slots, no missing slots,616 parent links,512 signature participants. Its actual stopped beacon root is0x154efd02197aa2e894bc81dee3d615ba167a95e7a8324a38f91d5e7567e08190. No original exchanges, signatures, reference implementation or elapsed time were changed.

A separate helper read the stopped native database through a read-only key adapter, without opening FileDB, replaying WAL, acquiring a writable node or scanning/changing the database. Independent SDK decoding/rehashing of its persisted execution header agreed with reference Geth: block82187, slot84822, hash0x47001c326977167abcc49636cf9ed863f94965a44f476b3cdc608c69b2a6c5de, state root0x7fa8c86d6e0add77bb638780b1ba20041e61a2d07f054c0f0c516b1dd0a65952. Two account proofs and their returned fields were verified; changed balances were rejected. Frozen consensus code independently rehashed the complete beacon block body and bound its execution bid hash to that same native header. The resulting chain is real BLS finalized ancestry -> beacon body/root -> bid execution hash -> execution header -> state root -> account proofs.

This stopped-state audit supplements the actual online samples. It is not an invented online closing RPC or additional operating time. Exact runtime/helper identities and complete scope boundaries are in FAST_LIVE_RESULT.json. The read-only helper sources are retained as review text.

## Original incident remains open

The original04 source diagnostic was not restarted when replacing its flawed private early log follower. The replacement has verified actual PID1 hashes, native baseline stacks, bounded raw reads and optional post-timeout CL native backtraces. An independent scan of its real native log found zero Engine timeouts before handoff. Absence of an event after only part of the6h45m run does not resolve the original5.5-hour incident.

A separate original-source control is catching up before injecting exactly one epoch-boundary Engine503 after8s. It preserves real native validation, hashes and BLS. It tests recovery from a timeout, not the origin of the native delay. Its pending or future result must not be merged into this clean30min observation. A real original waiting chain and causal production regression remain required for root-cause closure.
