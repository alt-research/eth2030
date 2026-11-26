# Latest Group C dependency and candidate-boundary audit

Reviewed on November 27, 2026 (Asia/Shanghai), starting from pushed source
`53904476515cbd8c6e324bea38a75c1557c173df`. This review covers HGT-020–029,
all forty criteria, and the normative dependencies consumed by their native
execution paths. The older qualification in RECHECK.md remains historical.

## Current authority

The EIP-8141/8250/8272/8151 file blobs still match the read-only references:
`9bc38a368881fa52e1a1a9970274e6511874d13f`,
`10bd1d84c9534c9db65f9ccbffcf8d5902b9c1ae`,
`0acc21ad8359be83c0929c7cc70e45326be8acff`, and
`bfd0db906f55033442a064357d8e6768490b8fd5`. This alone does not establish
that their dependencies are current. Expanded comparison found three merged
dependency corrections, all from November 26 UTC:

| Dependency | Immutable merged source | Consequence |
|---|---|---|
| EIP-8037 | [#12624](https://github.com/ethereum/EIPs/pull/12624), commit `514f00c369f9586e4d70a25da798ae2bd3d0ead7`, blob `43cb2b4fb8c72a66d42a06e21a75b70f0539ccbc` | Storage-only creation collisions apply via EIP-7610. Charge by account existence, then restore the failed child's account state charge. |
| EIP-8038 | [#12626](https://github.com/ethereum/EIPs/pull/12626), commit `b0c9c52b789d0fd7fcc44896456b43daa67b8ca5`, blob `2953df7781f6dd4ccbd03cc8eaea23a77fd17ba5` | SELFDESTRUCT's ACCOUNT_WRITE applies only to a positive transfer creating a dead beneficiary; numeric prices are unchanged. |
| EIP-2780 | [#12622](https://github.com/ethereum/EIPs/pull/12622), commit `5310a31af7458fa7d95a8f9ef4a185738495e034`, blob `5810f375698717ec92251f37505fba2d815bc684` | Explicitly removes origination-time delegation-target warming; names the EIP-7976 floor and EIP-7981 access-list data surcharge. |

Exact source snapshots are in `specs/hegota/upstream/2026-11-27/` with SHA-256
and immutable revision/blob identities in the C register. The read-only refs
and independent A/B oracle expectations are unchanged. Current 7610, 7976 and
7981 blobs also match refs (`5e96d303`, `15abf800`, `aaf7a767`).

The collision history matters: [#12296](https://github.com/ethereum/EIPs/pull/12296),
merged September 18, explicitly removed storage-only collisions. #12624 now
explicitly restores them. Its author review approves the correction, while its
PR description calls EIP-7610 Final. The current EIP-7610 front matter actually
still says **Last Call**. Its text also prescribes retroactive collision checks.
Neither a PR description nor this implementation establishes a finalized
mainnet activation decision.

The current discussion streams were fetched in full for their post-ID lists,
and their latest numbered posts were read: 8141 181/highest 187, 8250 5/6,
8272 3/3, 8151 32/33, 2780 10/10, 7610 23/23, 8037 50/53, 8038 25/25.
[7610 post 23](https://ethereum-magicians.org/t/eip-revert-creation-in-case-of-non-empty-storage/18452/23)
explicitly questions retroactivity versus hard-fork activation; it does not
resolve the front-matter discrepancy. The latest 8037 posts request changes
to SELFDESTRUCT refills/state-growth rationale, while merged text still denies
those refills. The 8038 warm-clear profitability report was withdrawn in post
25 after accounting for the earlier cold access. The 2780 thread's older PAY
and code/non-code cost proposals do not override the current merged schedule.
The 8151 cache/witness discussion still requires stateful code checks outside
pure signature recovery, as implemented in the selected native profile.

The selected native Hegota composition consumes the latest EIP-8037 collision
rule at the existing `HogotaTime`, including ordinary creation and canonical
Frames. Frames only exist at that fork. Frozen pre-Hegota profiles retain their
historical behavior for replay and independent comparison; this is **not** a
claim of universal retroactive EIP-7610 activation. No mainnet timestamp, new
transaction format, or new optional public policy is invented. This version
boundary is explicit in `group-c.json` and guarded by the checker.

Canonical-paymaster [#12041](https://github.com/ethereum/EIPs/pull/12041) remains
open/unmerged at `4a98fcefa5c6f02a25289bf55b9d92902d02b4c6`. Its exact local
proposal profile remains opt-in; canonical upstream adoption/fork pinning is
still the sole formal public release gate. No proposal review comment changes
the merged transaction rules.

## Confirmed omissions and corrections

1. Native CREATE/CREATE2 checked only nonce/code, so storage-only destinations
   lost existing storage and were deployed. The selected Hegota paths now read
   the complete current storage root. The old EIP-7610 helper's ten-slot probe
   also missed arbitrary keys; it now checks the root, including dirty writes.
   A VM adapter unable to establish the root fails closed. All production
   `core/state.StateDB` backends provide that capability.
2. Frame storage-only collisions must not bypass the conditional account charge.
   Empty-by-existence destinations are charged before collision failure; the
   failed child restores that charge through the same journal snapshot. A
   deficient charge halts the creator. A sufficiently funded creator catches
   the zero result, keeps its nonce increment and can spend the restored state
   budget. Balance-only destinations remain valid; prefunded storage still
   collides without a new-account charge. Amsterdam's reservoir path uses the
   same destination rule while retaining its existing charge/rollback machinery.
3. Public sender deployment consumed a whole-root fact without retaining that
   dependency. The journal now forwards the read through the read-set adapter;
   cached values are from original pre-state. Any sender storage-root change
   invalidates deployment acceptance, including unread hashed slots and removed
   branch state. Unrelated writes reuse the cached result. Candidate revalidation
   observes its own root without changing canonical state/cache/reservations.
4. The STF call probe duplicated production fork configuration and omitted
   Osaka/Amsterdam. It now calls `ConfigureBlockEVM`, including the selected
   creation rule, slot and conditional precompile configuration.

Root inspection records an account access for BAL, without fabricating ten
slot reads or writes. Public deployment still rejects a destination other than
the sender before reading its state. Existing depth/balance/nonce-overflow
prechecks retain their ordering ahead of destination access.

EIP-8038 and EIP-2780 clarification controls pass through the existing native
execution. Eight SELFDESTRUCT combinations distinguish cold/warm, dead/live
and positive/zero transfers. Delegation tests distinguish cold resolution from
explicit access-list warming, including the 1,280 gas data surcharge for one
20-byte address. No parameter or implementation change is needed for these
already conforming paths.

## Complete criterion reconciliation

| Criteria | Latest review result |
|---|---|
| C20.1–4 | Sixteen prefix forms, bounds, top-level code exceptions and traced/caught/transient restrictions remain current. Storage collision adds no prefix or public opcode permission. |
| C21.1–4 | Added whole-root deployment dependency and canonical/candidate invalidation; existing separate signer-class/helper-code reads, same-head refresh, reverse indices and scalar cleanup remain covered. |
| C22.1–4 | Recent roots retain exact runtime/tuples/slot-age checks and complete keyed replay identities. No timestamp-derived slot or manager state installation. |
| C23.1–4 | Signed maximum-fee exposure, payer classification, one reserved withdrawal, atomic admission/replacement and release remain covered. Failed storage-collision admission leaves no metadata. |
| C24.1–4 | Fee context, identity/bumps, dimension/byte/count bounds, deterministic eviction and unpaid retry limits remain covered. Root invalidation releases reservations before fresh admission. |
| C25.1–4 | Exact plain/blob bytes, proof-checked fetches, negotiated capabilities, bounds and structured receipts remain covered by native handlers and real peers. |
| C26.1–4 | Candidate state/slot/fees and public/private separation remain current. Added candidate storage-root rejection without contaminating canonical state/cache. Invalid Frame/Amsterdam transitions already refund their reserved gas dimensions; no speculative builder change. Native Engine construction uses disposable state and exact supplied slot. |
| C27.1–4 | Authenticated submission, complete base/keyed queries, original fee context, canonical receipt/body history and restart retain native consumers. |
| C28.1–4 | Authenticated raw simulation, exact selected fork, overrides/errors, dimensions and bounded trace/cancellation remain covered. Native configuration is reused by the STF probe. |
| C29.1–4 | Existing eight real-peer profiles remain mandatory. Seven new required native consumers and immutable dependency identities are enforced; source-bound full/race/build/vet/baseline/Python evidence is regenerated. |

## Counterexamples and qualification

The new native collision/cache/probe tests are also run with the eight affected
production files from exact predecessor `53904476`. Two compile-only additions
declare the new fork field and dependency enum; they do not add a producer,
reader or collision behavior. This permits unchanged new tests to distinguish
the missing behavior from the predecessor's working empty/prefunded deployments,
pre-fork replay, underfunded parent halts and clarification controls. Expected
vectors and refs are never modified to obtain a passing result.

The predecessor overlay fails thirteen regression subcases in five enclosing
native consumers; twenty-one control subcases and both clarification consumers
pass. This is an execution counterexample, not a compilation-failure control.

The former unit assertion that slot 100 should be missed was corrected to the
normative any-storage rule. It was an implementation-mirroring unit test, not
one of the frozen independent wire/state oracles.

Current local source/log identities, required consumers, optional skips and
retained historical counterexamples are recorded in A/B/C verification evidence.
Exact final-head hosted results belong to the live English PR checks and body;
predecessor workflow results never attest to a new head.

Implementation `79b82283` passes the full 240-package suite (24,078 passing
events, zero failures, 124 optional skips) and the 58-package affected race suite,
including STF (9,786 passing events, zero failures, 74 optional skips). Mandatory
A/B/C consumers are 62/79/102 with zero required skips. Full build/vet, both
binaries, independent baseline 12/12 and 94 Python checks in each mode pass.
Twenty historical counterexamples are retained alongside the new control.
