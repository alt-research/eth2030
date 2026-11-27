# Group D latest-EIP completeness re-audit

Rechecked on 2026-11-27 (Asia/Shanghai), starting from PR #171 head
`00ca827fd8d55a21b367adc8e8f3a92334eae36e`. EIPs master remains
`c7cc5197513cd062a0fa0dc81cb66e5760228ac6`; EIP-8504/#12672 remains
`96da3c17047ad31608c40bfe75049db78dc4417d`. The latest EIP-8298,
7906, 8151 and 8288 blobs match the immutable normative register. The latest
8504 PR comments still distinguish #12714 as an alternative requiring an ACD
decision. No new normative wire assignment or aggregation parameter was found.

## Reproduced gaps and corrections

| Boundary | Normative basis | Reproduction and correction |
|---|---|---|
| ASSERT trace lifetime | EIP-7906 Gas Cost: reuse an index while the diff is unchanged; per-read work is constant after indexing | Sixteen caught static child reverts caused 17 whole-slot-table reads. Observable-mutation generations now distinguish state/log writes from unchanged snapshots. Read-only reverts and zero-value account touches retain the index. Mutating/body reverts invalidate it. Ordered snapshot lookup/suffix truncation avoids scanning all earlier calls on every revert. |
| CL witness commitment | EIP-8504 Consensus Layer and Block Header | SSZ decoding retained witness bytes but omitted the execution-header witness commitment. The 20-field schema now reconstructs it, including active-empty witnesses. Encoding rejects an active header whose witness field is missing or inconsistent. Existing independent SSZ bytes and roots are unchanged. |
| Legacy research proof boundaries | EIP-8504 Execution Does Not Verify Witnesses: a stateful/stateless/proof validity backend must enforce dependency checks or reject the block | Legacy synthetic trace/BAL generators, the old hash-only guest and both STF witness/input paths accepted dependency blocks without authenticating witnesses. They now reject the unsupported profile before returning a result or consuming the guest context. This includes valid, missing, tampered, bare and active-empty forms. The STF encoding retains transaction hashes rather than complete proof declarations/witnesses, so both synthetic and real-RV32IM STF paths reject the unsupported input before prover setup. These helpers are not promoted to actual EVM proof backends; native import still verifies witnesses before execution. |

| Generic sync delegation | EIP-8504 Block Validity and Execution Does Not Verify Witnesses | The queued, sequential and batch sync pipelines could delegate a missing/invalid witness block to a root-only executor and writer. They now enforce the independent declared-witness predicate before either side effect. The chain validator/executor still owns fork activation and ordinary validity; field presence never schedules a fork. |

The trace journal also invalidates on later observable writes/logs, preventing
stale tables if its index is materialized before a mutation. Nested rollback
and snapshot revision reuse are covered independently of the EVM consumer.

## Additional production-consumer coverage

- Actual KZG cell proofs combined with an independent leanSPHINCS witness:
  six-field raw submission, five-field pooled blob envelope inside the outer
  dependency wrapper, identity/deep-copy preservation, admission, byte budgets,
  Engine blob lookup and rejection by ordinary eth pooled messages. Missing
  witnesses/blobs, wrong versions, bad cell/signature proofs and extra fields
  fail without retaining pool metadata.
- Native creation adopts existing code and self-destructs in the same
  transaction. A later attempt to adopt from that transient child is rejected.
  ASSERT still sees the child's code/nonce before end-of-transaction deletion;
  a failing assertion restores the factory/body while retaining sender nonce
  and payment. No deleted or reverted adoption emits a BAL code change.
- Actual LOG3 output reaches TXTRACE in emission order with exact topics/data.
  EVENTDATACOPY returns the real bytes; an invalid zero-length source range
  fails the assertion and removes body logs while keeping the paid prefix.

The original 8 tasks and 32 criteria remain intact. The mandatory register now
contains 55 native consumers. Existing expected oracles, refs and root trackers
are unchanged. Full/race, source-bound evidence and final-head hosted results
are recorded separately in the PR review trackers and linked runs; prior passes
are not silently represented as current-source qualification.

The previously documented public release gates remain in force. No unresolved
aggregation, canonical-paymaster adoption, official transport/CL assignment or
cross-profile composition is invented to obtain a completion label.
