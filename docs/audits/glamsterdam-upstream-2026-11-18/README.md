# Glamsterdam upstream discussions and remaining work — 2026-11-18

Development tracking: [Glamsterdam completion program](../../plans/glamsterdam/README.md). This audit remains the dated evidence snapshot.

This follow-up reviews current Ethereum planning, EIP pull-request discussions, consensus changes and execution API decisions against ETH2030. It supplements and corrects the [first readiness audit](../glamsterdam-2026-11-18/README.md). **Glamsterdam remains incomplete, and implementing today's EIP text alone would not close acceptance: both unresolved upstream design choices and confirmed local production defects remain.**

## Evidence boundary

Local baseline: `feat/track2-hegota` at `7f1c5537` (protocol code unchanged from `83e3860d`). EIPs were read at upstream commit `3db38a3725105cb67b8804ac3e147967113bc3ca`; the latest observed consensus-specs commit was `2ea9d96a420927eb1a35d7c9399c8b3691c69f6a`. Local reference submodules were not updated. Selected sibling Lighthouse source was read at `f85fa05b6`, without modifying, building or running it.

[discussion-index.json](discussion-index.json) records 27 fetched PRs, seven fetched issues, six comment threads, PR head revisions and observed open/merged states. Search inventories covered EIPs 8037/7928/7773, Gloas, Amsterdam execution APIs and executable-spec invariants. The connector has separate issue and PR searches; initial issue-only searches were replaced with PR searches, not treated as evidence of absence. Search results were leads: substantive findings use fetched PR bodies, diffs, review comments and closure/merge states. This is a targeted review, not an exhaustive review of every Ethereum discussion. A closed issue or a closed, unmerged PR is not itself proof that a change shipped.

Comments are stakeholder reports unless independently confirmed here. Some comment dates are not supplied by the connector; PR head revisions and retrieval time are retained instead. First-party call summaries were read, but full recordings and auditors' entire reports were not independently replayed/reproduced. No email, GitHub comment or other external message was sent.

## 1. The current plan has an unresolved compatibility gate

The [EF update, including its November 9 update](https://blog.ethereum.org/2026/11/06/glamsterdam-8037-8038), says Hoodi will not fork until application compatibility concerns are resolved; potential gas-spec revisions are being evaluated. The [November 16 ACDT agenda and discussion](https://github.com/ethereum/pm/issues/2299) solicited audit findings and application examples. The [first-party call decision summary](https://forkcast.org/artifacts/acdt/2026-11-16_103/key_decisions.json) records no 8037 decision on that call: the formal path forward was deferred to [ACDE #249 on November 19](https://github.com/ethereum/pm/issues/2285). That call is still in the future at this audit's cutoff. No final gas design or activation date should be inferred from its agenda.

The [official upgrade plan](https://github.com/ethereum/pm/blob/master/glamsterdam-mainnet-plan.md) lists Hoodi and mainnet as TBD. EIP-7773 still contains the old Hoodi activation row; [EIPs #12568](https://github.com/ethereum/EIPs/pull/12568), which removes it, is open. This is a concrete example of why a meta-EIP table alone does not establish the current rollout plan. January 2027 is discussed in the call summary as a target/signal, not an announced mainnet date.

The current 18-item core list also has implementation dependencies outside the literal count: [EIP-7688](https://github.com/ethereum/EIPs/blob/3db38a3725105cb67b8804ac3e147967113bc3ca/EIPS/eip-7688.md) requires 7495 and 7916. [#12449](https://github.com/ethereum/EIPs/pull/12449) proposes listing them explicitly in the meta-EIP and has an author review explaining they should have been listed with 7688. Do not count an SSZ helper as completed CL conversion: production container roots, gindices, proofs, runtime limits, network size bounds and fork migration still need verification.

## 2. Gas work must be expressed as compatibility obligations

The [official gas-invariants repository](https://github.com/ethereum/glamsterdam-gas-invariants) identifies eight assumptions involving failed-call handling, immediate state-gas refills, revert credits, fixed gas budgets, `gasleft()` accounting, the remaining 1/64, fixed reserves/stipends and 7702 authorization rollback/reuse. These are application control-flow and off-chain accounting concerns, not just price changes.

Our old 25% reservoir heuristic is wrong even against today's merged text. However, the first audit's recommendation to implement current LIFO refills is only a **versioned current-text obligation**, not a settled final design:

| Discussion | Observed state | Consequence for our backlog |
| --- | --- | --- |
| [EIPs #12581](https://github.com/ethereum/EIPs/pull/12581) | Open, unmerged; head `2ba9176f` | Proposes reservoir-only refills and a legacy execution-gas share for new state, removing current LIFO bookkeeping. Need a design decision and pinned vectors before claiming final compatibility. |
| [#12581 discussion draft](https://github.com/ethereum/EIPs/pull/12581#issuecomment-6412891161) | Author explicitly says draft not pushed to PR | A further proposal delays some rolled-back state-gas credits until settlement, so a failed callee cannot fund its caller's failure handler. Do not confuse the comment's counter model with either merged text or PR head. |
| [EIPs #12582](https://github.com/ethereum/EIPs/pull/12582) | Open, unmerged; stacks on #12581 | Proposes per-call-frame state-gas allowances. A small execution budget currently does not bound use of the shared reservoir. If chosen, nested factories, multisigs, 4337 verification and 7702 relayers require new limits and tests. It is not an unconditional scheduled requirement. |
| [EIPs #12624](https://github.com/ethereum/EIPs/pull/12624) | Open, unmerged | Calls out contradictions involving creation collision with nonempty storage, start/end system calls, surviving balances after SELFDESTRUCT and intrinsic/runtime charging. Trace each normative dependency against EELS and final EIPs; do not implement stale explanatory prose literally. |
| [Temporary-contract refill discussion](https://github.com/ethereum/pm/issues/2299#issuecomment-6917033995) | Stakeholder concern, not an adopted rule | CREATE plus same-transaction SELFDESTRUCT affects temporary deposit addresses and short-lived execution. Add this case to compatibility qualification; do not assume a future TCREATE proposal repairs existing addresses or bytecode. |

Required test matrix: funded/unfunded/exhausted reservoir; every relevant CALL/CREATE family; success/revert/exceptional halt; cold/warm and first/repeated writes; state cleared/restored; pre-execution vs post-entry 7702 failure; fixed signed gas budgets; failure caught by caller; and both execution/state block limits, floor, fees and receipts. Use actual consumer outcomes, not only total gas or transaction success. Estimates can find a cheap successful outer transaction whose intended inner call failed.

The [application developer comment and Juicebox PoCs](https://github.com/ethereum/pm/issues/2299#issuecomment-6860064283) are an input to this acceptance matrix, not evidence that our implementation reproduced their impact. Deployed immutable application compatibility remains an upstream ecosystem gate even if all clients compute identical roots.

## 3. New confirmed P0: BAL does not bind the native block hash

`pkg/core/types/Header` has `BlockAccessListHash`, but `header_rlp.go` never encodes or decodes it. `Header.Hash()` hashes that RLP. The [local probe](probe.go), using independent headers to avoid cached-hash mutation, observes:

```text
different_BAL_commitments_same_header_hash=true
different_BAL_commitments_same_header_RLP=true
```

The [current EIP-7928](https://github.com/ethereum/EIPs/blob/3db38a3725105cb67b8804ac3e147967113bc3ca/EIPS/eip-7928.md) adds the field to the header. Comparing a supplied field against locally computed BAL is useful but cannot replace including it in the canonical block hash. This correction lowers 7928 from the previous audit's broad “substantial foundation” category to **confirmed consensus-format mismatch**.

Finish the header shape, RLP order, decoding, copy/persistence, block hashes, Engine conversion and cross-client vectors. Test that changing the BAL commitment changes the header/block hash; test real post-fork blocks against an independent client. No network exploit or complete signed-payload reproduction was attempted here.

## 4. New confirmed P0: SLOTNUM is a disconnected field/opcode path

The first audit overstated header support. `core/types/header.go` has no native slot field; native RLP cannot include it. `engine/payload` carries a slot separately and the SSZ projection takes one from its context, but native execution's `BlockContext` construction in `core/execution/processor.go:1193` never supplies `SlotNumber`. The RPC backend omits it as well. The probe's native ApplyMessage executes SLOTNUM successfully and returns zero; this is not a live beacon-slot measurement, but it confirms the zero-initialized execution context identified by source inspection.

[EIP-7843](https://github.com/ethereum/EIPs/blob/3db38a3725105cb67b8804ac3e147967113bc3ca/EIPS/eip-7843.md) already requires the slot in the header. Open [#12621](https://github.com/ethereum/EIPs/pull/12621) additionally clarifies its order after the BAL hash and CL validation against the beacon slot. The open clarification is not the reason the existing header requirement applies.

Finish the complete route: CL slot → attributes → block header → RLP/hash/storage/replay → EVM → RPC readback. EL must preserve the value; validating it against the beacon slot is a CL responsibility. Opcode tests which manually set BlockContext.SlotNumber do not exercise this route.

## 5. BAL availability is an Engine consumer obligation

Merged [EIPs #12347](https://github.com/ethereum/EIPs/pull/12347) requires BAL availability through `engine_getPayloadBodiesBy*V2` for at least **14,299 epochs**. Its discussion explains why: a CL storing blinded payloads must reconstruct complete payloads during its serving window. It permits regeneration rather than prescribing a particular storage implementation; serving cost, nonfinality and abuse protection still matter. Open [#12546](https://github.com/ethereum/EIPs/pull/12546) aligns 8159 wording with this merged change.

Confirmed local gaps:

- `core/rawdb/history.go:20` still sets 3,533 epochs, and `engine/backend.go:443,515` applies that shorter window to payload-body lookup. The helper uses block numbers as an approximation for slots; missed slots also require a real epoch/time retention design.
- Production `node/backend/engine_payload.go:261,275` calls `payload.BlockToPayloadBodyV2`, which fills transactions and withdrawals only. The probe serializes a synthetic block with a BAL commitment and observes `{"transactions":[],"withdrawals":[]}`: no BAL field is populated.
- The alternative in-memory Engine backend does populate a field, but uses `json.Marshal(blockBAL)` as bytes, rather than the canonical BAL RLP. The body type's plain `[]byte` JSON field also requires wire-format review. Successful producer output does not establish historical V2 body correctness.

Acceptance must cover ByHash and ByRange through the actual node/HTTP route, canonical BAL bytes matching the committed header, restart, retention boundaries, unknown vs unavailable data, orphaned blocks and CL rehydration. Reusing the producer's in-memory BAL during one run is insufficient.

Some upstream discussion identifies necessary tests rather than new missing features. Merged [#12571](https://github.com/ethereum/EIPs/pull/12571) pins minimal integer RLP; our `bal/hash.go` already uses big.Int scalar slots/values and removes read entries for slots with writes, which is positive evidence. Do not count those encodings as absent. Open [#12620](https://github.com/ethereum/EIPs/pull/12620), [#12611](https://github.com/ethereum/EIPs/pull/12611) and [#12612](https://github.com/ethereum/EIPs/pull/12612) motivate further differential cases: net changes after finalization, 7702 no-op code, failed CREATE nonce changes, repeated withdrawals, static-context halts before state access, and repriced pre-state charging. Each needs local execution evidence before being called a local defect.

## 6. ePBS needs adversarial and fork-transition acceptance

### Payload authentication is not finalized

[consensus-specs #5770](https://github.com/ethereum/consensus-specs/pull/5770) and [#5820](https://github.com/ethereum/consensus-specs/pull/5820) are both open. They address an envelope with a claimed bid-matching block hash whose actual contents differ. #5770 computes the execution block hash locally in CL gossip validation; the discussion covers RLP/MPT complexity, empty transaction entries, validation cost and avoiding a blocking newPayload call on the gossip hot path. #5820 instead commits to payload_root in the bid, removes the envelope signature and builder index, and authenticates content by that root. As of head `7af0af3c`, its author has removed an earlier parent_payload_root field. These are alternatives, not two cumulative requirements.

The [ACDT comment](https://github.com/ethereum/pm/issues/2299#issuecomment-6831513586) explicitly asks for a choice to unblock a spec release. #5820's [review](https://github.com/ethereum/consensus-specs/pull/5820#discussion_r4602321894) also found tests that failed on the new root mismatch before reaching the intended randao/gas/hash check; the [author says those tests were fixed](https://github.com/ethereum/consensus-specs/pull/5820#issuecomment-6961891493). Pin the chosen head, isolate negative assertions and use behavior mutations to verify each rejection. Merely checking that a mutated payload is rejected is insufficient.

Our native `consensus/epbs_transition.go` compares envelope fields, while `fork_choice_ptc.go` explicitly retains pre-Gloas root-chain weight propagation and tracks payload presence as a block-root boolean. It does not establish the complete current FULL/EMPTY/PENDING weighted fork-choice and payload-authentication specification. Actual deployment uses Lighthouse, which must be reviewed/tested separately; native helper tests do not certify it.

### Sync and transition obligations

| Upstream item | Observed status | Required local follow-up |
| --- | --- | --- |
| [#5781 optimistic sync](https://github.com/ethereum/consensus-specs/pull/5781), [#5818 vectors](https://github.com/ethereum/consensus-specs/pull/5818) | Both open | Validate per-payload optimistic state, invalidation of FULL descendants while preserving valid EMPTY variants, latestValidHash across gaps/branches, restart and FCU on an EMPTY head. The proposed 24-case suite is a test candidate, not a merged guarantee. |
| [#5808 first-slot availability](https://github.com/ethereum/consensus-specs/pull/5808) | Merged | First Gloas payload must not start as already available. The inspected local Lighthouse upgrade initializes every bit true and contains no later clear in that file: a concrete source-level lag requiring regression verification. |
| [#5822 bid signature domain](https://github.com/ethereum/consensus-specs/pull/5822) | Merged | Sign first-fork bids from a pre-fork head using the bid-slot fork version. Verify deployed builder and CL, rather than assuming native APIs with caller-supplied forkVersion select it correctly. |
| [#5816 gossip domain](https://github.com/ethereum/consensus-specs/pull/5816) | Open | Track message-epoch domain selection for first-fork blocks/attestations while head is still pre-fork. Separate proposed text from established message-domain requirements. |
| [#5736 builder onboarding](https://github.com/ethereum/consensus-specs/pull/5736) | Merged after an earlier close/reopen discussion | Separate validator and builder registries for shared keys, invalid signatures and deposit ordering. Old comments saying it was deferred are no longer its final status. |
| [#5836 onboarding across branches](https://github.com/ethereum/consensus-specs/pull/5836) | Open | A pending finalized deposit might already be applied as a validator on another branch. Track proposed eligibility restriction and test validator-index consistency; this is not covered by depositing once after genesis. |
| [#5835 cell-delta group version](https://github.com/ethereum/consensus-specs/pull/5835) | Merged | Gloas partial-message group version byte is 0x01. Test actual CL-to-CL partial message exchange; complete-column tests do not cover this. |
| [#5754 payments after missed epochs](https://github.com/ethereum/consensus-specs/issues/5754) | Open, substantive follow-up discussion | Test payment settlement once, slashed proposer, absent quorum, late FULL child and builder balance/withdrawals together. A later comment alleges additional duplicate-payment variants; these were not reproduced here and must not be presented as confirmed local exploits. |

## 7. RPC tooling is part of completion

Merged [execution-apis #950](https://github.com/ethereum/execution-apis/pull/950) resolves [#946](https://github.com/ethereum/execution-apis/issues/946): eth_createAccessList returns the actual accessed addresses/storage keys, not whichever of the discovered and empty lists happens to be cheaper. `rpc/ethapi/eth_api.go:588` still explicitly returns an empty list as a placeholder. This is a confirmed missing consumer feature. The historical header issue additionally prevents reliable SLOTNUM simulation/replay.

Open [#977](https://github.com/ethereum/execution-apis/issues/977) distinguishes transaction gas-cap/reservoir semantics for estimates and access-list creation from eth_call/debug_traceCall and validation-disabled simulations. Its client table is explicitly source-based. Our production RPCBackend.EVMCall creates an EVM and calls it directly, bypassing intrinsic transaction processing and reservoir setup; estimateGas and access-list handlers reuse it. Reconcile these methods with the adopted rule instead of applying one execution mode indiscriminately.

Open [#852](https://github.com/ethereum/execution-apis/pull/852) and [#976](https://github.com/ethereum/execution-apis/issues/976) address trace breakdowns of regular/state gas, refunds and reservoir entry values. Nested gasUsed depends on reservoir funding, and cannot safely be treated as execution gas or added to receipt gas without understanding the dimensions. No matching stateGasUsed/stateGasReservoir/regularGasUsed trace fields were found in our RPC sources. Treat the exact schema as pending; establish consistent accounting and consumer tests now.

## 8. Verification must gain independent invariants

[execution-specs #3948](https://github.com/ethereum/execution-specs/pull/3948) is open and proposes a deployed-idiom corpus validated with targeted mutations. Its tests deliberately distinguish old fork assumptions from current-fork behavior; all-green does not mean historical invariants were preserved. [#3982](https://github.com/ethereum/execution-specs/pull/3982) is merged and adds chain invariant checking during fixture filling, with violations failing the run. Its review discusses enabling checks in release generation, not just a nonbinding warning job.

Our acceptance needs official native-path vectors, compatibility scenarios, conservation/accounting checks and precise failure assertions. Do not credit geth-backed fixture execution as native-engine conformance. Upgrade tests must seed nonzero fields and exercise missing/late blocks, not only regular transitions with defaults.

Read discussion replies before creating work:

- [RANDAO issue #5826 and rebuttal](https://github.com/ethereum/consensus-specs/issues/5826#issuecomment-6871879332): epoch transitions copy the mix forward and bid validation precedes the block's new reveal. Different epoch expressions need not yield different mixes. No local bug is claimed from the issue title; retain an epoch-boundary regression.
- [SSZ cross-type-root issue #5823](https://github.com/ethereum/consensus-specs/issues/5823) explicitly says its single-field container/list case is not a live Gloas exploit and relies on a future ambiguous type context. It is a design note, not a release blocker by itself.
- [PTC duplicate-validator issue #5825](https://github.com/ethereum/consensus-specs/issues/5825) is closed; [#5827](https://github.com/ethereum/consensus-specs/pull/5827) is closed but unmerged. Keep duplicate-member aggregation cases, but do not infer a particular merged fix from closure alone.

## 9. Updated completion assessment

The previous eight confirmed mismatches and missing 8246/8282 work remain. This pass adds **two confirmed core mismatches (7928 header commitment and 7843 slot propagation)**, plus concrete BAL-serving/retention and RPC-access-list gaps. The 18-core matrix is now **10 mismatches, two missing EL paths, three CL acceptance boundaries and three foundations** (7732, 7976, 8024). “Foundation” is not completion. SSZ dependencies, supporting network protocols, RPC behavior and adverse-case release qualification remain additional scope.

Recommended acceptance order:

1. **P0 — restore consensus commitments:** canonical BAL/slot header RLP and hashing, actual slot execution/replay, independently checked post-fork blocks.
2. **P0 — gas/state semantic model:** preserve a versioned merged-text baseline, follow the 8037/8038 decision, then implement and qualify the adopted refill/call-boundary model together with 2780/7778/7981 and 7702 runtime rollback. Do not silently ship either open PR as mainnet consensus.
3. **P0 — close known state/request defects:** 7708/7954/7997/8246/8282, canonical factory/precompiles, system contracts and independently received requests commitments.
4. **P1 — real consumers and recovery:** BAL availability, canonical historical bodies, RPC access discovery/estimation/tracing, negotiated eth/70–72 and snap/2 where promised, persistence/restart/reorg.
5. **P1 — actual CL/builder closure:** chosen payload authentication, FULL/EMPTY optimistic invalidation, onboarding, first-fork availability/domains, payment-once behavior, 7688/7495/7916 proofs and cell-delta interop.
6. **Release gate — current native vectors, compatibility corpus, exact negative checks and multi-client adverse-case runs**, with frozen client/spec revisions and effective fork flags.

The earlier 12.5–24 engineer-week range is an implementation planning baseline, not a delivery commitment. It did not establish upstream design freeze or application-compatibility clearance. This review adds confirmed work and explicit external gates; it does not justify a more precise date or percentage.

Later planning should not be mixed into the Glamsterdam count. The [November 12 ACDC decisions](https://forkcast.org/artifacts/acdc/2026-11-12_191/key_decisions.json) record PQ-oriented reprioritization, EIP-8483 proposed for Hegotá and Quick Slots withdrawn/declined. Those are roadmap signals, not permission to treat every future feature as Glamsterdam acceptance or to certify existing FOCIL/frame/PQ code against the eventual next-fork scope.

## Verification performed in this pass

```sh
# From pkg/
go run ../docs/audits/glamsterdam-upstream-2026-11-18/probe.go
go test -count=1 ./core/types ./core/rawdb ./engine/payload ./node/backend ./rpc/ethapi
go build ./...
```

The probe completed; [output](probe-output.txt) records the hash, slot, retention and body-conversion findings. The five focused package suites passed; [output](package-tests.txt) is retained. These results demonstrate why green existing tests do not close these gaps. The first audit's tagged-suite failure remains unresolved. No full-suite rerun, fresh devnet, independent client execution, contract-impact reproduction or Lighthouse build was performed. No protocol implementation was changed.
