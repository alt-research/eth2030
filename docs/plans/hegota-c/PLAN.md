# Hegota Group C: specification-correct integration plan

Reviewed 2026-11-26 against merged project master `5d777ad06f886c8b20fb010a47e65ddb05df8287` (PR #169). This is the detailed acceptance addendum to Group C of [the development plan](../hegota-development-plan-2026-11.md). Its HGT-020–029 scope supersedes shorthand completion assumptions in the original table; no Group D–J work is implicitly adopted.

## Review conclusion

The original ten-task outline is directionally consistent with the EIPs, but insufficient to establish complete conformance. It omits concrete prefix combinations, top-level permissions, exact context and dependency invalidation, atomic reservation/resource failures, blob transport restrictions and a defined Frame simulation contract. Existing helpers and successful Engine inclusion-list tests do not establish public-pool propagation or ordinary-pool builder integration.

Implement only rules belonging to the selected profile. Public-pool rejection must not become block invalidity. Node-configured fee increments, limits, expiry margins, retry bounds and derived RPC fields are client policy/API contracts, not new consensus rules.

## Sources and authority

- EIPs revision `ade9e9cab7e148bba4cd293f916a4a189cd7a8aa`: read EIP-8141 payload/signatures, behavior, mempool, networking and referenced `assets/eip-8141/CanonicalPaymaster.sol`; EIP-8250 payload/state/introspection/activation/mempool; EIP-8272 current slot, contract, public handling and test matrix; dependencies 2718/2929/3651/3607/7702/7843/7594/8037/8038/1474 as consumed.
- Consensus-specs revision `d2021d2719235f0df4726a1d330502cb15aa8ea3`; actual Engine/header slot input is reused. No timestamp-derived slot or new CL rule is introduced.
- [EELS tracker #2829](https://github.com/ethereum/execution-specs/issues/2829) distinguishes implementation status from normative authority. An unchecked tracker item is not a replacement specification.
- [EIPs PR #12041](https://github.com/ethereum/EIPs/pull/12041), head `4a98fcefa5c6f02a25289bf55b9d92902d02b4c6`, remains open: a proposed 355-byte paymaster with different administration/authentication semantics is not silently adopted.
- Merged [PR #12061](https://github.com/ethereum/EIPs/pull/12061) explains derived receipt status, but the current merged EIP text remains authoritative; its PR description does not itself add a consensus status field. Closed/unmerged envelope proposals #12214/#12421 are not implemented through this work.

Recheck source identities before publishing. Reference contents remain read-only. Any changed normative source must get a dated impact record; conditional proposals do not enter the public path from helper presence.

## Explicit unresolved/profile boundaries

1. **D01, keyed nonces:** EIP-8250 says to replace only nonce fields but retains a stale three-fee example conflicting with EIP-8141's two-fee budget. Preserve B's isolated nonce-only delta. Implement/test selected-key identity, one pending transaction per sender, dependencies and replacement in an explicitly isolated policy consumer. Public decoding, gossip, activation or database migration requires an authoritative combined schema. An isolated consumer does not qualify public composed interoperability.
2. **Canonical paymaster recognition:** the merged Solidity asset uses a calldata signature over the hash covering that calldata and contains nonstandard inline `verbatim` bindings. It supplies no frozen compiled runtime/settings. ETH2030 paymaster v1 changes witness handling and is explicitly local. An official canonical-code exception cannot be claimed for those bytes. Keep exact code recognition/versioning mechanically separate from generic/default-code sponsorship and from any expressly selected research template. PR #12041 is tracked as an alternative, not merged authority. Do not rewrite the merged reference to manufacture equivalence.
3. **Noncanonical paymasters:** the canonical subsection contains a restrictive sentence, while Policy Summary, Paymasters, Non-canonical paymaster, Replacement and Eviction and rationale explicitly admit noncanonical paymasters. Apply the specific noncanonical rules (generic trace rules, cap one, maximum-fee exposure), without extending the canonical trace exception. Document this textual inconsistency rather than deleting the specific rules.
4. **RPC/simulation:** EIP-8141 specifies consensus data and requires per-frame/dimensional estimates, but does not standardize a complete JSON simulation schema. Freeze a documented ETH2030 API profile before implementing it. A type-6 call must never fall through ordinary CALL parsing. A scalar gas result is not a substitute for per-frame execution/state budgets. Signed payload fields cannot be mutated in a gas search while claiming unchanged authentication or signing hash.
5. **Optional VM instructions:** allowing SETDELEGATE in a deploy trace does not activate unadmitted EIP-7819. Test allowed trace behavior only when the selected VM/profile supplies the instruction; no extra opcode/fork adoption is implied.

These boundaries are tracked separately from implemented mechanisms and executed tests. No completion report may present an unresolved public activation or unspecified canonical template as interoperable completion.

## Task acceptance register

### HGT-020: validation prefix and trace

- **C20.1 grammar:** enumerate all four base prefix shapes with none, expiry, recent-root, and expiry+recent-root verifiers (16 combinations). Reject reversed/duplicate/misplaced verifiers, bad tuple lengths, unknown flags, non-VERIFY account validation, mismatched resolved sender, atomic prefix and VERIFY after payment. Classify protocol frame shape before reading sender state. Reject expired deadlines. Keep block-valid private transactions separately testable.
- **C20.2 work and stop:** signature validation cost plus declared prefix execution limits must be at most 100000; state limits at most 500000. Exercise equality and +1, including 16 cold root keys and default-account creation. Execute through successful completion of the payer-setting frame, never the body. Failed/caught banned validation behavior cannot be hidden by an outer success. Reverted approval does not terminate validation.
- **C20.3 trace:** enforce storage ownership, code/existence/delegation constraints, zero-value helper calls, banned opcode rules, GAS immediately followed by CALL-family instructions, deployment only at sender and nonempty deployed code. Restrictions apply at all call depths. Entry dispatch of the sender and the permitted default-code behavior are not interchangeable with generic third-party calls.
- **C20.4 exact exceptions:** expiry TIMESTAMP only in its actual protocol frame/runtime context. Recent-root SLOTNUM and tuple-derived SLOAD only at top level, exact address and exact runtime, no nested permission. Canonical-paymaster exception requires an admitted exact template; generic noncanonical paymasters receive no trace exemption. Optimizations must match normal gas/receipts/warming and are optional.

### HGT-021: dependencies and revalidation

- **C21.1 capture:** record sender nonce/code/balance/existence/storage, helper/factory/delegation code, payer balance/code, canonical withdrawal state, expiry runtime/deadline and all recent-root storage/code/activation/expiry dependencies. Record only the validation prefix. Duplicates must not multiply dependency or reservation entries.
- **C21.2 index:** maintain reverse dependency indices and cached pre-state values. Re-simulate affected transactions; unchanged unrelated dependencies must not invoke the VM again. Base/blob fees, dimensional block limits, fork state and time/slot transitions still require their applicable cheap validity checks.
- **C21.3 canonical transitions:** remove included transactions and release resources; compare actual new canonical state and handle code adoption. Reorg tests must remove a dependency through the old branch without a replacement write on the new branch. Never infer absence of changes solely from the added branch's BAL.
- **C21.4 lifecycle:** replace/remove dependency metadata atomically with transactions. Fork rollback, failed admission, resource rejection, reset, expiry and reorg must leave no orphan entries, cached acceptance or stale signer/nonce assumptions.

### HGT-022: keyed/recent-root public policy

- **C22.1 roots:** enforce exact verifier grammar/runtime and tuple predicates; duplicates are allowed, with distinct dependency records. Cover 1/16/17 tuples, malformed lengths and 8191/8192 boundaries.
- **C22.2 context:** admission SLOTNUM is canonical head slot+1, checked for overflow; it does not tick through skipped empty slots. Payload inclusion uses the exact proposed header slot and timestamp. Root-age eviction and optional local admission margin are explicit; a timestamp-derived slot is forbidden.
- **C22.3 dependency lifecycle:** expiry queue and reverse indices are updated on replacement/removal. Revalidate complete age/storage predicates on relevant changes, lost deployment, code change, activation rollback and long reorg. All 16 prefix shapes remain bounded by the shared validation budget.
- **C22.4 isolated keyed policy:** canonical key sets/sequence identity, max one pending per sender even for disjoint keys, overlapping/nonoverlapping pre-state changes, MAX exhaustion and optional pre-state legacy-nonce dependency. No public route or envelope activation bypasses D01.

### HGT-023: payer exposure and paymasters

- **C23.1 all payers:** reserve max_fee, not GASPRICE-derived cost or current max_cost, for every payer including sender and default-code sponsors. Shared payer exposure must never exceed available funds.
- **C23.2 kinds:** distinguish self-payment, empty-code sponsor, generic noncanonical code paymaster and admitted exact canonical template. Apply cap one only to the relevant noncanonical PAY contract; an empty-code sponsor is not a paymaster. Subtract an entire canonical pending withdrawal once per payer, including matured/unexecuted requests.
- **C23.3 atomic transitions:** validate prospective replacement against exposure with the old reservation removed; move payer reservations and counters only on successful admission. A failed fee bump, replacement, capacity check, sidecar verification or state read cannot retain prospective metadata or drop the old transaction.
- **C23.4 revalidation:** deterministic insolvency handling, withdrawal/code changes, replacement to a different payer, inclusion, eviction, reset, fork rollback and reorg all preserve exact exposure/counts. The local experimental template remains explicitly identified and does not become upstream-standard bytecode through tests.

### HGT-024: ranking, replacement and bounds

- **C24.1 fees:** rank using the including context's actual base and blob base fees and a documented effective-priority policy under the total fee budget. Do not substitute blob base fee 1 on an active head. No fabricated per-gas/per-blob cap is introduced for canonical Frames.
- **C24.2 replacement:** same sender/nonce identity (or isolated selected-key identity), both max_fee and priority increment, configured rounding/zero boundaries and any explicitly configured larger blob increment. Cross-format replacements cannot bypass the canonical identity/policy rules.
- **C24.3 limits:** independent execution/state/blob capacity, per-sender count, pool count and total bytes including sidecars. Oversized public admission is a local limit, not block invalidity. Reject before persistent metadata/counters grow without bound.
- **C24.4 eviction:** invalid first, then nearest expiry, then lowest effective priority with deterministic tie breaks. Remove exposure/dependencies and suppress stale queued announcements. Re-admission requires receipt again. Bound unpaid builder validation attempts with a documented local retry policy that permits more than one transient failure.

### HGT-025: P2P transport

- **C25.1 plain:** preserve exact canonical typed bytes in supported nonblob Transactions, pooled retrieval and BlockBodies; hash/type/size advertisements and decoded transaction identity agree.
- **C25.2 blobs:** never full-broadcast blob Frames in Transactions. Announce/fetch on demand using the exact EIP-7594 v1 wrapper; check body/counts/version/lengths/versioned hashes and all cell proofs before pool admission. Sidecar metadata cannot bypass replacement or byte limits.
- **C25.3 receipts:** wire receipts carry [type,cumulative,payer,frame-results] in the fork-carrying negotiated protocol; preserve per-frame status/gas/logs. Do not confuse typed consensus byte strings with the structured network receipt tuple.
- **C25.4 peers and limits:** use actual negotiated protocol/fork capabilities; unsupported peers, unknown/duplicate/oversized announcements, wrong sizes/types, invalid proofs, request limits and batching are tested through real protocol handlers. No unassigned protocol version is invented as an EIP requirement.

### HGT-026: builder selection

- **C26.1 context:** ordinary pool selection uses the same candidate state snapshot and exact payload slot/time/fee context as execution. Root/expiry/public-policy admission is rechecked as required; consensus validity of private/IL input remains separate from public propagation.
- **C26.2 resources:** reserve each dimension independently, update running state between selected transactions and reject invalid state without contaminating payload roots, receipts, BAL or remaining capacity. Use actual blob fees in ordering.
- **C26.3 failures:** record/bound repeated unpaid approval failures as local policy. Do not evict on one transient state-dependent attempt. Releases, removed transactions and announcement metadata stay consistent.
- **C26.4 cancellation:** cancellation, failed/rebuilt payload, changing fee recipient, skipped slots and reorg do not mutate canonical state or leave provisional payer/dependency reservations. Existing Engine authority and FCU paths are reused.

### HGT-027: public submission/query

- **C27.1 submission:** real native eth_sendRawTransaction accepts valid plain/nonblob and network-wrapped blob Frames through the same admission path as gossip. Strict DATA/QUANTITY widths, canonical decoding, invalid signature and sidecar errors are explicit; public keyed decoding stays gated.
- **C27.2 transactions:** pending/mined/by-block queries return exact original identity, sender, chain/nonce, modes/flags/targets/limits/values/data, signatures/fees/blob hashes. Per-gas display fields cannot replace total max_fee. Invalid user JSON is not silently discarded.
- **C27.3 receipts:** return payer and ordered frame results, execution/state usage, skipped/failed effects, global log indices and committed roots, including cold disk readback. Pin/document the derived transaction status API policy; do not add it to consensus RLP.
- **C27.4 lifecycle:** inclusion, canonical fork selection, orphaned receipt queries, historical body retrieval, restart and fee derivation use the same original block context and persisted bytes.

### HGT-028: Frame simulation/estimation/trace

- **C28.1 selected API:** freeze JSON schema/version and errors for canonical Frame simulation before coding. Support complete transaction input through a documented native API. Existing standard methods must explicitly handle or reject type 6 instead of simulating it as ordinary CALL. Client extensions are labeled as such.
- **C28.2 semantics:** replay authenticated raw Frames with the correct sender/ENTRY_POINT origin, chain ID, slot, fees, payer, approvals, per-frame budgets and immutable signing hash. Public prefix rules are not applied to a full private simulation body. Context cancellation/timeouts and state overrides are bounded.
- **C28.3 estimates:** report both dimensions per frame plus intrinsic/floor/refund/blob/escrow accounting and explicit failed/skipped results. Measurements do not guarantee a minimum valid budget: GAS/TXPARAM/signing commitments make arbitrary limit mutation nonmonotonic. Any relaxed validation mode is explicitly local and cannot be admitted as a real transaction.
- **C28.4 trace:** actual Frame/call/opcode/log outputs and failure locations agree with untraced execution; no state, nonce, balance, logs or pool reservations persist. Trace/return-data size and execution time have limits. A scalar legacy estimator is not presented as a two-dimensional Frame estimator.

### HGT-029: native end-to-end qualification

- **C29.1 real peers:** submit via external HTTP RPC to one native node and observe a second connected peer receive/fetch/admit the same bytes without manual pool insertion or Engine IL injection. Cover plain and blob transport and invalid propagation controls.
- **C29.2 ordinary builds:** build from pending transactions, independently import the payload, select canonical FCU and compare state/receipts/BAL, payer/fees/nonce and raw query bytes across nodes. Existing inclusion-list tests remain separate consumers.
- **C29.3 recovery:** restart from disk and reorg across sender/payer code/state, root write/deployment, expiry and activation boundaries. Check affected revalidation, reservation/index cleanup and canonical receipt/body queries; distinguish legitimate reorg reinjection from forbidden repropagation of an evicted transaction.
- **C29.4 completion gate:** every criterion has positive/negative native consumers and current source/test provenance. Full build/vet/tests, affected race and hosted checks pass. Preserve A/B required consumers, independent oracles and explicit optional-test skip reasons. Do not claim official devnet/CL/RC qualification belonging to later groups.

## Implementation order and completion reporting

1. Lock this plan/authority review and profile/API boundaries.
2. Prefix grammar and trace/context exceptions, with dependency capture (HGT-020–022).
3. Prospective admission metadata, atomic payer exposure/replacement, fee ordering and resource/eviction lifecycle (HGT-021–024).
4. Exact candidate builder context, P2P handlers and public RPC/simulation contract (HGT-025–028).
5. Native two-peer, ordinary builder, recovery/reorg end-to-end qualification and complete criterion register (HGT-029).

Reuse existing canonical execution and interfaces. Completion reports separate implemented mechanisms, tested selected profiles and unresolved public interoperability gates. No missing authoritative schema/template/API is replaced by an invented consensus rule.
