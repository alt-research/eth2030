# Historical baseline and current prerequisite inventory

HGT-006. The mandatory runner is `pkg/cmd/eth2030-baseline`, with two unchanged official legacytests files and an explicit 12-case manifest. Each case executes native `execution.ApplyTransaction` and direct upstream Geth **v1.17.10** independently, comparing state root, logs hash and gas against fixed expected values and each other. Neither executor supplies the other's oracle. The ETH2030 Geth wrapper is not the reference.

The corpus covers seven older-fork cases (Homestead, EIP150, EIP158, Byzantium, Constantinople, ConstantinopleFix and Frontier) and five later cases (Berlin, London, Paris, Shanghai and Cancun). The exact executed records in [baseline.json](../../../evidence/hegota-a/baseline.json) are authoritative. Twelve passing cases are a minimum historical baseline, not full historical, Prague, Glamsterdam or Hegota conformance.

The runner rejects missing/tampered inputs, empty corpora, missing/zero roots or logs, unknown fields/forks, invalid indices, changed case counts and unsupported expected exceptions. It does not silently count skips, panics or arbitrary exceptions as passes. `TestLockedNativeReferenceBaseline` and `TestBaselineRejectsInvalidOrUnsupportedEvidence` execute positive and negative contracts. The original upstream all-zero oracle is retained only as rejected evidence.

## Reassessment after integration

Every prerequisite in [conformance.json](../../../specs/hegota/conformance.json) now has a current assessment: fixed integrated baseline, actual source files, observed behavior, concrete executed tests and remaining qualification scope. Historical findings are retained separately. Code presence and passing bounded tests do not close every selected-spec rule.

| Requirement | Current implementation and evidence | Remaining consumer scope |
| --- | --- | --- |
| 7732 / ePBS | Legacy unauthenticated bids fail closed; native Bogota wire/deferred persistence exists | Independent CL registry/domain/signing root and admitted ePBS composition, HGT-040/070/080 |
| 7928 / BAL | Frame state journal/BAL and validated storage/corruption controls exist | Full published serialization and independent system-transition replay, HGT-015 |
| 7685 / requests | Receipt deposits, withdrawal/consolidation queues and Amsterdam builder queues are collected in order; rollback tested | Conditional PQ request and independent CL admission rules, HGT-075/076 |
| 6110 / deposits | **Historical missing-log-path finding resolved**: `collectRequests` calls `DepositRequestData(receipts, address)`; malformed matching events reject | Independent selected-network deposit/CL agreement, HGT-081 |
| 8037 / state gas | Explicit canonical Frame pools, ownership, refill and atomic rollback execute; legacy reservoir is separate | Full independent parameter/creation/refill matrix, HGT-011/012 |
| 8038 / pricing | Shared fork setup and canonical account/storage charges; nonzero block-height regression | Full independent warm/cold/access/delegation matrix, HGT-011/020 |
| 2780 / intrinsic/value | Existing intrinsic, sender creation, value and code-deposit paths | All selected-profile existence/value boundaries, HGT-011/020 |
| 7778 / refund | Pre-refund block execution and net state capacity are separate from post-refund payment | Exhaustive multi-transaction refund/floor/capacity combinations, HGT-011/015 |
| 7976 / floor | Canonical frame/signature budget and clipped-tip settlement | Large-data/signature boundary differential qualification, HGT-011/020 |
| 7825 / cap | Canonical execution cap and separate state reservation exist | Exact boundaries at every ingress/build/import route, HGT-002/011/015 |
| 7934 / size | Validator measures full block RLP against the active execution-layer bound | Prefix-growth and omitted-transaction-fit boundary oracle, HGT-015/044 |
| 7843 / slot | Header slot requirement, shared VM slot setup and Engine/restart commitment exist | Independent CL skipped slots and conditional root pool rules, HGT-018/042/070 |
| 7702 / delegation | Native delegated Frame approval and nested rollback execute | Conditional 8151/8298 adoption/migration/fault-proof composition, HGT-019/035/036 |
| 7708 / logs | Native transfer/log and atomic rollback paths exist | Full selected-profile creation/approval/receipt-root combinations, HGT-011/012/014 |
| 8141 / conversion | **Historical reconstruction finding resolved**: canonical typed envelope dispatch bypasses legacy Message reconstruction | Independent width/signature/blob differential coverage, HGT-002/019/081 |
| 3675 / forkchoice | **Missing native head propagation resolved**: actual FCU selects/rewinds state and refreshes pool | Independent composed-fork CL interoperability, HGT-070/080 |

This inventory is complete as an A deliverable. It neither opens duplicate implementation tasks for code already present nor claims that later qualification/release tasks are complete. Each current assessment's tests must appear as actual passes in source-bound verification; missing/skipped tests cannot satisfy this evidence.

## Builds, fixtures and skips

[Verification](../../../evidence/hegota-a/verification.json) binds full-suite, focused race, build/vet, Python checks and the 12-case baseline to current source fingerprints. [Skipped tests](../../../evidence/hegota-a/skipped-tests.json) enumerate fresh full/race skips separately. Baseline skips must be zero. Test/subtest pass events are labeled as such, not inflated into independent fixture counts. Prior runs remain at immutable revisions, including [the original historical baseline report](https://github.com/alt-research/eth2030/blob/6a17479/docs/plans/hegota-a/baseline-and-gaps.md).
