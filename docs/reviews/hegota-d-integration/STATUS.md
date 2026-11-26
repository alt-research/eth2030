# Group D status

Development begins from merged PR #170 at master
`81e63f586642a1406d56db9fa7cbf00941feb7bf`. The working tree was clean and
the branch was created directly from the fetched master, without replaying the
merged Group C commits. Specification and duplication review is in progress.
The first real primitive is implemented: the pinned leanSPHINCS verifier passes
all 26 independent EthLib verification vectors mirrored by pinned leanVM,
including malformed inputs, key/message changes and tampering in every layer.
Valid signatures use exactly 515 Keccak-f permutations; all 7,740 truncations
are rejected before hashing. Full Group D integration/release qualification is
not claimed yet.

The latest direct-proof source and EIP-8288 clarification snapshot are pinned.
The [32-criterion plan](PLAN.md) records source/format conflicts and conditional
activation boundaries. Proof/dependency encoding, inherited base/keyed signing,
deduplication, gas and recognized-prefix handling now have independent vectors
and native constraint/position/work tests. Execution/pool/payload integration,
SETCODEFROM and ASSERT are still in progress.

SETCODEFROM now has native opcode, fixed execution pricing, paid-access/static/
system guards, raw source validation, execution-environment code adoption,
snapshot origins and creation completion support. BAL supports strict hash-only
adoption tuples and strictly-earlier availability resolution without a bytecode
fallback. Native tests cover invalid/warm/cold/same/large sources, CALL families,
7702, CREATE/CREATE2, ignored invalid initcode return, same-transaction rejection,
revert and immediate-aware scans. Full bal/vm/execution suites pass. Consumer
wiring, public proxy limits and broader compatibility qualification remain open.

ASSERT execution now restores the full body to the post-payment prefix snapshot,
preserves consumed execution/status, removes body state gas/logs and skips the
remaining assertions. TXTRACE and EVENTDATACOPY have all selectors, full-width
bounds, exact operand order and read-only subtree guards; the diff journal
captures pre-values before payment/nonce writes and builds sorted net tables
lazily. Native selector/copy/rollback/settlement/BAL tests and complete
core/types, core/vm and core/execution suites pass. Additional composition and
application compatibility cases remain part of final qualification.

The proof proposal is wired through bounded raw submission, exact wrappers,
stateless verification caching, public admission (including body dependencies),
one-wrapper lookup/accounting, builder collection, canonical union, header/body
commitments, both Engine backends, payload body queries, chain/raw database
codecs, restart and parallel processing. Native HTTP submission/simulation,
Engine build/import/body retrieval/restart and live peer wrapper exchange pass.
Missing/tampered/extra witnesses fail before state contamination. Ordinary eth
transaction messages reject dependency transactions; hgtp/1 is a separately
negotiated native experiment with a chain/pinned-profile handshake and bounded
announce/request/response messages, not an official eth message assignment.

The Gloas 19/20-field progressive payload has independent remerkleable vectors
for absent, active-empty and two witnesses, using pinned ethlambda's opaque
progressive witness lists. EIP-7807's different execution-block schema is not
silently mixed with this profile; its unspecified composition fails closed.
The full integrated package run and remaining compatibility/composition review
are still in progress; final completion/release qualification is not claimed.
