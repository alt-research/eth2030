# Group E: normative implementation and acceptance

Baseline: merged PR #171 (`cc31afffaa12919bbc08dc854c03af1e1928dc3c`).
Scope: HGT-040–048, nine tasks and 36 criteria below. References are captured
verbatim outside read-only refs in `specs/hegota/upstream/2026-11-27/group-e`.
EIPs master: `68db81b0a75edb0e16d8f181241f994875eba52d`;
consensus-specs: `36888ee87b0faa6ad7804b017a0184e8e6e6a1c6`;
execution-apis: `9e85b16c7fb54a838c9a612842c138c2bdf9bc8d`.

## Specification decisions

EIP-7805 states the base inclusion contract, but its inline CL containers,
domain and Engine status describe an older companion revision. The selected
native companion is current pinned Heze plus Engine Bogotá: `(slot,
dependent_root)` identity, SSZ/progressive transaction roots, domain
`0x10000000`, 8192 transaction bytes, committee size 16, and independent
`PayloadStatusV2.inclusionListSatisfied`. IL dissatisfaction never makes a
valid execution payload invalid. Old committee-root/Keccak/RLP helper types,
16-transaction/2M-gas caps and percentage-based compliance scores are legacy
research APIs and are not the selected Heze protocol.

Heze gossip accepts only the current slot; the import handler and request
protocol also support the specified retention window. Timely means strictly
before 75% of the slot. Late but validated ILs remain available to builders and
can prove equivocation; they do not expand an attester's timely view. Signed
messages require complete consensus context and BLS verification. An EL cannot
invent beacon committees, shuffling ancestry, validator keys or fork versions.

The selected Engine contract is `getInclusionListV1`, `forkchoiceUpdatedV5`
and `newPayloadV6`. `engine_newInclusionListV1` is an older nonstandard extension,
not a substitute for the official request's exact transaction view. Capability
advertising must agree with actual backend support. Per-payload accepted views
must survive deferred validation, branches and restart.

EIP-8369 is Informational and explicitly requires an enforcing Standards Track
extension before Profile 2 activation. Claims encoding/commitment/limits,
canonical diffs, code-load limits, final slot/budget constants and post-reveal
duties remain undefined. Implement the determinate reference predicates and
negative controls with explicit supplied research parameters; do not assign
wire fields or activate AA-VOPS by choosing candidate numbers. Existing native
full-state Frame omission replay is a separate selected experimental consumer
from Group C, not evidence of Profile 2 or partial-stateless enforcement.

EIP-8504 does not define omitted-IL witness transport or verification. PQ IL
capacity and resource measurements therefore remain bounded research evidence;
no committee/IL limit is increased and no unsupported omission is certified.
Heze/ePBS external-network qualification remains shared with HGT-070/080.

## Transaction-end state boundary

EIP-7805 evaluates omissions after the last transaction, before EIP-4895
withdrawals and post-execution request calls. Execution-validated EIP-7928
phase n+1 metadata identifies whether final block state can serve that check;
affected account/full-state probes require an exact transaction-prefix state.
Missing metadata requires reconstruction, and corrupted metadata is an error.
BLOCKHASH probes use authenticated branch headers independently of body
availability; missing required headers propagate as errors.
Native import and building share the executor's immutable Frame context checks,
so wrong-chain and inactive-schema envelopes are ignored before witness gates.
Context-valid omitted PQ witness composition remains undefined.
Cached verdicts must identify the predicate used; legacy records with retained
views are recomputed, while legacy compact records require CL view resubmission.
See `docs/reviews/pr-172/TRANSACTION_BOUNDARY.md` for the complete audit.

## Pre-execution validity and gas dimensions

Native dependency candidates share the actual executor's nonmutating nonce,
aggregate fee (including blobs) and reservation rules before witness handling.
Passing those rules does not establish proof validity or define omitted-witness
transport. State read failures remain errors. Both ordinary and Frame usage
contribute to independent capacities; missing derived receipt counters are
reconstructed from the authenticated transaction prefix. RLP projection uses
the maximum consumed dimension, including state-dominated prefixes. Internal
predicate version 2 migrates retained version-0/1 views; compact legacy records
require their exact CL view to be resubmitted.

Native Heze ingress uses one validated arrival clock and publishes an owned
view only after cache persistence. Failed writes cannot consume a live quota
or prune previously retained messages. The complete nine-task audit is in
`docs/reviews/pr-172/RUNTIME_BOUNDARIES.md`. Current EIP-8037 is pinned by reusing
the existing current C/D snapshot, rather than editing stale read-only refs.

## Complete acceptance register

| Task | Criteria |
|---|---|
| HGT-040 | 1: immutable current sources/discussions and precedence; 2: explicit timings/branch identity; 3: execution validity separate from satisfaction; 4: Profile 2/PQ/external activation gates enforced and disclosed |
| HGT-041 | 1: canonical SSZ and independently frozen roots/domains; 2: branch-keyed exact-byte store/deep copies/dedup; 3: timely/late/equivocation/request behavior; 4: retention/pruning/reorg/restart |
| HGT-042 | 1: exact Bogotá methods/required fields/capabilities; 2: malformed IL transaction tolerance and strict DATA wire; 3: VALID Boolean versus other-status null; 4: fork/version/backend rejection and retained-view FCU |
| HGT-043 | 1: all IL and pool funding/nonce cascades reach a fixed point; 2: duplicate/invalid trial rollback and exact gas accounting; 3: request ownership/build-ID/rebuild isolation; 4: concurrent update/cancellation/deadline behavior without deadlocks |
| HGT-044 | 1: exact envelope presence and type/signature/chain validity; 2: nonce/256-bit balance/fee/code checks without EVM replay for Profile 1; 3: declared gas and exact appended RLP size including prefix changes; 4: malformed/blob/Frame/unknown/incomplete-context distinctions |
| HGT-045 | 1: actual native Engine build from nonempty IL; 2: import retains execution validity and separate satisfaction; 3: accepted branch/restart/finalization persistence; 4: two actual nodes and explicit alternate-backend behavior |
| HGT-046 | 1: exact static shapes/expiry/batch rules; 2: occurrence-before-dedup signature/prefix budget debits; 3: code-hash/state/access/replay requirements and claimed-index default semantics; 4: exhaustive enforcement prerequisites reject incomplete activation |
| HGT-047 | 1: authenticated gossip-to-store-to-Engine consumer; 2: omission blocks extension while valid payload remains stored; 3: timely/late/equivocation/branch/full/invalid exceptions; 4: optimistic transition/ancestor/pruning stability and bid-bit inclusivity |
| HGT-048 | 1: actual canonical PQ transaction/witness sizes and 8KiB capacity; 2: real verifier resource measurements; 3: bounded propagation/validation pipeline measurements with declared environment; 4: unchanged limits and explicit unsupported PQ omission/CL deployment |

All completion claims require named native tests, fresh pass events and source
fingerprints. Prior A/B/C/D expected oracles are preserved. Website/GitHub Pages
roadmap and progress must describe these exact results and remaining gates.
