# Consensus client iteration after Track 4

This plan is based on ETH2030 master `ad35748` and the Lighthouse revisions
below, inspected on 2026-11-16. The acceptance checks describe future work.
[Track 4 merge and CI](https://github.com/alt-research/eth2030/pull/55)

## Decision and first milestone

Keep Lighthouse as the release CL and use the native Go 3SF/PQ observer
for research. The next milestone is a **nonempty Heze inclusion list that
changes what the proposer builds and what the attesters vote for**. In
parallel, establish a reproducible Gloas/Amsterdam baseline.

| Order | Deliverable | Acceptance evidence |
|---|---|---|
| 1 | Pin the Lighthouse reference and Engine contract fixtures. | Record client revisions and image digests, genesis and fork settings, and actual Engine requests and responses. |
| 2 | Complete the Bogotá Engine methods in the native EL. | The EL returns the expected list shape, retains transactions from V5 payload attributes, and returns separate execution validity and inclusion satisfaction in V6. |
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

## Client direction

Use upstream Lighthouse as the production CL codebase and retain an ETH2030
integration branch with a small patch queue. Lighthouse's beacon node and
validator client own validator duties, consensus state, gossip validation,
forkchoice and finalization. ETH2030 owns payload construction, execution,
receipt and state commitments, and state sync. The authenticated Engine API
connects them. Keep validator duties and slashing protection with the
Lighthouse validator client or its established external signer.

Follow stable releases for operator testing and pinned upstream unstable
revisions for fork devnets. The local `integrate/unstable-202611` branch at
`2c468be9c` carries two commits over upstream `25b9b7541`. Its net tracked
change is one documentation symlink. The then-current upstream unstable
tip is `1a704e58f`, 20 commits beyond that base, and stable is `57c2abc13`
(v8.3.3). The older `feat/fix-native-aa` branch has 69 commits absent from
the new unstable tip. Use it to find specific behavior and integration
requirements. Review old patches individually against upstream FOCIL,
Gloas envelope processing, light-client support and the public Beacon API
before carrying any forward. In particular, check the envelope endpoint
against Dora's actual consumer before deciding whether to retain it.
[Lighthouse upstream development history](https://github.com/sigp/lighthouse/commits/unstable/) · [Lighthouse v8.3.3](https://github.com/sigp/lighthouse/releases/tag/v8.3.3)

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

The first implementation change belongs in ETH2030's native Engine HTTP
backend. At `ad35748` the dispatcher and capability exchange omit
`engine_newPayloadV6`. Lighthouse upstream explicitly requires V6 when
importing a Heze payload. The EL compatibility parser discards nonempty
Lighthouse `inclusion_list_transactions`. The EL's
`engine_getInclusionListV1` currently emits a JSON object with a
`transactions` member. The Bogotá API specifies an array and the pinned
upstream Lighthouse type is a transparent transaction-list wrapper. These
are source-verified contract gaps. The existing private Track 4 devnet
uses Fulu as its final configured CL fork.
[EL dispatcher](../pkg/engine/handler.go) · [EL compatibility parser](../pkg/engine/lighthouse_compat.go) · [EL list response](../pkg/engine/inclusion_list.go) · [Lighthouse Engine client](https://github.com/sigp/lighthouse/blob/1a704e58f5c4ab82252fab46e35789d868a53211/beacon_node/execution_layer/src/engine_api/http.rs) · [Bogotá Engine API](https://github.com/ethereum/execution-apis/blob/main/src/engine/bogota.md)

Implement the contract in this order.

1. Pin the CL and EL API draft tuple and agree on explicit forks, method
   capabilities, production attribute encoding and exact transaction-list
   encodings. Capture request and response fixtures from both clients.
2. Make committee list acquisition return the expected list shape and
   enforce the draft's selection and size rules. Preserve the list from
   proposer-side `forkchoiceUpdatedV5` to payload construction and bind it
   to the candidate context. Include slot, parent, target gas and list
   content when identifying builds. Keep committee authorization, gossip,
   and the subjective set of timely lists in the CL.
3. Implement `newPayloadV6` and `PayloadStatusV2` in the native EL.
   Compute inclusion satisfaction independently of execution validity.
   Retain the list for accepted payloads and use it on the applicable
   revalidation path. Include the list context when caching satisfaction
   because different CL callers can evaluate the same execution block
   against different lists. Return execution validity and list
   satisfaction as separate fields. Apply the draft's treatment of
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
