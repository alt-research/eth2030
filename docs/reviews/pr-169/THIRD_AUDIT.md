# Third Group B clause audit, 2026-11-26

The second audit and its green CI did not cover every memory and static-context boundary. This review checked the complete Group B task list again against the upstream EIPs master, which still resolves to `ade9e9cab7e148bba4cd293f916a4a189cd7a8aa`. The consensus reference and merged master baseline remain unchanged. Reference contents were not edited.

## Findings reproduced before correction

| Finding | Normative basis | Correction and executable witness |
| --- | --- | --- |
| APPROVE incorrectly expands memory for zero length and rejects a full-width offset | EIP-8141 APPROVE Gas and instruction behavior explicitly follow RETURN memory semantics. An empty region does not access or expand memory, irrespective of its offset. | Reuse the existing RETURN memory-size function. `TestFrame8141ApproveEmptyReturnIgnoresOffset` checks offsets 0, 32, 2^64 and 2^256-1 at exactly 108 frame execution gas: 100 warm entry plus 8 for pushes. The old implementation charges 111 at offset 32 and invalidates the transaction at the two full-width offsets. |
| A static SSTORE exception incorrectly records a storage read | EIP-8141 VERIFY uses STATICCALL restrictions; EIP-214 Specification prohibits SSTORE in static execution. EIP-7928 records actual accesses, so an operation rejected before gas calculation does not access its slot. | Preserve `writes: true` when installing the Frame-specific SSTORE operation. `TestFrame8141StaticSstoreHasNoStorageAccess` catches the child exception in a normal sender frame, checks unchanged zero/nonzero storage and no state gas, and requires the writer account but no slot in BAL. Child budgets 2300 and 50000 distinguish a stipend rejection from the previously incorrect implicit read. |

`TestFrame8141ApproveReturnMemoryBoundaries` additionally checks the returned last byte, a full word, expansion across a word boundary, zero length at a full-width offset, and nonempty offset/length overflow. Successful frame gas is exactly 120 or 123 including the initial MSTORE; invalid regions restore approval, state root and both block reservations. A call tracer verifies the actual returned bytes.

All ten APPROVE memory cases also execute the same program with only the final APPROVE changed to RETURN in the unchanged upstream Geth v1.17.10 Amsterdam VM. The extra scope stack word is ignored by RETURN. Output, success/failure and body gas must agree, excluding the Frame-only 100-gas entry. This is a memory-semantics oracle, not independent implementation of the Frame envelope.

## Negative control

The final new tests were executed with the two original production files (`frame_8141.go` and `jump_table.go`) from `c9f847e03dabce8e579bfb6385801b8327625e96` substituted using a Go overlay. All three new parent tests fail, with nine failed test/subtest events. The same tests pass with the correction. The report records the immutable predecessor, original file hashes and actual negative-run digest. No reference source or independent expected value was changed to make a test pass.

## All Group B tasks reviewed

| Task | Review result |
| --- | --- |
| HGT-002 | Rechecked field order, widths, modes/flags, canonical RLP, signature-length commitments and transaction/signing hashes against the independent wire goldens. No new encoding change. |
| HGT-003 | Corrected APPROVE memory semantics, checked exact returned data and exceptional rollback. Rechecked scope/ADDRESS identity, default accounts, escrow, current nonce increment, repeated and nested approval. |
| HGT-010 | Rechecked all parameter selectors, 256-bit index/offset handling, structural versus completed-result visibility, arbitrary-only signature data and zero-padding/copy boundaries. New APPROVE tests cover the shared memory boundary. |
| HGT-011 | Rechecked independent execution/state reservations, entry costs, creation/deposit, owner refills, refund/floor and clipped fees. Corrected APPROVE's exact execution cost without changing state charges or fee formulas. |
| HGT-012 | Rechecked call/frame/batch snapshots, prior-frame state ownership, approval/payment/nonce rollback, warm sets, refunds, logs, transient clearing and child exceptions. Static SSTORE now rejects before metering; invalid APPROVE regions commit nothing. |
| HGT-013 | The fixes remain in the shared native VM used by transaction processing, builder, import and replay. Existing signed native build/import/reorg/restart tests remain mandatory; no parallel execution path was added. |
| HGT-014 | Rechecked canonical per-frame status/gas/logs, skipped/failed constraints, payer encoding, receipt root and log indices. Corrected memory gas flows through the existing canonical receipt path. |
| HGT-015 | Corrected the static SSTORE BAL entry. Rechecked existing precompile, delegation, gas-sentry, SSTORE post-state failure and SELFDESTRUCT boundaries, final net changes, system updates, corrupt commitment rejection and size/item limits. |
| HGT-016 | Rechecked canonical and historical transaction/receipt readback, FileDB recovery, replay and reorg. No wire or storage schema is changed. |
| HGT-017 | Rechecked sorted/keyed/legacy domains, access/data pricing, sequence exhaustion, atomic consumption, BAL non-warming, retry, CREATE and immutable pre-state introspection. D01 remains open under the plan's controlled-profile allowance. |
| HGT-018 | Rechecked exact ordinary deployment/runtime, domains, packed tuples, header-slot input, duplicate/cold keys, window, overwrite and native canonical fork switching. Pool grammar/trace/dependency work remains Group C. |
| HGT-019 | Rechecked protocol signature schemes, low-s/length/digest constraints, default-account selection and arbitrary-witness separation. RETURN parity tests use a code-authorized account, not implicit authentication of an arbitrary signature. |

The criterion register retains all 36 criteria and now requires 75 native test identities. The checker rejects deletion of either new APPROVE consumer, the static SSTORE consumer or EIP-214. No acceptance criterion was removed or relaxed.

## Scope and compatibility

These corrections change consensus behavior for research blocks exercising the defective paths. Previously overcharged APPROVE receipts or spurious static SSTORE BAL commitments may not replay under the corrected rules. Rebuild such research history from a valid checkpoint; this change does not rewrite roots or perform a public-chain database migration.

EIP-8250's three-fee example still conflicts with its nonce-only instruction and the current EIP-8141 two-fee budget. No new upstream resolution was found. The isolated nonce-only state/vector profile completes HGT-017 only under the development plan's explicit allowance; no public composed activation, decoder, importer or pool route is introduced.

Fresh local qualification and exact source hashes are in the A/B evidence. Hosted checks must be read against their tested commit on PR #169; the previous green head contains the two reproduced defects and is not current-head qualification.

## Fresh local qualification

At functional revision `f68ba648ed0448650cf67c73df46d6b253ff9679`, the full suite completed all 240 packages with 23,746 named passes, zero failures and 124 explicit skips. The focused race run completed all 36 packages with 7,986 named passes, zero failures and 74 explicit skips. All 75 required B tests and 62 required A tests passed. Build/vet, both CLI builds, the 12-case historical differential baseline and 75 Python tests in both interpreter modes passed. Individual skip reasons and source/log digests are retained in the evidence.
