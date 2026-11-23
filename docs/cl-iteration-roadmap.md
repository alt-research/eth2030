# Consensus client iteration after the merged Frame milestone

Updated November23,2026. Lighthouse`eth2030` is`fa16f70abee7d31c9817ab8d782af73bce241484` after PR4-15;1-3 were already ancestors. EL tracking masterd743925e includes PR141 with unchanged frozen Go package tree. Lighthouse #13-15 are merged into eth2030. Exact CI head `215525de` passed all 27 test-suite jobs and local-testnet (9 passed, one conditional checkpoint skip); merged `fa16f70a` has the same tree. G-01 is closed for this private profile. The primary simulator uses explicit 6s timing; the original 3s failure remains retained. See [repair evidence](reviews/remaining-20261123/CI_REPAIRS.md) and [current campaign](reviews/remaining-20261123/CURRENT_STATE.json).

The completed bounded 6h45m05s run and 14 unchanged-reference BLS windows authenticate actual stopped CL/native slot 97868. Original 04 precise waiting holder/physical I/O and actual 48h capacity remain unconfirmed. Gloas signed builder/payment/late/withheld and independent Glamsterdam full/snap controls are delivered at their recorded sources; they are no longer generic missing implementation tasks. [Current criteria and source TODOs](plans/glamsterdam/NEXT_STEPS_20261123.md).

The retained November 18 Frame/FOCIL milestone below has its own source and synchronized-EL boundary. Independent Frame-client agreement, Dora and service cutover remain separate.

## Delivered baseline

- Lighthouse `eth2030` includes runtime `a4ceb86e` and tracking commit `29d66c238`.
  Its upstream integration, Heze duty/signing/publication and Req/Resp paths are
  retained with regression coverage.
- Native Bogotá keeps execution validity separate from inclusion satisfaction,
  including deferred FCU and persisted view/restart cases.
- The final paired control finalized coded sponsorship at slot 66 and a blob
  Frame at slot 165. A fresh validator-free CL joined at slot 256 and read both
  original finalized envelopes. Its EL was already synchronized.
- Real envelope processing now handles the trusted checkpoint without its
  unavailable pre-anchor parent. Live V6 requests retain nonempty observed lists;
  bad signatures/bids and missing custody data remain rejected or pending.
- Receiver restart preserved both workloads' historical receipts, storage and
  sponsor settlement. Runtime version metadata matches the built source.

[Final control](verification/frame-8141-20261118-paymaster-history-final.json) ·
[Execution record](frame-8141-execution.md) · [Active work plan](next-stage-work-plan.md)

## Next iterations

1. **Independent contract:** choose matching Frame/Bogotá fixtures and an
   independent EL, or version the keyed-nonce experiment separately. Compare
   actual execution/IL verdicts, canonical roots and recovery.
2. **Service consumers:** adopt upstream Dora/workload behavior for covered
   contracts. Wire the qualified Heze SSZ candidate through normal epoch/receipt
   consumers and qualify pruning, event recovery and restart before retiring forks.
3. **CL reliability:** exercise skipped-slot checkpoints, VC restart, remote
   signing, redundant BN publication, alternate-peer and partial-response recovery.
   Preserve duty/root binding and avoid signing conflicting lists for a duty.
4. **Gloas integration gates:** maintain the complete integrated CI and then
   extend only adopted builder/peer profiles beyond the delivered frozen controls.
5. **Capacity and recovery:** extend declared growing-state/history operation and
   restart beyond the independently qualified finite Glamsterdam sync profiles.
6. **Longer horizon:** Heze light-client/proof vectors, decoupled consensus and
   PQ key/committee lifecycle remain separately pinned research acceptance.

Keep the strict live-view and fork-choice verdict regressions when integrating
upstream. The experimental upstream helper's broad empty-list fallback is not
used to resolve unknown live parents. Use released implementations for covered
features and keep experimental compatibility within upstream component boundaries.

Every live profile uses one bounded enclave, explicit client/service caps and
retained source/image identities. Stop and remove owned runtime resources after
every terminal result.

## Historical design and migration review

The dated review below predates the merged milestone. Its source comparisons
and legacy-port rationale remain useful; the delivered baseline and next
iterations above supersede its implementation-to-do statements.

## Decision and first milestone

Keep Lighthouse as the release CL and use the native Go 3SF/PQ observer
for research. The next milestone is a **nonempty Heze inclusion list that
changes what the proposer builds and what the attesters vote for**. In
parallel, establish a reproducible Gloas/Amsterdam baseline.

| Order | Deliverable | Acceptance evidence |
|---|---|---|
| 1 | Pin the Lighthouse reference and Engine contract fixtures. | Record client revisions and image digests, genesis and fork settings, and actual Engine requests and responses. |
| 2 | Complete the Heze validator path in the CL and the Bogotá contract in the EL. | The VC originates signed lists, the proposer sets the bid bits, and the BN uses V6 and consumes inclusion satisfaction independently of execution validity. |
| 3 | Run the nonempty-list devnet on independently driven EL/CL pairs. | Follow an ordinary transaction and a sponsored Frame transaction from committee selection to canonical receipt. Read the sponsor's balance and observe the CL's decision when a valid payload omits a required transaction. |
| Parallel | Verify Gloas/Amsterdam behavior and recovery. | Read back external builder payment, PTC behavior, BAL agreement, blob data availability, and late-join or restart state. |

## Evidence available today

| Verified scope | Next evidence gap |
|---|---|
| PR #55 merged with six GitHub CI checks passing. | Run the pinned EL/CL pair with the new Heze Engine contract. |
| The live Track 4 profile exercised Fulu, private PQ authentication, binary overlay conversion, and historic state readback after EL restart. | Exercise a nonempty FOCIL list at Heze. |
| Earlier full-feature testing observed Gloas bid activity, nonempty Amsterdam BALs, and Heze activation. | Observe external payment and PTC outcomes, blob data columns, and the effect of inclusion satisfaction on CL voting. |
| Track 3 exercised CL-driven synchronization and the native observer in integration tests. | Repeat the late-join test with an independent client and retain a research-only comparison of native observer outcomes. |

[Track 4 devnet record](../STATUS.md#track-4-integration-and-devnet-verification-2026-11-16) · [Earlier full-feature run](../STATUS.md#full-feature-devnet-interop-after-bal-enforcement-2026-11-12) · [Track 3 verification](track3-node.md#verification-and-boundaries)

### Lighthouse baseline verification

The two-pair Fulu profile used the `ethereum-package` revision `c4a8e15d`,
genesis generator `5.3.5`, and ETH2030 EL image `89cfff9` with image ID
`sha256:406801bc8e4b6e0d711c4f369e75f911e08bc73272188eb309fb7df82889a3ad`.
Each image below passed `verify-track4.sh` with two independent EL/CL
pairs. Both ELs agreed on the transaction's canonical execution block
and state root, and both Beacon APIs returned its payload at slot 5.

| CL image | Image identity | Verified execution block hash |
|---|---|---|
| Official stable `v8.3.3-57c2abc` | `sha256:53f9063d98c3d3a2d290207873f019ea2298cfc02f8cd58d87c33b04aab856d1` | `0x56431bf7af2b3e0663b3b650b3601f363236ac9d11e0ff8c64563d35572c3415` |
| Official unstable `v8.4.0-rc.0-1a704e5` | `sha256:219533433228d9ec2ad51cf5a6c07d718f609b4e633bf656f2fa06501aa76305` | `0xf634830bd299595e05ec6e924203c6754fb7fe524b59f8b25db5b6bfc524e102` |
| Local release build from `f85fa05b6` | `sha256:c35c0f1e5c92f5a075c099c9e069fd11de41ad94262129488d6918a7e428443f` | `0x27f1592560c04dc9d896f7feaa749b7643de42fbb09b7d85e20466294ab4368b` |

The integration source passed `cargo fmt --all --check`,
`cargo check --locked -p lighthouse`, 40 `execution_layer` tests, 14
blob-fetch tests and 30 inclusion list verification and store tests. The
local release image records the source revisions in its OCI labels. The
official unstable image includes the revision in the Beacon node version.
[Track 4 verifier](../pkg/devnet/kurtosis/scripts/verify-track4.sh)

The official stable and unstable runs both finalized epoch 2. For each
run, both Beacon APIs agreed on the finalized execution hash and both ELs
returned the same value for `finalized`. The stable hash was
`0x123afc88769f924dcb81616cdcaa8ffdf89b692d7ac172701ffbb7a882bab230`.
The unstable hash was
`0xdd18a70831159d7c41bf9e05f854f976c902d64e0bc1214e9ada8f1e663b45a2`.

The local build started at Fulu and crossed into Gloas at epoch 1. At
slot 34, both nodes returned the same block, bid and signed envelope
through the current Beacon API. Both ELs returned execution hash
`0x8594575386aad5684a6c0dc3c86fdc645310bbc64a5ba96b8dba5125d08c92bd`
and state root
`0xac5aaeef252d49341738616388b72062308a06c7952deed3b673e65dc5d9dfb3`.
This covered self-building. External builder payments, the PTC voting
outcome and blob availability still have separate acceptance scenarios.

A separate Gloas-at-genesis run with generator `5.3.5` failed consensus
SSZ decoding (`OffsetsAreDecreasing(0)`). Repeating with generator `6.2.8`
booted both CLs. Their Gloas genesis state pointed to execution hash
`0x2896b31b1b2af06ed072759070d8e4f748af6238c6a44a7ed150ee3778c51920`.
Both ETH2030 ELs had genesis hash
`0x33f6a2f9c5687ef1234b1ccf998d22956361071ba9be78cd418de4788714c5ba`
and returned `SYNCING` on the first forkchoice requests. RLP-reencoding
the EL RPC header with the empty BAL hash and `slotNumber=0` reproduced
the CL's expected hash exactly. Fix the native Amsterdam genesis header
and keep the Fulu-to-Gloas transition as the working integration profile.
[Genesis header construction](../pkg/core/config/genesis.go) · [Genesis generator v6.2.8](https://github.com/ethpandaops/ethereum-genesis-generator/releases/tag/v6.2.8) · [EIP-7928](https://eips.ethereum.org/EIPS/eip-7928) · [EIP-7843](https://eips.ethereum.org/EIPS/eip-7843)

## Client direction

Use upstream Lighthouse as the production CL codebase and retain an ETH2030
integration branch with a small patch queue. Lighthouse's beacon node and
validator client own validator duties, consensus state, gossip validation,
forkchoice and finalization. ETH2030 owns payload construction, execution,
receipt and state commitments, and state sync. The authenticated Engine API
connects them. Keep validator duties and slashing protection with the
Lighthouse validator client or its established external signer.

Follow stable releases for operator testing and pin upstream unstable
revisions for fork devnets. The current stable release is `v8.3.3`
(`57c2abc13`). The integration update `f85fa05b6` on
`integrate/unstable-20261116` merges the 20 upstream commits since
`25b9b7541`, including stable, at upstream unstable revision `1a704e58f`.
That baseline integration tree differs from upstream by one documentation
symlink. The published `eth2030` branch at `3e3978d4e` also carries the
Heze self-build change that keeps the bid bits and Engine transactions
in the same list view.
The earlier `feat/fix-native-aa` branch has 69 commits absent from this
upstream tip. The patch audit below identifies the behavior to carry
forward from that prototype.
[Lighthouse upstream revision](https://github.com/sigp/lighthouse/commit/1a704e58f5c4ab82252fab46e35789d868a53211) · [Lighthouse v8.3.3](https://github.com/sigp/lighthouse/releases/tag/v8.3.3)

### Historical Lighthouse patch audit

The March branch changed 120 files from its upstream base `17d183eb5`.
Use `1a704e58f` for new work. Reimplement the missing behavior against the
Heze and Bogotá drafts and keep the old branch available for regression
scenarios.

| Historical changes | Current upstream and required follow-up |
|---|---|
| Gloas PTC, execution forkchoice, envelope availability, storage and fork-boundary fixes | Use the upstream implementations. Keep the March skipped-slot, pending data, fork-boundary and restart cases as regression inputs. The Gloas Beacon API reads an envelope with `GET /eth/v1/beacon/execution_payload_envelopes/{block_id}`. Dora `6f702f2` reads execution data by the bid hash from the EL and has no caller for the March `/blocks/{block_id}/envelope` route. |
| FOCIL types, gossip, committee selection and list store | Use the upstream code keyed by slot, dependent root and validator. March wire data used `inclusion_list_committee_root`. The current message uses `dependent_root` and hex transaction strings. At `1a704e58f` the network supports gossip and has no `InclusionListsByIndices` request/response. Implement the draft request with `slot`, `dependent_root` and the committee bitvector. |
| VC `InclusionListService`, scheduling and `6552c7124` default broadcast | Upstream polls Heze duties and publishes accepted signed lists from its Beacon API to gossip. At the pinned revision there is no production caller taking a duty through EL list acquisition, validator signing and publication. Restore that behavior using the upstream API and signing domain. Test delivery between independent beacon nodes and retry with a redundant BN. |
| Heze proposer and verifier, including the old `INCLUSION_LIST_UNSATISFIED` result | Upstream prepares payload attributes from its list store. Fork commit `6e5317104` attaches bid bits from the same list view to local Heze builds. Envelope import still constructs `NewPayloadRequest::Gloas`. The Engine V6 client exists, but normal import never constructs its request and the status conversion drops `inclusionListSatisfied`. Complete incoming bid checks and the validation and forkchoice paths using Bogotá's separate execution status and satisfaction Boolean. |
| Gloas and Heze light-client shortcuts | Use upstream Gloas headers with the execution block hash proof. The selected revision returns `HezeNotImplemented` for Heze. Schedule Heze light-client work when its schema and proof vectors are available. |
| Explorer inclusion-list reads | Dora calls the removed `/blocks/{block_id}/inclusion_lists` endpoint and calculates committee positions from response order. Move its indexer to Lighthouse's `inclusion_list` event and resolve positions by committee, slot and dependent root. Cover missing, duplicated and equivocated lists and persist the events needed to reconstruct historical views. |

[Beacon API envelope contract](https://github.com/ethereum/beacon-APIs/blob/master/apis/beacon/execution_payload/envelope_get.yaml) · [Heze validator specification](https://github.com/ethereum/consensus-specs/blob/313cbfe561517a81eb11607fea7c6985e562eaa9/specs/heze/validator.md) · [Heze networking specification](https://github.com/ethereum/consensus-specs/blob/313cbfe561517a81eb11607fea7c6985e562eaa9/specs/heze/p2p-interface.md) · [Lighthouse envelope import](https://github.com/sigp/lighthouse/blob/1a704e58f5c4ab82252fab46e35789d868a53211/beacon_node/beacon_chain/src/payload_envelope_verification/payload_notifier.rs)

Keep an opt-in research environment for native 3SF, timing and PQ votes.
The existing Go observer uses a configured ML-DSA registry, votes over
`ncs/2`, and follows the external CL checkpoint. It is useful for
differential measurements and restart testing. A standalone research CL
would require its own validator duties, block production, state transition,
forkchoice, sync and restart state, and a single Engine forkchoice writer.
Use pinned leanSpec fixtures and an independent Lean client for that work.
[Native observer contract](track3-node.md#explicit-native-consensus-observer) · [leanSpec](https://github.com/leanEthereum/leanSpec)

## Upstream roadmap commitments

| Horizon | ETH2030 CL work |
|---|---|
| Glamsterdam | Establish reproducible Gloas/Amsterdam interoperability and close external builder, BAL and blob-availability scenarios. Track the currently published repricing and fork config. |
| Hegotá | Deliver Heze FOCIL with Bogotá Frame transactions. Keep Quick Slots as a separate timing experiment while EIP-8198 is considered for inclusion. |
| I* / J* | Track decoupled consensus, state transition design and the public PQ registry / minimum viable PQ sequence. Prototype against versioned specs and retain the existing private PQ and binary overlay as implementation evidence. |
| Later forks | Track proof enforcement and Lean PQ consensus as separate upstream decisions. The relative order of mandatory execution proofs and PQ consensus remains a dependency to monitor. |

The EF Hoodi notice postponed the old Glamsterdam activation. The
Hegotá meta EIP schedules FOCIL and Frame transactions. The Strawmap and
EF priorities describe provisional later milestones. Maintain upstream
schedule data independently of private devnet fork epochs and the
repository's feature tracker.
[Hoodi notice](https://blog.ethereum.org/2026/10/22/glamsterdam-hoodi-delay) · [Hegotá meta EIP](https://eips.ethereum.org/EIPS/eip-8081) · [Strawmap](https://strawmap.org/) · [EF protocol priorities](https://blog.ethereum.org/2026/09/07/protocol-priorities)

## P0 · Freeze the EL/CL contract and exercise Heze

Run the baseline Gloas/Amsterdam profile while implementing the Heze
adapter. Record a reproducibility manifest containing the EL and CL
commits and image digests, consensus-specs and execution-apis revisions,
genesis hash, fork schedule, builder image and fixture versions. Keep
private integration settings in the devnet profile. Record RPC method,
version and payload identifiers with no credentials. A green check should
require evidence from independent node roles and the finalized consumer.

The first implementation change spans the Heze CL and ETH2030's native
Engine HTTP backend. At `ad35748` the EL dispatcher and capability
exchange omit `engine_newPayloadV6`, and `engine_forkchoiceUpdatedV5`
delegates to V4. Lighthouse's wire test at `1a704e58f` sends camelCase
`inclusionListTransactions` as an array of hex strings. The corresponding
EL JSON decoder reads that shape. The separate snake-case compatibility
path still discards the list, and the EL V4 attribute type has no
`targetGasLimit` field. The EL's `engine_getInclusionListV1` emits an
object with a `transactions` member. Bogotá and Lighthouse require an
array. Use the same pinned contract for the VC, BN and EL test fixtures.
[EL dispatcher](../pkg/engine/handler.go) · [EL compatibility parser](../pkg/engine/lighthouse_compat.go) · [EL list response](../pkg/engine/inclusion_list.go) · [Lighthouse Engine client](https://github.com/sigp/lighthouse/blob/1a704e58f5c4ab82252fab46e35789d868a53211/beacon_node/execution_layer/src/engine_api/http.rs) · [Bogotá Engine API](https://github.com/ethereum/execution-apis/blob/981aa66d79364be8322a9c358aaf0a571cca0873/src/engine/bogota.md)

Implement the contract in this order.

1. Pin the CL and EL API draft tuple and agree on explicit forks, method
   capabilities, production attribute encoding and exact transaction-list
   encodings. Capture request and response fixtures from both clients.
2. Make committee list acquisition return the expected list shape and
   enforce the draft's selection and size rules. Preserve the list from
   proposer-side `forkchoiceUpdatedV5` to payload construction and bind it
   to the candidate context. Include slot, parent, target gas and list
   content when identifying builds. Keep committee authorization, gossip,
   and the subjective set of timely lists in the CL. Connect VC duties to
   signing and publication, retain the self-build bid bits from the lists
   used, and apply the draft's inclusivity checks during block and bid
   handling.
3. Implement `newPayloadV6` and `PayloadStatusV2` in the native EL.
   Compute inclusion satisfaction independently of execution validity.
   Retain the list for accepted payloads and use it on the applicable
   revalidation path. Include the list context when caching satisfaction
   because different CL callers can evaluate the same execution block
   against different lists. Return execution validity and list
   satisfaction as separate fields. Send the Heze request from normal
   envelope verification and consume the Boolean in the CL's
   attestation and forkchoice decisions. Apply the draft's treatment of
   undecodable transactions and transactions that become invalid on
   the selected parent.
4. Verify the complete flow in two independently driven CL/EL pairs.
   Produce a signed, nonempty list carrying a transaction seen by the
   committee EL and absent from the proposer EL's pool. Observe the same
   transaction in gossip, Engine payload attributes, the imported
   execution block and the canonical receipt. Repeat with a sponsored
   EIP-8141 transaction and verify the payer's balance delta on both ELs.
   In a controlled comparison, submit an execution-valid payload omitting
   a required transaction and assert the CL's voting/head behavior and
   V6 satisfaction result. Include empty, late, equivocated and
   conflicting-parent lists, with decisions checked against the pinned
   version. Complete the recovery case across payload import, forkchoice
   and restart.

Use separate acceptance scenarios for Gloas/Amsterdam. Exercise a
registered external builder with a nonzero bid and read payment and
withdrawal state after a timely envelope and PTC participation. Test
self-build and missing or late envelope behavior as distinct cases.
Include nonempty BAL agreement and rejection of a header-consistent
payload carrying an incorrect BAL, as well as blob payloads with
independently verified data columns and an unavailable-data case. Use a
second mainstream client where the pinned fork is supported. Run an
execution-node late join and restart while the CL continues to attest.
The earlier full-feature devnet observed Gloas bid activity, Amsterdam
BAL enforcement and Heze activation. Those observations establish a
baseline for these targeted scenarios.
[Earlier devnet evidence](../STATUS.md#full-feature-devnet-interop-after-bal-enforcement-2026-11-12) · [EIP-7732](https://eips.ethereum.org/EIPS/eip-7732) · [EIP-7928](https://eips.ethereum.org/EIPS/eip-7928) · [EIP-7594](https://eips.ethereum.org/EIPS/eip-7594) · [EIP-7805](https://eips.ethereum.org/EIPS/eip-7805) · [EIP-8141](https://eips.ethereum.org/EIPS/eip-8141)

## P1 · Extend compatibility, recovery and timing

- Exercise the merged EL with a real Lighthouse CL and a clean late-join
  EL database in separate full and snap scenarios. Use an independent
  source where supported. Measure root and proof agreement, finality
  advances during sync, interruption, state adoption and post-restart
  progress. Keep EL snap moving-pivot and bounded-memory work on its
  existing backlog. The bare presence of `snap/1` only establishes
  capability advertisement. [Track 3 acceptance design](track3-remaining-acceptance.md)
- Test EIP-8198 in a separate timing profile that crosses a scheduled
  slot-duration boundary. Record missed proposals, attestations and
  participation, plus proposal, Engine, envelope and data-column
  latency distributions. Derive acceptance from the specified
  deadlines and compare against a same-workload baseline. Check
  restart, clock skew and the validator client's duty scheduling at
  the boundary. A constant six-second private genesis covers only that
  constant configuration. [EIP-8198](https://eips.ethereum.org/EIPS/eip-8198)
- Pilot EIP-8025 with Lighthouse's upstream proof interface and an
  independent proof provider. Bind the exact `NewPayloadRequest` root,
  chain ID, schema and outcome using the pinned contract. Demonstrate
  proof request, generation, gossip, verification and retrieval, and
  observe the behavior for missing, delayed or mismatched proofs. For
  the current supplementary verification draft, retain EL execution
  as the validity path. ETH2030's BAL-bound helper currently has a test
  caller. The live provider lifecycle remains open. [EIP-8025](https://eips.ethereum.org/EIPS/eip-8025) · [Lighthouse proof configuration](https://github.com/sigp/lighthouse/blob/1a704e58f5c4ab82252fab46e35789d868a53211/beacon_node/src/cli.rs)

Treat `eth/70`, `eth/71`, and canonical binary state / witness gas as
EL work with separate wire and state acceptance. Schedule CL changes
for these items when an upstream Engine or data-availability contract
requires them. [EL protocol registration](../pkg/eth/handler.go) · [Track 4 remaining work](../CHECK.md)

## P2 · Decoupled consensus and PQ research

Start with identical captured available-chain inputs and compare the
current CL outcome with a pinned decoupled-consensus reference. Record
head, justified and finalized roots per slot for normal delivery, a
delayed execution envelope, missing data, competing branches, restart
and recovery after a partition. Progress to a separate two-client Lean
devnet once forkchoice, state transitions, signatures, and serialization
match the selected fixtures. Keep the experimental Engine endpoint,
peer network and validator keys isolated from the integration network.

Distinguish the EL's PQ transaction authentication and private key
accounts from the CL's proposed validator registry, lifecycle and
heartbeat. For a PQ CL milestone require spec-backed key registration
and updates, validator provenance, independent signature and aggregation
vectors, measured size and latency, and a restart test across key
transitions. Revisit the standalone CL decision when a pinned
specification, two-client interoperability, executable validator
duties and durable recovery are demonstrated. Until then the
Lighthouse integration is the release path. [leanSpec](https://github.com/leanEthereum/leanSpec) · [Lean consensus workstreams](https://leanroadmap.org/) · [EF protocol priorities](https://blog.ethereum.org/2026/09/07/protocol-priorities)
