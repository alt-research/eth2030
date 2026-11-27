# Selected Group E composition and release boundaries

Baseline: master after PR #171, `cc31afffaa12919bbc08dc854c03af1e1928dc3c`.
This implementation selects the exact Heze and Engine Bogotá revisions in
[PLAN.md](PLAN.md), not the older inline CL/Engine sketch in EIP-7805. It adds
no mainnet fork timestamp, committee parameter or EIP-8369 consensus rule.

## Source and discussion audit

The November 27 review followed EIP-7805 and EIP-8369, all eight current Heze
companion documents and Engine Bogotá line by line. Verbatim snapshots and
blob/SHA-256 identities are in `specs/hegota/group-e.json`. EIP-7916 confirms
ProgressiveList serialization and JSON mapping equal ordinary SSZ lists;
only Merkleization changes. Empty list serialization is empty bytes, not a
zero sentinel. The earlier Lighthouse unit sample `0x00` was an invalid,
previously ignored placeholder; its replacement exercises actual offsets.

The [7805 discussion](https://ethereum-magicians.org/t/21578) was checked through
the latest committee-selection/incentive exchange. It introduces no replacement
binding container or timing rules. The [8369 discussion](https://ethereum-magicians.org/t/29298)
and [PR #12110 review](https://github.com/ethereum/EIPs/pull/12110) retain the
builder-claimed index and its deliberately weaker guarantee for nonce queues
and shared payers. Fresh keyed nonces do not remove the index requirement.
The occurrence-before-dedup signature debit rule, low-slot surface, absence of
a proxy/namespace exception, authenticated code and missing-data requirements
are preserved. Candidate budget values have not become consensus constants.

## Ownership and runtime path

`focil/heze` is the canonical companion implementation. Existing committee-root,
Keccak/RLP, fixed-transaction-count and gas-capped FOCIL helpers remain legacy
research APIs. They cannot authenticate the native Heze ingress. A Node requires
an explicit `ConsensusView` supplied by a CL: real shuffling ancestry, processed
committee, validator keys, fork version and genesis validators root. Missing
context never disables BLS checks. The current native topic manager is an EL
adapter, not a beacon Gossipsub stack.

The store is keyed by slot/dependent beacon root. First arrival determines
timeliness; the exact deadline is strictly before 7500 basis points. Late lists
remain available for honest builders and equivocation detection. Two conflicting
messages exclude the member from both views. Request indices are committee
positions; repeated committee members are deduplicated in responses. Retention
keeps slot N through N+MIN_REQUEST_SLOTS+1 inclusive. Reopened stores first discard expired entries and reject future cache entries,
then reverify retained signatures and ancestry; failure to obtain authenticated state blocks use.

The official Engine calls carry the caller's immutable list view. There is no
new bare-IL update method: new attributes create a replacement payload ID,
cancel unfinished work, and retain completed IDs within the bounded cache.
Context deadlines are local build policy (8 seconds), not a new consensus slot
constant. Invalid trials restore state and gas. Sender-indexed retries revisit
IL nonce and funding cascades; dynamic full-state Frame candidates retry after
state transitions. Exact appended block RLP sizing includes headers, list prefix
growth, withdrawals and the selected dependency-witness union.

The native backend executes and durably retains accepted payload views across
branches/restarts. Execution validity and IL satisfaction are separate; a valid
omitting payload remains stored. `PayloadDecisions` supplies the Heze CL-side
decision predicates without changing ancestors or re-evaluating known-valid
decisions after pruning. An actual external CL must consume that verdict at
its fork-choice boundary. Bogotá methods are advertised only with a capable
backend; the alternate backend rejects those versions explicitly.

## AA-VOPS and PQ gates

EIP-8369 is Informational. `vops_reference.go` implements explicit-parameter
research predicates for four shapes, signature/prefix occurrence budgets,
ordinary low slots, authenticated code and default indices. Declared execution
and state limits are summed only for these local resource experiments; no
composed gas constant is assigned. Unsupported/unrecognizable prefix shapes
use conservative full declared-frame accounting and are never admitted. This
is not a complete Profile 2 eligibility/replay engine. Every enforcement request
returns `ErrProfile2NotSpecified`; its thirteen prerequisites enumerate claims,
canonical diffs, isolated replay, final bounds, vectors and post-reveal duties.
The existing Group C full-state Frame omission consumer remains a separate
explicit experimental profile; it does not certify partial-stateless replay.

The PQ measurement uses real canonical EIP-8504 transaction bytes and uncached
leanSPHINCS verification, then the signed SSZ/BLS/store/union pipeline. A 167-byte
sample fits 49 times in the 8192-byte transaction sum; these repeated occurrences
measure byte capacity, not 49 independently spendable transactions. Its 7740-byte
witness is external to the canonical transaction. Measurements report host,
runtime and exact scope. They include no network tail-latency claim. Omitted-IL
witness composition remains unspecified; the native backend returns an explicit
error instead of treating missing witnesses as an omission excuse. Committee
size and IL bounds are unchanged. Canonical PQ IL envelopes reuse an already
authenticated local pool wrapper for building; exact transaction identity stays
canonical and the included dependency union carries the witness. This is not an
omitted-IL witness transport assignment.

HGT-046 activation remains gated by D04. External Heze Gossipsub/reqresp framing,
fork digests, independent CL/ePBS devnet and network latency qualification remain
HGT-070/080 integration prerequisites. Completion of local native consumers
does not mean those external duties or Profile 2 consensus are complete.

## Base FOCIL versus informational profile boundaries

The Profile 1 account predicate is an optimization, not authority to exclude
other envelopes from base EIP-7805. Non-Profile-1 envelopes retain independent
full-state append validation, including blob-gas capacity. Blob and Frame
transactions outside the informational 8369 candidates do not automatically
receive a new consensus omission excuse. Static-invalid Frame/proof envelopes
are tolerated before any unavailable-witness gate.

A canonical IL blob can reuse its admitted pool sidecar. A valid blob without
data aborts building instead of returning an unsatisfied payload or empty blob
bundle. Completed failed IDs can be retried with the same exact attributes
after availability changes; successful completed IDs remain immutable. The
new native regression exercises real KZG admission, invalid signatures,
execution-valid-but-unsatisfied import, missing-data failure, same-ID retry and
the complete authenticated blob bundle.
