# Group E transaction-boundary completion audit

> Historical pre-rebase audit. Merged EIP-7805 PR #12470 supersedes candidate
> execution, type-3 enforcement, size exemptions and cache-version assumptions
> below. Use [LATEST_7805.md](LATEST_7805.md), the current criterion plan and
> source-bound qualification for the active rules. Original findings/results
> remain preserved with their recorded source identities.

Date: November 27, 2026 (Asia/Shanghai). Starts from published PR #172 head
`63f85e737d8dd84c8913d6f6ef996179e5e373d5`, based on merged PR #171/master
`cc31afffaa12919bbc08dc854c03af1e1928dc3c`. The independent Lean work in the
original worktree and all independent A/B/C/D expected oracles are preserved.

## Normative decision

EIP-7805 defines S immediately after the last transaction in B. EIP-4895
withdrawals occur after user transactions. EIP-7002 request draining is a
post-execution system call. EIP-7928 distinguishes pre-execution index 0,
transaction indices 1 through n and post-execution index n+1. Therefore final
block state is not generally S. A withdrawal in B cannot fund an IL transaction
appended before that withdrawal, and a later request call cannot change the
validity of a Frame evaluated at the transaction-end boundary.

EIPs master `68db81b0a75edb0e16d8f181241f994875eba52d`, Heze
`36888ee87b0faa6ad7804b017a0184e8e6e6a1c6` and Engine Bogotá
`9e85b16c7fb54a838c9a612842c138c2bdf9bc8d` remain current. The additional
2935/4895/7002/7928 files in read-only refs were byte-compared with that current
EIPs revision; their immutable hashes join the E register (21 sources total).
The 7805 discussion still has 17 posts, last posted November 21; the 8369
discussion still has four, last posted September 9. Their previous decisions,
including the author retaining claimed-index evaluation, remain unchanged.

## Reproduced missing behavior

The native builder correctly skipped an unfunded IL transfer and then processed
its withdrawal. Native newPayloadV6 nevertheless used the final block balance
and returned VALID/false. A raw Engine regression failed before correction.

An upgrade regression also failed: even after correcting evaluation, an older
persisted Boolean survived restart without reevaluation. Correcting the code
alone would leave earlier wrong satisfaction decisions usable by FCU.

## Implementation

The existing state processor now exposes a transaction-prefix entry point
sharing the same transaction execution loop. It includes actual pre-execution
systems, transactions and their receipt/gas accounting, and stops before
withdrawals and request collection. EIP-2935 leaves BLOCKHASH's 256-block semantics unchanged. Native IL execution
now follows the authenticated header chain even when snap history has no bodies.
Each link authenticates its parent hash, so the queried header itself is not
required. The bounded per-evaluation hash window is cached. Missing/corrupt
required intermediate headers propagate callback errors instead of providing
a fabricated zero hash that could make a valid Frame appear omittable. A native
snap-pivot regression reproduced that wrong satisfaction verdict before the fix.

The ordinary full-block entry point retains
all original finalization behavior. A prefix BAL is not a whole-block commitment.

Native omission checks authenticate the stored, execution-validated BAL and use
phase n+1 only to identify potentially changed fields. For Profile 1, untouched
sender account fields retain the existing eligibility path with no candidate
EVM execution. Affected senders require exact prefix reconstruction. Frames,
other full-state candidates and exact byte-boundary execution probes require
reconstruction when any post-phase state changes could affect them. Missing
historical BAL data also requires reconstruction; corrupted commitments fail.
Reconstruction uses the payload's parent branch, proper fork state conversion,
pre-execution systems and BLOCKHASH ancestry. It does not fabricate old values
from the BAL, whose wire format contains post-values only.

Completed native satisfaction records now carry an internal predicate version.
This is database metadata, not an Engine field, EIP constant or fork parameter.
Legacy live records are reevaluated with their exact retained CL views before
FCU. Legacy compact records have lost those views and return SYNCING/null until
the CL resubmits newPayloadV6 with its list. Current compact records retain the
new version and correct Boolean. The execution-sync admission path likewise
requires a current, resolved verdict record; execution validity itself is unchanged.

The shared Frame executor now exposes its existing immutable fork, chain-ID
and selected nonce-schema checks before either IL witness gate. A foreign-chain
or inactive-schema dependency envelope is already invalid, so both import and
building ignore it without demanding an unspecified witness. The executor uses
the same helper, preserving independent explicit research-profile behavior.
Actual native wire subtests reproduced both errors before this correction;
existing valid-witness and malformed-signature assertions remain unchanged.
This does not assign any transport or validity rule to a context-valid omitted
PQ transaction whose required witness is unavailable.

## Native acceptance cases

- A same-block withdrawal cannot fund an omitted transaction; the final block
  still credits the full withdrawal, and the next block can spend that credit.
- Missing/pruned BAL data reconstructs the same verdict. A corrupt stored BAL
  raises an error, and restoring its authenticated bytes restores normal service.
- A post-system storage change cannot make an otherwise valid Frame omittable,
  or make an invalid-at-transaction-end Frame mandatory. Actual private builds
  agree with both import verdicts and leave final block state unchanged by probes.
- Prefix processing includes actual pre-system storage writes, excludes
  withdrawals/post-system writes, and leaves parent state unmodified.
- Header-only authenticated snap ancestry supports a real BLOCKHASH-dependent
  omitted Frame after the pivot; removing a required header returns an error.
- Wrong-chain and inactive nonce-schema PQ envelopes are ignored by native
  import/build before witness requirements; actual valid pool witnesses still
  support canonical inclusion.
- Restart corrects a legacy retained wrong verdict. Legacy compact metadata
  requires its original view to be resubmitted. Current-version compaction
  preserves the verdict and latest-view replacement behavior.

The older native fork-activation test's manually supplied empty view now carries
the current cache predicate version; its activation/reorg assertions are unchanged.
Actual legacy migration is independently exercised by the native wire cases above.

These cases add four mandatory consumers, bringing the register to 34. Eleven
additional compiled-removal controls cover state selection, prefix execution,
pruned metadata, legacy live/compact migration, version-preserving compaction, header-only
ancestry missing-header error propagation and both early immutable-context guards.
The complete control register contains 26 cases. Actual fresh execution results
are recorded in VERIFICATION.md and the source-bound evidence after qualification.

## Complete task review

| Task | Result of this pass |
|---|---|
| HGT-040 | Current source/discussion precedence remains consistent; the exact last-transaction cut point is now explicit and pinned. |
| HGT-041 | SSZ, current-slot authenticated gossip, two-message allowance, timely/late views, equivocation, committee-position requests and Heze cache restoration remain unchanged. |
| HGT-042 | Bogotá methods, schemas, opaque malformed IL bytes, custody widths, capabilities and VALID Boolean/non-VALID null remain enforced; unresolved legacy compact records use SYNCING/null. |
| HGT-043 | Existing request ownership, replacement/cancellation/deadlines, nonce/funding retries and pool wrapper recovery remain in the shared builder; the new cut-point tests independently compare private builds and imports. |
| HGT-044 | End-of-transaction account/full-state evaluation now excludes later operations, preserving pre-systems, exact bytes, remaining gas, full-width fields and RLP probes. |
| HGT-045 | Actual native Engine and restart consumers now also qualify changed evaluation predicates and retained/compact cache migration. |
| HGT-046 | Defined reference predicates and D04 prerequisites remain implemented; no Informational 8369 consensus activation is added. |
| HGT-047 | Local authenticated ingress and CL satisfaction/fork-choice predicates remain qualified; external HGT-070/080 transport and devnet acceptance remain open. |
| HGT-048 | Real byte-capacity/verifier/local-pipeline measurements remain bounded evidence; no omitted-witness transport or network-tail qualification is implied. |

The selected defined implementation is distinct from complete public activation.
The same Profile 2 enforcing-specification, omitted PQ witness and external
CL/ePBS prerequisites remain open and visible on the website.
