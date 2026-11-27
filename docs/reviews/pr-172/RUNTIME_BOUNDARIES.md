# Group E complete-consumer runtime audit

Date: November 27, 2026 (Asia/Shanghai). Published starting head:
`15262949d2b4dbf0f41bf4919dd0a2d0d5550c7c`. Base remains merged PR #171/master
`cc31afffaa12919bbc08dc854c03af1e1928dc3c`. Independent Lean work, read-only
references and A/B/C/D expected oracles are preserved.

## Current sources and precedence

EIPs `68db81b0a75edb0e16d8f181241f994875eba52d`, consensus-specs Heze
`36888ee87b0faa6ad7804b017a0184e8e6e6a1c6` and Engine Bogotá
`9e85b16c7fb54a838c9a612842c138c2bdf9bc8d` remain current. The 7805 discussion
still has 17 posts (last November 21), and the 8369 discussion four (last
September 9). Merged PR #12110 retains the enforcing-extension and claimed-index
requirements. Optional proof PR #12672 remains open at
`96da3c17047ad31608c40bfe75049db78dc4417d`; it defines no omitted-IL witness wire.

EIP-8037 was read in the local reference and compared with current upstream.
The local file differs in creation-collision/system-call clarifications and
rationale, so the existing Group C/D verbatim snapshot is reused without duplication at
`specs/hegota/upstream/2026-11-27/eip-8037.md` (blob
`43cb2b4fb8c72a66d42a06e21a75b70f0539ccbc`). The independent execution/state
reservation and max-of-dimensions header rules are unchanged. The E register
now pins 22 immutable normative files; an unchanged repo head does not imply
that every old local reference file has current content.

The binding clauses used in this repair are:

- EIP-7805 checks missing transactions against the state after the last
  transaction, tolerating invalid candidates. Bogotá requires independently
  reported execution validity and inclusion satisfaction.
- EIP-8141 checks the sender nonce, structural/protocol signatures and aggregate
  `max_fee` before any frame. The fee covers execution, state and blob gas;
  sufficient execution-only fees do not establish the complete fee budget.
- EIP-8250 replaces the selected nonce rule with its exact legacy/keyed sequence
  predicate. The existing explicitly selected composition remains unchanged.
- EIP-8037 and EIP-8141 reserve execution/state separately. Ordinary transactions
  and Frames share the block state pool. With EIP-7778, execution accounting is
  gross and state usage is net. Header `gas_used` is their maximum, not paid gas
  or their sum. System-call gas does not contribute to either block dimension.
- EIP-7934 bounds the actual complete block RLP. A wrong projected `gas_used`
  integer can change its RLP width at the exact byte boundary.
- Heze `on_inclusion_list` failures must not modify the live store. Timeliness
  uses the authenticated arrival clock, and failed cache publication must not
  consume quotas or prune the previous view.

## Reproduced gaps and repairs

### Pre-execution validity before witness handling

Actual native RPC tests failed for both selected nonce formats when a candidate
had a wrong nonce, insufficient aggregate fees, insufficient blob fees, or a
reservation larger than either gas dimension. The backend returned an omitted
witness error instead of recognizing a definite failure; private builds also
failed for the nonce and fee cases.

`prepareFrame8141` now contains the existing executor's nonmutating preparation
rules. Execution, native omission checks and private dependency builds share
those rules. The public precondition check neither runs EVM/proof code nor
consumes nonces, fees, warmth or pool capacity. Lazy state-read failures remain
errors and take priority over a fabricated invalid nonce. Builder nonce failures
remain retryable after an earlier included transaction changes their state.

Known invalid nonce/fee/global-cap candidates do not demand unnecessary receipt
history. A candidate passing preparation still does not establish proof validity.
Its omitted-witness composition remains explicitly unsupported. Positive native
controls exercise that error; no witness field, proof execution or new omission
excuse is assigned.

### Mixed-envelope capacity and persistence

A real ordinary storage transaction consumes state gas before an omitted Frame.
The old backend summed only Frame state gas and used the wrong execution
fallback on Amsterdam-only configurations. Native import could therefore report
an omission as unsatisfied when the Frame's state reservation did not fit, or
excuse a Frame that fit the exact execution boundary.

`BlockGasDimensions` shares the existing block-validity accounting. Native
capacity reconstructs both dimensions, including ordinary `StateGasUsed`, and
uses Amsterdam gross execution. Canonical receipt RLP omits derived counters;
missing metadata triggers transaction-prefix replay rather than treating zero
as available capacity. Actual native build/import and restart tests exercise
both an exhausted state dimension and an exact execution boundary.

The internal predicate version is now 2. Version-0/1 retained views are rechecked;
legacy compact records wait for the original CL list to be resubmitted. Current
compact verdicts remain reusable. A native regression injects the published
version-1 wrong Boolean, restarts and verifies migration for retained and compact
records. This version is database metadata, with no new Engine or fork field.

### Exact RLP projection across both gas pools

A compiled codec/builder regression reproduced an oversized block being admitted
when state use crossed an RLP integer-width boundary and execution stayed below
it. Builder size projection now uses the actual maximum consumed dimension.
Native execution probes use the same pool-derived header value for both ordinary
and Frame appends. Profile 1 uses the previous header total as a conservative
lower bound on multidimensional growth, then probes ambiguous boundaries exactly.
The scalar declared-gas eligibility rule in base EIP-7805 is preserved.

The serialization regression independently fills a complete block to the byte
limit with a state-dominated header. An extra byte is rejected, the exact limit
is accepted and rejection does not poison the prefix or reserve gas. This is a
codec boundary test, not a claim that its padding transaction is executable;
the independent native storage/Frame tests qualify actual resource accounting.

### Atomic native ingress publication

Real filesystem failures during temporary-file creation and rename reproduced
memory changes despite an unsuccessful handler call: earlier lists were pruned
and retries lost their two-message quota. A second clock lookup after successful
authentication also made a valid message fail after mutation.

The native adapter now works on an owned tentative store. It uses the exact
clock returned by gossip validation, prunes that tentative view, persists its
snapshot, and publishes memory after success. Failure before publication leaves
live bytes, timeliness, equivocation records, pruning and quotas untouched.
Actual filesystem regressions verify repeated failures, recovery, quota counts,
prior-file preservation, temporary-file cleanup and authenticated restore.
The existing durable file synchronization/rename protocol is preserved.

## Complete task review

| Task | Result of this pass |
|---|---|
| HGT-040 | Current source and discussion precedence rechecked; current 8037 content pinned separately from stale local reference. |
| HGT-041 | SSZ, domains, authenticated branch identity, strict timing, quota, requests, retention and equivocation reviewed; native publication/clock failures repaired. |
| HGT-042 | Exact Bogotá schemas, opaque malformed ILs, capability/version checks, custody widths and VALID Boolean/non-VALID null retained. |
| HGT-043 | Owned/cancellable builds, nonce/funding retries and rollback retained; shared preconditions and dimensional byte accounting repaired. |
| HGT-044 | Signature/account/gas/byte eligibility reviewed; exact transaction-end state retained, complete preconditions and mixed-envelope capacity repaired. |
| HGT-045 | Native imports/builds and restart remain independent of satisfaction; version-2 retained/compact migration and atomic ingress now exercised. |
| HGT-046 | Defined informational reference predicates and all D04 prerequisites rechecked; no incomplete enforcing profile activated. |
| HGT-047 | Authenticated native ingress, optimistic transitions, ancestor stability, availability and bid-bit predicates reviewed; external CL/ePBS qualification remains HGT-070/080. |
| HGT-048 | Actual bytes/verifier/local pipeline measurements remain bounded; definite invalidity is distinguished from undefined witness composition. |

Five additional mandatory consumers bring the register to 39. Existing
consumers remain required. New compiled-removal controls cover preparation,
state/gross accounting, predicate version, dimensional RLP, read failures,
nonmutation and atomic ingress. Historical context/static removal controls now
remove both redundant protections while retaining their unchanged consumers.
Only compiled behavioral failures count; final results belong to the fresh
source-bound verification artifacts and the English PR's exact-head checks.

Public activation still requires an enforcing Profile 2 specification, an
omitted PQ witness composition and external Heze CL/ePBS acceptance. This audit
implements determinate rules without claiming those prerequisites are fulfilled.
