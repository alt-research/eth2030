# Lighthouse and Hegotá implementation plan

Reviewed 2026-11-16. This plan covers the ETH2030 Lighthouse fork, the
native ETH2030 execution client, and their integration with Dora and the
devnet. Hegotá includes EIP-7805 FOCIL and EIP-8141 Frame transactions.
Heze is the consensus-layer fork and Bogotá is the execution-layer fork.
[Hegotá meta EIP](https://eips.ethereum.org/EIPS/eip-8081)

The [legacy branch integration plan](lighthouse-legacy-branch-merge-plan.md)
maps the old commits to their current production call sites, PRs,
and regression tests.

## Development baseline

| Component | Reference and evidence |
|---|---|
| ETH2030 Lighthouse | `jiayaoqijia/lighthouse:eth2030` at `3e3978d4e`, based on upstream unstable `1a704e58f`. The fork commit `6e5317104` carries the same inclusion-list view through local payload construction and Heze bid assembly. The regression passed on `3e3978d4e` and checks both the Engine request and the resulting bid when an equivocation arrives during the request. |
| Operator compatibility | Official Lighthouse stable `v8.3.3` and the pinned official unstable each finalized epoch 2 in separate two-pair Fulu devnets. Both CLs and both ELs agreed on the finalized execution hash in each run. This evidence predates the fork commit above. |
| Glamsterdam | Local integration build `f85fa05b6` crossed from Fulu into Gloas. At slot 34 both pairs returned the same signed envelope, execution block and state root. Direct Gloas genesis remained blocked at the EL genesis hash after generator `6.2.8` fixed the CL's SSZ decode error. |
| Native ETH2030 EL | `origin/master` at `ad35748` is the source audit baseline. The earlier live Fulu/Gloas evidence used image revision `89cfff9`. The current source has an earlier Frame implementation and a partial Bogotá Engine surface. |
| Historical CL | `feat/fix-native-aa` at `6552c7124` records old interoperability and regression scenarios. Its AA-named commit `3b5fc06f6` changes FOCIL and adds no Frame decoder. EIP-7701 has been withdrawn. |

[Fork development notes](https://github.com/jiayaoqijia/lighthouse/blob/3e3978d4eacda648b32ddc45ac32222b00afc6ba/ETH2030.md) · [Previous live evidence](cl-iteration-roadmap.md#lighthouse-baseline-verification) · [EIP-7701](https://eips.ethereum.org/EIPS/eip-7701)

Use `eth2030` as the fork's development target and pinned upstream stable
for operator compatibility checks. The GitHub default remains `stable`.
Repository admin access is needed to change that setting. Development
PRs can already target `eth2030`.

For each upstream update, compare the old and new upstream bases with our
fork delta. Retain locally owned behavior, reproduce the regressions the
old code addressed, and retire patches once equivalent upstream behavior
passes the relevant cases. Keep generated protocol vectors and fork
settings tied to the version they exercise.

## Delivery order

| Milestone | Changes | Acceptance |
|---|---|---|
| P0.1 Contract and baseline | Pin the fork and fixture tuple. Run Fulu and Fulu-to-Gloas using the actual `eth2030` branch. Start the Amsterdam genesis fix independently. | New images reproduce Fulu finality and Gloas envelope import on two pairs. Golden fixtures distinguish Bogotá request and response versions and both Frame draft encodings. |
| P0.2 Frame interoperability | In parallel, update the EL's Frame consensus execution and the CL's local payload verification. | Real canonical receipts and payer balances agree for plain and sponsored Frame transactions. A blob-bearing Frame payload survives local CL verification and historical backfill. |
| P0.3 FOCIL and Bogotá | Implement the Engine contracts with ordinary transactions as fixtures. Complete CL duty production, gossip/retrieval, bid checks and satisfaction handling. | A signed, nonempty list compels inclusion at a proposer that did not receive the transaction in its mempool. The execution-valid omission case produces the specified CL decision. |
| P0.4 Combined and downstream acceptance | Combine the Frame and FOCIL paths, test restart and late join, and migrate Dora and the transaction generator as needed. | A sponsored non-blob Frame enters the finalized chain because of FOCIL. Dora reads the correct committee position and receipts after restart. Run blob-bearing Frame independently. |
| P1 and research | Run separate timing, execution proof, full/snap sync and Lean consensus scenarios. | Apply the acceptance criteria in the broader roadmap. |

P0.2 and P0.3 can land and be tested separately after the relevant
fixtures are pinned. P0.4 uses both implementations. A sponsored Frame
with a blob belongs in the Frame and data-availability scenario. The
inclusion-list producer should avoid blob-carrying transactions under
the selected Bogotá draft.
[Bogotá selection rules](https://github.com/ethereum/execution-apis/blob/981aa66d79364be8322a9c358aaf0a571cca0873/src/engine/bogota.md#engine_getinclusionlistv1) · [Compatibility and research roadmap](cl-iteration-roadmap.md#p1--extend-compatibility-recovery-and-timing)

### P0.1 · Pin the contract and reproduce the baseline

Record full source revisions, image digests, execution and consensus fork
settings, chain ID, genesis generator and output hashes, validator
distribution, fixture revisions, builder and instrumentation versions in
each run manifest. Map a CL epoch to the intended EL timestamp and query
both nodes for the effective schedule. Keep Engine JWTs and validator
secrets out of retained traces.

Start with the existing Track 4 launcher and verifier in a fresh
enclave. Record the booted CL version and EL image identity before using
the result as the new baseline. The Heze and mixed Frame scenarios need
their own fixtures and assertions.
[Track 4 devnet runbook](../pkg/devnet/kurtosis/README.md#track-4-private-devnet)

Use these references to start the fixture comparison.

- EIP-8141 at `dc5d5fa6`, Heze consensus specs at `313cbfe5`, and the
  Bogotá Engine draft at `981aa66d`. Review later upstream changes as an
  explicit revision of the test contract. Record any considered changes
  present in the code or fixtures, including EIP-8015 beacon block
  fields, EIP-8250 keyed nonces and EIP-8272 recent roots.
- Bogotá devnet fixtures `tests-bogota-devnet@v2.0.0` plus vectors for
  subsequent spec differences. The November nonce-boundary revision
  postdates that fixture release. Record which expected outputs are
  independently generated for the selected version.
- Replay the earlier Frame encoding as a historical fixture. Compare
  exact RLP bytes, signing root, transaction hash, fee budgets,
  authorization and receipt values with the selected draft. Include an
  ordinary transaction on either side of the fork boundary.

[Pinned Frame draft](https://github.com/ethereum/EIPs/blob/dc5d5fa688b7fe87b95a907fcc128b4fde2f4d2f/EIPS/eip-8141.md) · [Pinned Heze validator spec](https://github.com/ethereum/consensus-specs/blob/313cbfe561517a81eb11607fea7c6985e562eaa9/specs/heze/validator.md) · [Bogotá fixture release](https://github.com/ethereum/execution-specs/releases/tag/tests-bogota-devnet%40v2.0.0)

## P0.2 · Frame transaction implementation

### Execution client

Audit the normal path from typed transaction decode and fork activation
through txpool admission, block building, payload import, state
transition and RPC receipt readback. The earlier EL represents an
individual frame as four fields, supports a keyed nonce, and carries
separate per-blob fee caps. Move the production path to the selected
seven-field transaction and nested frame layout, signature scheme and
canonical signing hash, nonce and frame-count limits, execution/state
gas limits, and shared fee budget. Preserve historical decoding when
required by an existing chain's configuration.

Use one consensus execution path for block building, direct block import
and Engine validation. Verify sender and payer approvals, payment escrow
and refund, frame modes and value transfer, default and delegated
accounts, atomic batch rollback, state-gas accounting, and receipt
encoding with independently generated expected state and receipts. Wire
receipt and transaction readback through the canonical database and RPC.
Check restart, reorg and the sender/payer balance affected by each case.
Apply the draft's mempool simulation, replacement, reservation and
eviction rules separately from block validity. Track nonce-boundary,
frame-count, gas-limit, signature and budget failures with state
unchanged at the rejected boundary.
[Frame types](../pkg/core/types/tx_frame.go) · [Execution](../pkg/core/eips/frame_execution.go) · [Txpool](../pkg/txpool/txpool.go) · [Receipt generation](../pkg/core/execution/receipt_generation.go)

### Consensus client

Keep EL execution and transaction authorization behind Engine. At
`1a704e58f`, Lighthouse's local verification calls Alloy's transaction
decoder and extracts hashes only from EIP-4844 envelopes. Inputs with
type `0x06` returned `unexpected tx type` with and without a blob. Fix
the active-fork decoding and blob extraction path for Frame transactions.
Keep original transaction bytes for execution block-hash verification
and use the pinned RLP layout for extraction. Prefer shared upstream
decoder support when available. Report malformed encoding and incorrect
commitments through the appropriate verification result.

Run the same finalized payload through EL-assisted envelope import,
CL-local verification and historical backfill. Assert empty, singleton
and multiple Frame blob hash lists. Compare mixed EIP-4844/Frame
transactions in block order and their hashes in transaction order.
Reordering, omission, extra commitments and malformed Frame inputs must
produce the expected result in the applicable path. Check that pre-fork
transactions and existing EIP-4844 handling continue to conform. Measure
whether optimistic finalized sync uses its intended path for valid Frame
payloads instead of relying on an EL-assisted fallback.
[Versioned hashes](https://github.com/sigp/lighthouse/blob/1a704e58f5c4ab82252fab46e35789d868a53211/beacon_node/execution_layer/src/versioned_hashes.rs) · [Historical envelope verification](https://github.com/sigp/lighthouse/blob/1a704e58f5c4ab82252fab46e35789d868a53211/beacon_node/beacon_chain/src/historical_blocks.rs)

## P0.3 · FOCIL and the Bogotá Engine contract

| Interface | Implementation and first regression |
|---|---|
| `engine_getInclusionListV1` | ETH2030 returns the draft's JSON array of transaction hex strings. Check empty and nonempty mempools, the 8 KiB aggregate byte budget, zero-length rejection and blob filtering. Lighthouse retrieves the actual list used for the assigned duty. |
| `engine_forkchoiceUpdatedV5` | Decode the canonical camelCase attributes with hex quantities, `targetGasLimit` and `inclusionListTransactions`. Reuse the canonical list-to-builder path and its payload identity. Thread the caller's gas target into that path. If old profiles still use the snake_case adapter, exercise its actual callers and preserve supported values. Keep the optional custody parameter and return `PayloadStatusV2` according to the draft. |
| `engine_getPayloadV6` | Verify the inherited Amsterdam response against the payload ID from V5 forkchoice. Trace the built transaction order, BAL and commitments into the proposed block. |
| `engine_newPayloadV6` | Add dispatch, capability and fork validation on the EL. Route real Heze envelopes through V6 on the CL, including the appropriate list view. Encode and consume `inclusionListSatisfied` independently of execution validity. Cover Amsterdam V5 and Bogotá V6 on both sides of activation. |

The proposer uses the previous slot's applicable lists under the parent
dependent root, including late lists as specified for proposal. Retain
the fork's self-build snapshot behavior. Validate incoming bid and block
inclusivity using the right view at each call site. Poll VC duties,
resolve the duty's slot and dependent root, retrieve the EL candidate,
sign under the Heze domain and publish through a BN before the prescribed
deadline. Handle head changes, skipped slots, redundant BNs and restart.
Retransmit the same signed message when retrying a publication. Cover
signer behavior and equivocation across retries and failover.

Use the upstream store and verified gossip path. Implement
`InclusionListsByIndices` if it remains absent from the selected
integration revision. Bind serving and recovery to slot, dependent root
and ordered committee indices. Test a missing gossip message recovered
by request/response, another branch's root, equivocation, retention
limits and peer responses containing invalid entries.

Check satisfaction against the envelope's applicable timely lists and
the selected execution state. Record separately which required
transactions were omitted and whether the spec permits those omissions.
An execution-valid payload may be inclusion-unsatisfied. Non-valid
execution outcomes carry null satisfaction. Preserve the list for
`ACCEPTED` as specified and cover retry, competing branches and FCU
revalidation. Cache by the inputs that determine satisfaction so the
same execution hash can be checked against different local list views.
Preserve verified satisfaction after the lists are pruned. Exercise the
optimistic-to-verified transition with the pinned satisfaction rules
and verify the result across restart. Run the same otherwise extendable
payload against satisfying and unsatisfying views. Record both
`should_extend_payload` decisions and the execution parent that the next
proposer selects.
[Bogotá Engine methods](https://github.com/ethereum/execution-apis/blob/981aa66d79364be8322a9c358aaf0a571cca0873/src/engine/bogota.md) · [Heze networking](https://github.com/ethereum/consensus-specs/blob/313cbfe561517a81eb11607fea7c6985e562eaa9/specs/heze/p2p-interface.md) · [Heze fork choice](https://github.com/ethereum/consensus-specs/blob/313cbfe561517a81eb11607fea7c6985e562eaa9/specs/heze/fork-choice.md)

## P0.4 · Devnet and consumer acceptance

Start with two independent CL/EL pairs, separately driven validators and
isolated mempools for compulsory-inclusion cases. Prove the proposer did
not see the transaction in its EL before receiving the list. Use an
independent reference EL or CL where its draft revision matches. Retain
an explicit result for each scenario.

| Scenario | Acceptance evidence |
|---|---|
| Fulu control | Canonical Track 4 transaction, matching finalized execution hashes and stable progression on both pairs. |
| Gloas and Amsterdam | Compare the generated EL genesis header with the CL fixture, including the empty BAL hash and genesis slot. Test direct genesis and a live fork transition. Verify self-build, external nonzero bid and builder payment, late/missing envelopes, PTC decision, BAL agreement and incorrect BAL rejection, and custody/availability with blob columns. |
| Heze and Bogotá activation | Cross a scheduled fork boundary from Gloas/Amsterdam. Capture an Amsterdam `newPayloadV5` before activation and a Bogotá `newPayloadV6` after it. Verify attributes and responses in both runs, wrong-fork method errors where specified, and matching canonical state past the boundary. |
| Frame without FOCIL | A plain and a sponsored Frame transaction reach canonical receipts. Both ELs agree on authorization, frame outcome, receipt and payer debit. Replay the receipt from stored canonical history after restart. |
| Blob Frame | A blob-bearing Frame and an EIP-4844 transaction enter an envelope. Compare the ordered hashes extracted from the transactions with the versioned hashes derived from the bid's KZG commitments. Verify data availability and historical import on a fresh node. Withheld data and a hash mismatch are separate controls. |
| Ordinary transaction through FOCIL | Capture the committee EL's list result, signed publication, peer delivery, the proposer Engine request, bid bits, imported block, receipt and finalized hash. Compare against the same setup without delivery. |
| Execution-valid omission | Hold the required transaction and parent context fixed. Verify that V6 returns `VALID` and satisfaction false, that the CL retains the valid envelope, and that `should_extend_payload` returns false where the satisfying control extends the payload. Verify that empty views add no constraint, late lists are excluded from the timely view, wrong-branch and equivocating lists do not contribute, and omissions permitted by the EL rules keep satisfaction true. |
| Sponsored Frame through FOCIL | Repeat the signed-list chain with a sponsored non-blob Frame. Match its bytes across the wire, bid bits, final receipt, sponsor balance delta and sender state on both ELs. |
| Restart and late join | Interrupt payload, envelope or data delivery, restart roles with their intended storage, and import the same history from a fresh node. Assert head, safe/finalized execution hashes, receipts and roots. Exercise optimistic finalized sync and historical backfill as separate observations. |
| Dora and transaction generation | Dora consumes the pinned BN's inclusion-list events and resolves ordered committee positions from slot and dependent root. After restart, compare its reconstructed list and displayed Frame transaction and receipt data with the BN and canonical EL RPC. Assert late, missing and equivocated list handling. Update the Frame generator when its wire encoding differs, and use its actual broadcast transactions in the acceptance run. |

The run should retain sanitized Engine request and response samples,
Beacon API/gossip observations, the validator duty and its decision,
transaction and payload identifiers, EL balance/receipt reads, and both
clients' finalized hashes. Record the assertion that failed when a
scenario cannot pass. A successful compile, fork activation, or
single-node receipt establishes only the evidence it directly observed.

## Review and release

Land changes by repository ownership. The CL branch owns validator
duties, forkchoice, gossip, envelope verification and Beacon APIs. The
EL branch owns transaction semantics, storage, payload construction and
Engine wire behavior. Dora and the generator consume the same pinned
contracts. Link the dependent PRs to the exact fixture revision and
test profile they expect. Scope draft behavior to the configured fork
and keep Fulu and Gloas baselines in regression testing.

Run the affected packages, locked Lighthouse compilation and lint,
spec/differential vectors, wire tests and the live acceptance matrix
before treating a client pair as ready for the integration profile.
For a devnet release, publish the matching CL and EL image digests with
the network profile, fixture references, verification record, and any
unsupported scenarios. Use those digests for replay. If an integration update
fails a gate, retain the previous verified pair for new devnet runs and
record the failed tuple. For a persistent node, check database
compatibility before any version rollback and use the corresponding
backup or fresh resync when required.

Follow the separate roadmap for Quick Slots, optional execution proofs,
full/snap recovery and decoupled/PQ consensus. Retain their own
specification pins, test profiles and completion evidence.
[CL project roadmap](cl-iteration-roadmap.md)
