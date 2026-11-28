# Merged EIP-7805 rebase qualification

The subsequent merged EIP-2718 byte-accounting clarification and current
completeness review are recorded in [LATEST_2718.md](LATEST_2718.md). The
pre-execution corrections below remain applicable; their original source and
qualification identities are preserved.

The final upstream recheck observed EIPs master
`fbc43ec3a187f0b3dd448e995a041227f4baa7b1`, merging PR #12470. The complete
EIP, review comments and discussion were read. The immutable new snapshot is
`specs/hegota/upstream/2026-11-28/group-e/eip-7805.md` (Git blob
`0ff894bb4136c6973c1f91a4598e8d5426f8b237`). Earlier November 27 snapshots,
source-specific measurements and qualification remain historical evidence.

The rebase aligns the inline CL contract with ePBS/Heze: current-slot gossip,
branch/dependent-root identity, late storage, equivocation, bid bits and EMPTY
fork choice. These match the selected current Heze adapter. The EIP's older
linked CL revision differs from the selected current head in optimistic-sync
and networking; the current `36888ee8` companion and Bogotá `9e85b16c` remain
pinned. SSZ/BLS expected oracles are unchanged.

## Execution rules and corrections

The merged EL algorithm checks every missing transaction independently against
the state after the last transaction and the same remaining capacities. It
ignores malformed envelopes and type 3; applies the current chain's
pre-execution rules; and checks EIP-8037's separate execution/state reservations.
It neither executes the candidate nor changes state or consumes capacity.

The former non-Profile-1 path replayed an omitted Frame. A native two-node
regression reproduced a wrong satisfied verdict when a statically valid Frame's
VERIFY would revert. The new path shares the executor's existing read-only
Frame preconditions, including protocol signatures, context, nonce, aggregate
fees and reservations. Runtime prefix/body/storage behavior is not substituted
for pre-execution eligibility. Informational Profile 2 is still not activated.
The corresponding former runtime-storage test is replaced by the merged-rule
assertion; its original expected behavior remains in immutable head `6796dbc6`.

Ordinary candidates now use the accepted payload's actual independent gas pools:
`min(tx.gas, TX_MAX_GAS_LIMIT)` against execution capacity and `tx.gas` against
state capacity. The header's maximum-used dimension cannot recover these pools.
The new account-only consumer covers asymmetric capacity, exact boundaries,
one-over failures and repeated checks without state/gas consumption.

Type-3 entries are ignored by omission checking and do not force a builder to
wait for an unavailable IL blob. Normal authenticated pool blobs remain
buildable. The original failed-build same-ID retry control is preserved with
a real missing dependency witness, where retry remains necessary.

## Size policy and builder validity

EIPs PR #12723 and execution-specs PR #4041 are open, unmerged proposals. The
merged EIP has no byte-size omission exemption. Its author explicitly leaves
this issue to #12723. The former speculative exact-RLP omission probe is removed;
it also violated the prohibition on candidate execution at gas-header boundaries.
No alternative proposed payload-size limit or margin is activated.

Actual building and block import still enforce EIP-7934. A new native backend
consumer builds and imports a real near-8-MiB block of signed ordinary
transactions, establishes that a valid IL candidate's counterfactual append
exceeds that RLP limit, and verifies that the unmerged size excuse is absent.
The high-gas composition risk remains an explicit upstream release prerequisite.
This does not relax block validity or claim that every high-gas payload can
satisfy a malicious IL under the current draft.

## State, caches and optional composition

Reconstructing the accepted payload's transaction prefix remains permitted and
necessary when its authenticated transaction-end state/dimensional receipts are
unavailable. That reconstruction may use BLOCKHASH and must surface missing
ancestry. The omitted candidate itself never runs; its code's BLOCKHASH reads
cannot introduce an availability requirement. The ancestry consumer now tests
both facts directly.

Satisfaction predicate version 3 invalidates versions 0–2. Retained exact views
are reevaluated; old compact verdicts require CL view resubmission. New live and
compact version-2 migration controls prevent stale full-execution/size verdicts
from surviving the update.

EIP-8504 is still an independently selected, unmerged draft. Its dependency
witness checks are additional block-validity requirements, and neither it nor
this rebase defines omitted-IL witness transport/composition. Known-invalid
context/preconditions and insufficient capacities remain decidable without a
witness; passing them does not silently certify the undefined composition.

## Evidence discipline

Current consumers and source hashes must be freshly qualified. Old compiled
controls touching changed production/tests are not re-claimed as unchanged.
Active controls are updated to the merged predicate and executed again;
obsolete full-candidate-storage/blob-enforcement controls are retained only in
the previous immutable qualification. Independent expected vectors and read-only
refs are preserved. Final publication-head CI and actual hosted proof/network
results are recorded in the English PR.
