# Selected Group E composition and release boundaries

Baseline: master after PR #171, `cc31afffaa12919bbc08dc854c03af1e1928dc3c`.
This implementation selects merged EIP-7805 PRs #12470/#12702 at EIPs `c24e3301`,
and the exact Heze/Engine Bogotá companions in [PLAN.md](PLAN.md). The merged
pre-execution-only predicate supersedes earlier omission assumptions. See
[the latest clause/discussion audit](../../reviews/pr-172/LATEST_7805.md).
No mainnet fork timestamp, committee parameter or EIP-8369 consensus rule is added.

## Source and discussion audit

The November 27 review followed the selected EIPs, all eight Heze companion
documents and Engine Bogotá line by line. The November 28 recheck read the
complete merged EIP-7805 rebase and its review/discussion, and confirmed the
selected companion revisions remain unchanged. Verbatim snapshots and
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
then reverify retained signatures and ancestry; failure to obtain authenticated state blocks use. Replacing a live CL adapter reauthenticates the in-memory snapshot under the replacement context before committing the switch; a failed replacement preserves the previous view and quota.

The official Engine calls carry the caller's immutable list view. There is no
new bare-IL update method: new attributes create a replacement payload ID,
cancel unfinished work, and retain completed IDs within the bounded cache.
Context deadlines are local build policy (8 seconds), not a new consensus slot
constant. Invalid trials restore state and gas. Sender-indexed retries revisit
IL nonce and funding cascades; dynamic full-state Frame candidates retry after
state transitions. Exact appended block RLP sizing includes headers, list prefix
growth, scalar/3D calldata and blob-gas header widths, withdrawals and the
selected dependency-witness union.

The native backend executes and durably retains accepted payload views across
branches/restarts. Below finality it releases IL bytes and unexecuted branches while retaining compact completed verdicts for inherited Paris stale-head notifications. Execution validity and IL satisfaction are separate; a valid
omitting payload remains stored. `PayloadDecisions` supplies the Heze CL-side
decision predicates without changing ancestors or re-evaluating known-valid
decisions after pruning. An actual external CL must consume that verdict at
its fork-choice boundary. Bogotá methods are advertised only with a capable
backend; the alternate backend rejects those versions explicitly.

EIP-7805's omission state is taken immediately after the last transaction,
before withdrawals and request-system calls. Authenticated BAL post-phase
metadata preserves the fast Profile 1 account path when its fields are
unchanged; affected senders and Frame preconditions reconstruct the precise
transaction prefix using the shared processor. Only actual accepted payload
transactions are replayed; omitted candidates never execute. Missing BAL data reconstructs,
and corrupt commitments fail. The cache predicate version prevents old final-
block-state verdicts from bypassing this correction: retained views are
rechecked, while legacy compact records wait for exact CL view resubmission. BLOCKHASH uses the authenticated branch
header window even when bodies are unavailable; missing/corrupt required headers
propagate errors instead of changing the omission verdict.

Pre-frame nonce, aggregate fee and gas checks are now shared with execution.
Definite failures need no dependency witness. State access errors remain errors,
and precondition-valid omitted witnesses retain the explicit composition gate.
Native capacity includes ordinary state usage and Amsterdam gross execution;
missing derived receipt counters reconstruct from the transaction prefix.
Builder and import byte projections use the maximum consumed gas dimension.
Predicate version 3 rechecks retained versions 0–2 and requires exact CL view
resubmission for old compact records without retained list bytes. The native ingress commits an owned/pruned cache view
only after persistence and uses the exact clock already validated at arrival.

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
Historical full-state omission experiments do not select the native merged
EIP-7805 predicate or certify partial-stateless replay.

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

HGT-046 activation remains gated by D04. Selected independent Heze two-pair
network controls qualify signed gossip, inclusion/finality, native VALID/false
fork choice/restart and bounded PQ-byte carriage at their recorded source.
Final publication-head requalification is tracked separately in the English PR.
Exhaustive beacon reqresp/fork-digest, mixed-client/ePBS and public-tail
qualification remain HGT-070/080 prerequisites; no public activation is claimed.

## Merged omission rules and builder validity

The Profile 1 account helper is an optimization, not authority to narrow base
EIP-7805 to Informational 8369 candidates. Every decoded omitted transaction
uses current-chain pre-execution validation. Type 3 is explicitly ignored.
Canonical Frame context and static protocol signatures, sender nonce, aggregate
fees, balance and intrinsic/reservation checks share actual executor helpers.
Runtime VERIFY/body/storage failure does not excuse a statically valid omission.
Each candidate uses the same immutable transaction-end state and capacity.

Ordinary execution reservation is `min(tx.gas, 2**24)` and state reservation is
`tx.gas`, checked against the actual independent EIP-8037 pools. Header
`gas_used` is the maximum used dimension and cannot reconstruct those pools.
Canonical Frame reservations use its selected execution/state budget. Missing
derived receipt counters may replay the accepted transaction prefix, never the
candidate. Static-invalid envelopes and insufficient capacity are decidable
before undefined omitted-dependency witness composition is considered.

Actual builders execute candidate transactions and enforce EIP-7934 complete
block RLP validity. Omission evaluation neither executes nor projects an append.
EIPs #12723 proposes a distinct size exemption and remains unmerged/disabled;
execution-specs #4041 is also unmerged. The real near-8-MiB native regression
checks that an otherwise valid candidate is not excused by speculative size
policy. High-gas size-limit composition remains an upstream release gate.

Type-3 IL entries cannot stall a builder or cause it to require a sidecar.
Authenticated normal pool blobs remain buildable with their actual sidecars.
Canonical PQ entries may reuse an authenticated pool witness for actual builds;
missing witness data fails that build and an identical owned payload ID can retry
after admission. Successful completed IDs remain immutable. The current native
consumers cover this retry, canonical GetIL blob exclusion, normal pool blob
availability, pre-execution-only Frame/storage behavior and version-3 migration.

The latest #12702 clarification counts EIP-2718 canonical transaction bytes.
Exactly 8192 bytes are allowed; block RLP/SSZ framing and external witnesses are
not part of this budget. The native boundary consumer and source checker lock
this contract. See [LATEST_2718.md](../../reviews/pr-172/LATEST_2718.md).
