# Lighthouse consensus client gap audit

Audited on 2026-11-16. The Lighthouse `eth2030` branch has live evidence for ordinary Heze
FOCIL inclusion and signed-list recovery after missed gossip. The next integration milestone
requires real EL handling of deferred Bogotá inclusion verdicts and execution of the pinned
EIP-8141 Frame format. Gloas builder and data-availability scenarios can be tested in parallel
once the reference binaries and fixtures are pinned.

## Audited revisions

| Component | Revision and scope |
| --- | --- |
| Lighthouse `origin/eth2030` | [`1bf89a4fc`][cl] has the Heze fork-choice, network and historical Frame changes. |
| Lighthouse upstream | [`e723ca55f`][upstream] is one upstream commit beyond the fork's baseline. It adds a light-client fork-boundary fix. Stable `v8.3.3` is the Fulu control used in the earlier baseline run. |
| ETH2030 EL | `master` at [`ad35748f`][el-master] contains Track 4. The published `feat/bogota-inclusion-list-wire` branch is at [`42e7ada9`][bogota-el]. Its descendant, `feat/frame-8141` at [`e628c770`][frame-el], includes the pinned Frame encoding and the memory-bounded Heze fixture. |
| Dora | [`12a0198`][dora] persists retained Heze inclusion views by block root. |

The [implementation plan](lighthouse-hegota-implementation-plan.md) holds the original
contract pins and its initial baseline. The results below incorporate the subsequent direct-genesis and recovery runs. The [legacy branch plan](lighthouse-legacy-branch-merge-plan.md) records
the disposition of the March fork changes. The [status log](../STATUS.md#heze-consensus-validation-2026-11-16)
holds the run evidence. The [checklist](../CHECK.md#consensus-client-follow-up-2026-11-16)
tracks open deliverables.

## Verified behavior

- **Gloas and Amsterdam.** Two node pairs crossed the fork with matching self-built
  envelopes and execution data. A direct-genesis run agreed on the execution hash, state
  root and BAL hash. The Fulu checkpoint provides a pre-Gloas control.
- **Ordinary Heze FOCIL.** Both ELs were isolated at submission and inclusion. The proposing
  EL had no transaction before receiving the signed list. The transaction appeared in the
  next-slot proposal and reached the same finalized block and receipt on both pairs. An
  Engine control built from the same parent with and without list delivery. Both ELs also
  passed the Amsterdam V5 to Bogotá V6 fork boundary checks.
- **Heze list recovery and restart.** A capped run deliberately dropped a receiver's gossip.
  A self-build bid triggered Req/Resp recovery of the same signed list and committee positions.
  The recovered list stayed late. After reopening the receiver, the imported block and the
  prior finalized checkpoint remained readable.
- **Directed regressions.** The CL's controlled Engine HTTP test binds a deferred
  `VALID,false` result to the executed beacon ancestor, including an empty head and
  fork-choice reload. The CL historical envelope test checks ordered Frame blob commitments
  and rejects reversed commitments and altered payload hashes. The EL Frame encoding and
  Bogotá V6 targeted Go tests passed during the audit.
- **Dora retention.** A prior capped run verified the stored inclusion-list positions and
  transactions after Lighthouse pruned its view and Dora restarted.

## P0 · Integration and release gaps

### Deferred Bogotá inclusion verdicts

The CL already distinguishes execution validity from inclusion-list satisfaction. The
[companion EL V6 handler][bogota-handler] checks the list only for an immediate `VALID`
result and retains no list when a result is `ACCEPTED`. The [FCU verdict lookup][bogota-cache]
reads an in-memory Boolean keyed by execution hash and treats a missing entry as satisfied.
The [Bogotá Engine contract][bogota-spec] requires retaining the list on `ACCEPTED` and
using it if FCU later validates the payload.

Close the retention and recomputation path in the EL, then exercise it from Lighthouse
against a real EL and an independent implementation that supports the same contract.
Use one execution payload with separate satisfying and unsatisfying views, a delayed
validation transition, and an empty beacon head over the executed ancestor. Capture
`engine_newPayloadV6` and `engine_forkchoiceUpdatedV5` responses, the reloaded CL's
fork-choice decision and the execution parent selected by the next proposer. Check that a
view supplied for a competing branch cannot affect the verdict. Cover EL restart and
the specified lifetime of a pending list.

### Pinned Frame execution and CL historical import

The EL supports the seven-field envelope, signing vectors, fee arithmetic and split-gas
receipt readback. Both [mempool admission][frame-pool] and [execution][frame-execution]
explicitly reject the new format. Lighthouse verifies ordered Frame blob hashes in its
local envelope checks. The dependency is the EL state transition, including per-frame
state-gas accounting and refills, rollback, fee escrow and sponsor settlement, followed by
fork activation, mempool policy, sidecar handling, Engine validation and replay.

Run plain, sponsored and blob-bearing Frame transactions using two compatible ELs.
Compare canonical receipt bytes, payer balance changes and state roots. Put a blob Frame and
EIP-4844 transaction in one envelope and check commitment order through live import,
historical backfill and optimistic finalized sync. Include incorrect hashes and withheld
data as separate controls. Verify readback after restart and on a fresh node.
The sponsored FOCIL case also needs a pinned eligibility rule and a paired allowed-omission
case. [EIP-8141][frame-spec], [EIP-7805][focil-spec], [EIP-8369][vops], [Hegotá Meta EIP][hegota]

### Reproducible Gloas and Heze acceptance

Pin the CL and EL revisions, image digests, Kurtosis package and genesis generator
revisions, fork schedule, and sanitized Engine and Beacon API fixtures. The
[Heze profile][heze-profile] uses local image tags and caps the ELs and beacon nodes at 2 GiB
each and the VCs at 1 GiB each. The [default EF spec runner][ef-runner] skips Heze.
Add a pinned cross-client wire fixture and an explicit Heze vector job as compatible
vectors become available.

Use the same pinned baseline for direct Gloas genesis and scheduled fork activation.
This Gloas lane uses the Amsterdam Engine contract and can run with the Bogotá branch
before Frame execution lands.
Exercise a nonzero external builder bid and payment, PTC decisions with timely, late
and missing envelopes, malformed BAL rejection, and custody and availability with
blob data columns. Retain the actual builder, Engine and Beacon observations and the
finalized execution hash on each pair. Two ETH2030 pairs establish same-client acceptance.
Cross-client acceptance requires an independent implementation of the pinned contract and
remains open when a compatible client is unavailable. [Gloas specification][gloas-spec], [EIP-7928][bal-spec]

## P1 · Lighthouse follow-up

- **Upstream and light clients.** Integrate upstream [#10401][upstream-pr] and rerun the
  Gloas light-client finality regression across a fork boundary. The upstream test skips
  Heze, and the fork still returns [`HezeNotImplemented`][heze-header] for header construction.
  Implement Heze headers, bootstrap, optimistic and finality updates once the schema and
  proof vectors are pinned. Verify the transition while the finalized checkpoint is
  still on the previous fork.
- **Validator duty recovery.** The [VC][vc] signs once and broadcasts to redundant beacon
  nodes. Cover a VC restart during the duty, head changes, skipped slots, a remote signer
  and redundant BNs. Verify the signed bytes stay consistent across retries and that
  deadline misses and equivocation are handled correctly.
- **List recovery from peers.** The [Req/Resp client][reqresp] asks the peer associated with
  the bid and removes its pending request after error or stream end. Cover alternate-peer
  recovery after failure or partial responses, competing dependent roots at a single slot,
  and wrong or duplicate entries. Verify the requested committee positions, signature,
  original timeliness and request-retention boundary. [Pinned Heze networking spec][heze-p2p]
- **Fresh full and snap late join.** Start with a new database, use independent source
  clients, and validate the adopted head, safe and finalized hashes, roots and receipts.
  Use the independently built Geth source and Lighthouse-driven ETH2030 receiver proposed
  in the [Track 3 acceptance design](track3-remaining-acceptance.md). Check finality advances,
  authenticated state readback and progress across interruption and restart. Lighthouse owns
  historical envelope and optimistic finalized sync checks. ETH2030 owns the separate snap
  moving-pivot and authenticated state-adoption tests.
- **Dora epoch statistics.** Align the BeaconState SSZ decoder with the pinned Gloas and
  Heze layouts. Compare epoch statistics against the Beacon API across the fork and after
  a restart. Preserve the already verified historical inclusion-list readback.

## Later roadmap

Quick Slots needs a scheduled slot-duration transition with Engine, envelope and
validator duty timing measured against the specified deadlines. EIP-8025 already has a
proof-engine and gossip path in Lighthouse. The separate proof profile still needs a real
provider, binding to the payload and evidence for missing, delayed and mismatched proofs.
The Hegotá meta EIP lists Quick Slots as considered and EIP-8025 as declined for that fork.
[EIP-8198][quick-slots], [EIP-8025][proofs], [Hegotá Meta EIP][hegota]

Lean consensus and PQ validator work require pinned consensus vectors and two-client
evidence for key registration and rotation, signatures and aggregation, fork choice,
finality and restart. The private Track 4 PQ transaction devnet covers EL transaction
authentication. [CL roadmap](cl-iteration-roadmap.md#p2--decoupled-consensus-and-pq-research)

## Implementation order

1. Start from Lighthouse `eth2030` at `1bf89a4fc` and integrate upstream `e723ca55f` on
   that development line. Use EL `feat/bogota-inclusion-list-wire` at `42e7ada9` for the
   Bogotá changes, and carry the fix into the descendant Frame branch before the combined
   acceptance run. Freeze the cross-client baseline and wire fixtures while implementing
   retained Bogotá verdicts and testing the real V6 to FCU transition.
2. Run Gloas builder, PTC, BAL and data-column scenarios. Implement the EL Frame state
   transition and run real Frame import, sponsorship, FOCIL and backfill as its paths land.
3. Close VC and Req/Resp recovery, Heze light-client validation, independent late join and
   Dora epoch-statistics readback.

The audit used live remote refs, local code, targeted EL tests and prior CL devnet records.
There was no new CL build or devnet run during the audit.

[cl]: https://github.com/jiayaoqijia/lighthouse/commit/1bf89a4fc933d58828c39bcda3fe6eb481d680f5
[upstream]: https://github.com/sigp/lighthouse/commit/e723ca55fe0da47a145967bb31a30be74b44079d
[el-master]: https://github.com/alt-research/eth2030/commit/ad35748f01361483dbe4b59ad4e858021804f29c
[bogota-el]: https://github.com/fyInALT/eth2030/commit/42e7ada99372650111b63b38bd2606c3eab19ece
[frame-el]: https://github.com/fyInALT/eth2030/commit/e628c770df98ba9ad269de74344afcdf4e6a92a3
[dora]: https://github.com/jiayaoqijia/dora/commit/12a0198f237c8bf1973256ae5c2498d7cef90587
[upstream-pr]: https://github.com/sigp/lighthouse/pull/10401
[bogota-handler]: https://github.com/fyInALT/eth2030/blob/42e7ada99372650111b63b38bd2606c3eab19ece/pkg/engine/bogota.go#L67-L78
[bogota-cache]: https://github.com/fyInALT/eth2030/blob/42e7ada99372650111b63b38bd2606c3eab19ece/pkg/node/backend/engine_bogota.go#L18-L46
[bogota-spec]: https://github.com/ethereum/execution-apis/blob/981aa66d79364be8322a9c358aaf0a571cca0873/src/engine/bogota.md
[frame-pool]: https://github.com/fyInALT/eth2030/blob/e628c770df98ba9ad269de74344afcdf4e6a92a3/pkg/txpool/txpool.go#L541-L543
[frame-execution]: https://github.com/fyInALT/eth2030/blob/e628c770df98ba9ad269de74344afcdf4e6a92a3/pkg/core/execution/processor.go#L817-L819
[frame-spec]: https://github.com/ethereum/EIPs/blob/dc5d5fa688b7fe87b95a907fcc128b4fde2f4d2f/EIPS/eip-8141.md
[focil-spec]: https://eips.ethereum.org/EIPS/eip-7805
[vops]: https://eips.ethereum.org/EIPS/eip-8369
[hegota]: https://eips.ethereum.org/EIPS/eip-8081
[heze-profile]: https://github.com/fyInALT/eth2030/blob/e628c770df98ba9ad269de74344afcdf4e6a92a3/pkg/devnet/kurtosis/configs/heze-focil.yaml
[ef-runner]: https://github.com/jiayaoqijia/lighthouse/blob/1bf89a4fc933d58828c39bcda3fe6eb481d680f5/testing/ef_tests/src/handler.rs#L46-L54
[gloas-spec]: https://github.com/ethereum/consensus-specs/blob/313cbfe561517a81eb11607fea7c6985e562eaa9/specs/gloas/beacon-chain.md
[bal-spec]: https://eips.ethereum.org/EIPS/eip-7928
[heze-header]: https://github.com/jiayaoqijia/lighthouse/blob/1bf89a4fc933d58828c39bcda3fe6eb481d680f5/consensus/types/src/light_client/light_client_header.rs#L109-L112
[vc]: https://github.com/jiayaoqijia/lighthouse/blob/1bf89a4fc933d58828c39bcda3fe6eb481d680f5/validator_client/validator_services/src/inclusion_list_service.rs#L77-L135
[reqresp]: https://github.com/jiayaoqijia/lighthouse/blob/1bf89a4fc933d58828c39bcda3fe6eb481d680f5/beacon_node/network/src/service.rs#L1012-L1067
[heze-p2p]: https://github.com/ethereum/consensus-specs/blob/313cbfe561517a81eb11607fea7c6985e562eaa9/specs/heze/p2p-interface.md
[quick-slots]: https://eips.ethereum.org/EIPS/eip-8198
[proofs]: https://eips.ethereum.org/EIPS/eip-8025
