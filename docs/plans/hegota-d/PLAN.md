# Group D: normative scope and acceptance plan

Baseline: merged PR #170, master `81e63f586642a1406d56db9fa7cbf00941feb7bf`.
This plan refines HGT-030–037 without reopening the merged A/B/C work.
Project documentation and PR content are English.

## Source and composition decisions

The selected proof proposal is EIP-8504/#12672 at
`96da3c17047ad31608c40bfe75049db78dc4417d` (blob
`d5b17df22eaa7a1b705985e76d41378ed12ec712`). Its latest text specifies pooled
wrappers, raw submission, witness body/header/Engine fields and gas accounting.
The older local snapshot still contains TBD sections, so it is insufficient.
The selected file is preserved separately; historical refs/oracles are unchanged.

EIPs master is `c7cc5197513cd062a0fa0dc81cb66e5760228ac6`. EIP-8288's latest
blob `611f58a8668bbffb798c3b3b073cfaa275265f34` has two merged clarifications:
its checks extend ordinary block validity and it adds a frame mode/header/payload
fields, not a new transaction type. Its `AGGREGATED_VK` and `DEPS_HASH` remain
TBD. Recursive aggregation is consequently outside this direct-verification
profile; a successful proof-frame execution never substitutes for witness checks.

EIP-8298 (`94f01555581bef0588f71aa8520b6c289c468890`), EIP-7906
(`2eff6bf25be056fc3c3342d376865e0f684bc16b`) and EIP-8151
(`bfd0db906f55033442a064357d8e6768490b8fd5`) match the read-only refs.
The plan's historical POST_TX wording is superseded by EIP-7906's **ASSERT**
mode 3, TXTRACE 0xb6 and EVENTDATACOPY 0xb7. Proof mode is 4; SETCODEFROM is
0xa7. Neither extension changes EIP-8141's existing signature-list encoding.

The EIP-8504 author's latest PR discussion explicitly describes #12714 as an
alternative design and says the ACD choice is outstanding. These formats must
not be blended. The selected native proposal has an explicit optional fork
schedule, nil by default, with the referenced provisional constants pinned.
This does not finalize those constants or establish mainnet activation.

EIP-8504 does not allocate eth-message or Engine-method version numbers. A
native transport profile must be explicitly identified and capability-isolated,
and its code points must not collide with the existing eth transport. Public
interoperability/final activation remains gated on upstream assignment and
parameter finalization. Existing payload methods may carry the selected native
extension only under its explicit fork profile; no new official version is claimed.

SETCODEFROM and ASSERT have separate optional schedules. EIP-8151 is reused
from C. Full private-key retirement requires its additional ecRecover rule;
standalone EIP-8298 does not itself disable contract-level ECDSA authorization.
Application compensation is not invented: ASSERT failure preserves prefix
payment/nonce and reverts body token transfers, even if that costs a paymaster.

## Complete acceptance register

| Task | Criterion | Required outcome |
|---|---|---|
| HGT-030 | D30.1 | Immutable latest source/history/discussion identities and frozen direct-profile ADR. |
| HGT-030 | D30.2 | Distinguish 8504, alternative #12714 and incomplete 8288; reject unsupported schemes/aggregation. |
| HGT-030 | D30.3 | Explicit nil-default activation, prerequisites, configuration persistence, compatibility and rollback. |
| HGT-030 | D30.4 | Provisional parameters and network/Engine/CL release boundaries are explicit, without invented standards. |
| HGT-031 | D31.1 | Exact proof mode/flags/target/value/budget/batch validation; preserve base and keyed signing bytes. |
| HGT-031 | D31.2 | Exact 96-byte triples, zero-hash substitution only in derived dependencies, sort/deduplicate and block union. |
| HGT-031 | D31.3 | Exact submission/wrapper bytes, dependency/witness binding, blob composition and malformed bounds. |
| HGT-031 | D31.4 | Independent base/keyed vectors; repeated declarations charge verification repeatedly and witness bytes once. |
| HGT-032 | D32.1 | Real pinned leanSPHINCS implementation; no SHA2 substitute, stub or mock acceptance. |
| HGT-032 | D32.2 | Complete independent EthLib vectors, valid/invalid encoding and tampering in every layer. |
| HGT-032 | D32.3 | Length rejection before hashing, fixed verifier iteration/permutation bounds, exact scheme/key/message binding. |
| HGT-032 | D32.4 | No-call/no-state proof execution and receipts; intrinsic/floor/cap/public-work behavior matches the proposal. |
| HGT-033 | D33.1 | Reject missing/invalid/extra/mismatched witnesses before prefix simulation, including proof frames in the body. |
| HGT-033 | D33.2 | Stateless verification caching, one admitted wrapper per transaction, atomic replacement/removal/byte accounting. |
| HGT-033 | D33.3 | Announce/request-only capability-bound witness transport; never send dependencies as Transactions/PooledTransactions. |
| HGT-033 | D33.4 | Actual RPC/peer retrieval, invalid propagation controls, bounded requests/cache and reorg/eviction behavior. |
| HGT-034 | D34.1 | Canonical dependency union, one witness per distinct dependency, exact list commitment including the empty list. |
| HGT-034 | D34.2 | Header/body/payload/body-retrieval/persistence round trips and original bytes survive restart. |
| HGT-034 | D34.3 | Build/import enforce all witness checks, gas/size bounds and ordinary validity; invalid payloads cannot contaminate state. |
| HGT-034 | D34.4 | Native independent payload consumers and wrong/missing/extra witness/hash/key/scheme controls; no recursive-proof claim. |
| HGT-035 | D35.1 | Exact source validity, low-160 address, system/static early halt, gas/warmness and pre-state BAL boundary. |
| HGT-035 | D35.2 | Self/delegate/callcode context, loaded-frame code versus reentry/introspection, nested and outer rollback. |
| HGT-035 | D35.3 | All creation paths adopt immediately, ignore returned initcode bytes/deposit costs and reject same-transaction sources. |
| HGT-035 | D35.4 | Exact hash-only BAL changes, origin/availability resolution, persistence after source changes and no bytecode fallback. |
| HGT-036 | D36.1 | Existing 8151 raw-code class/gas/BAL/dependency checks remain intact and are reused. |
| HGT-036 | D36.2 | Code adoption retires ordinary origin/redelegation authority; ecRecover-off/on behavior is distinguished. |
| HGT-036 | D36.3 | Public SETCODEFROM ban, immediate-aware mutable-helper scan, bounded proxy-code dependencies and complete lifecycle/reorg release. |
| HGT-036 | D36.4 | Real permit/multisig/fault-proof compatibility/admission report; unsafe ecosystem assumptions remain release gates. |
| HGT-037 | D37.1 | ASSERT static suffix, default behavior, approval prohibition throughout its subtree and unconditional body rollback. |
| HGT-037 | D37.2 | Exact net pre/current balance/storage/code/nonce/event trace, ordering, pre-charge/payer and complete 256-bit checks. |
| HGT-037 | D37.3 | Exact event copying, bounds/memory/gas and no trace-induced state/access-list/BAL reads. |
| HGT-037 | D37.4 | Prefix nonce/payment preserved; body logs/state gas reverted while execution gas/status remain; token/false return/retry/batch/skipped-assert cases. |

Every criterion requires native positive/negative consumers and source-bound
evidence before completion is claimed. Full builds/vet/tests, affected race,
mandatory A/B/C regressions and exact-head hosted results remain required.
