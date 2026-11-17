# ETH2030 next-stage work plan

Snapshot checked on 2026-11-17. The next milestone is a reproducible
Gloas/Amsterdam and Heze/Bogotá client pair on reviewed revisions, with the
pinned Frame transaction executing through the native EL. Finish the missing
independent-client and recovery checks before describing that milestone as
interoperable. Track 3 state sync can advance in parallel. Track 4's private
profile remains the regression control while public state and proof work
continues on separate profiles.

This plan updates the execution order in the
[devnet upstream implementation plan](devnet-upstream-implementation-plan.md).
The [fork audit](devnet-fork-upstream-audit.md#feature-level-disposition)
assigns implementation ownership. Retain those feature boundaries when
preparing the next review branches.

## Source and evidence snapshot

| Area | Current state | Consequence |
| --- | --- | --- |
| ETH2030 mainline | `alt-research/eth2030:master` is [`ad35748f`][track4-pr], the Track 4 merge. The latest implementation branches are on the fork. GitHub reported no open PRs on the upstream ETH2030 repository when checked. | Prepare review branches from the current mainline before scheduling promotion. |
| Native Bogotá | [`db91e7ef`][bogota-branch] includes the Amsterdam slot commitment, native V6/FCU V5 retention, and the follow-up guards. Real local HTTP and FileDB tests cover deferred validation, repeated views, restart, malformed attributes and sync completion. | Qualify the patched native path on wire with Lighthouse. Keep independent EL parity as its own gate. |
| CL integration | Published fork `eth2030` remains [`1bf89a4fc`][cl-published]. [`413541ba`][cl-candidate] contains the previous upstream merge and exact-request Req/Resp deduplication. | Use the staging branch to prepare the next integration pair. |
| Live devnet evidence | The [control artifacts][controls-evidence] record Fulu slot 64 finality and original receipt readback after an EL restart with Lighthouse `v8.3.3` and ETH2030 `e628c770`. With CL `bc56927f` and EL `b7944602`, Gloas self-build verified at slot 4 and ordinary Heze FOCIL delivered a transaction from an isolated mempool through the slot 64 list into the finalized slot 65 proposal. | Re-run the relevant cases before attributing these results to the patched tips. Gloas external builder, PTC and blob-availability evidence remains open. |
| Frame | [`20b4bff1`][frame-branch] carries the Bogotá fixes and pinned seven-field codecs. The [pool admission][frame-pool] and [execution][frame-execution] paths both explicitly reject the pinned format. | Start implementation at the native activation and transition boundary. |
| Third-party services | Upstream Spamoor `f0f83695` matches the base Frame bytes and signing hash. Its scalar nonce boundary differs. The released Go codec read live Gloas slot 32 SSZ and rejected Heze slot 64 at `HistoricalRoots`. | Qualify ordinary upstream services separately. Keep a matched Heze codec until a compatible CL and Dora pair passes replay. |

## P0.1 · Package the fixes and qualify the patched pair

Prepare four review units.

| Review unit | Scope and dependency |
| --- | --- |
| EL Amsterdam/Bogotá | Review [`ad35748f..db91e7ef`][bogota-branch]. Check the slot commitment, V5/V6 and FCU version split, retained list views, and post-sync head ownership together. |
| Devnet controls | Start from current mainline and carry the control-only delta `e628c770..2239039e` from [the control branch][control-branch]. Keep the capped Heze profile changes with their declared Bogotá prerequisite. The branch history also carries the Frame codec commit `67aef736`, so prepare the control review from the scoped delta. |
| Lighthouse | Refresh [`413541ba`][cl-candidate] against [`3777cfd7`][cl-upstream] in an isolated staging worktree. A Git merge simulation showed a conflict in `beacon_node/beacon_chain/src/beacon_chain.rs` where upstream changed FCU handling for proof relay. Preserve the executed-ancestor and list verdict semantics while adopting the upstream relay policy. Include upstream's new Gloas column reconstruction and slot-validation changes in the next PeerDAS regression. |
| Documentation | Review the roadmap and progress records separately. The existing [documentation branch][docs-branch] also changes the devnet README and can overlap the control PR there. |

Before an integration promotion, build and label the actual reviewed EL and
CL heads. Run Fulu finality and EL restart, Gloas at genesis and a scheduled
Fulu-to-Gloas transition, then ordinary Heze FOCIL and Req/Resp recovery after
deliberate gossip loss. Exercise a staged native V6 payload whose FCU arrives
after restart, with both true and false inclusion verdicts and an empty
beacon head over the executed ancestor. The peer request cases should cover
the same slot with different dependent roots, partial replies and alternate
peers. Capture both clients' wire responses and canonical block, state and
receipt results. Run required CI on the review heads and retain the prior
image pins for rollback.

Run the Gloas builder/availability and independent Bogotá checks alongside
this qualification. For Gloas, require an actual signed nonzero builder bid,
envelope and payment. Check PTC decisions under late or missing envelopes,
bad BAL rejection and data-column delivery/custody with blobs. For Bogotá,
qualify the independent EL's actual `ACCEPTED` and FCU wire behavior against
the same draft before comparing the local verdict. A same-implementation
pair establishes the control. Mark independent coverage open if a compatible
peer remains unavailable. Use the [existing PR plan][pr-plan] for fixture
entry points and bounded runs.

## P0.2 · Complete native Frame execution and exercise it through the CL

Start from the reviewed native Bogotá code, then carry the Frame codec
commits forward from the integration branch. Continue against the
[pinned EIP-8141 draft][frame-draft] and [Bogotá Engine draft][bogota-draft].
The [Hegotá meta EIP][hegota] lists FOCIL and Frame as scheduled. Treat
keyed nonces, recent roots and FOCIL VOPS eligibility as separately versioned
requirements when preparing fixture variants.

| Order | Implementation scope | Required result |
| --- | --- | --- |
| 1 | Add the pinned Frame transition at `applyTransactionInternal`. Use the shared state journal and execution path for authorization, frame approvals, gas reservation/refill, rollback, escrow and settlement. | Exact state roots, sender and payer deltas, gas dimensions and receipt entries for ordinary, sponsored, reverted and atomically rolled-back cases. Include independent vectors for invalid signatures and the scalar nonce boundary. |
| 2 | Connect fork activation, public-pool admission, payload construction, Engine validation, blob sidecars and historical readback. | Reject the new format before activation. After activation, import the same payload through the builder and a second EL, agree on receipts and state after restart, and reject bad sidecar/order controls. |
| 3 | Produce ordinary, delegated-sender, sponsored and blob Frame workloads with an upstream-based generator. Replay historical envelopes through Lighthouse and compare with the explorer's receipt view when its matched codec is ready. | Exact signature hashes and canonical receipts. Preserve ordered Frame and EIP-4844 commitments and prove fresh-node history and optimistic finalized sync against a compatible source. Verify publication through independent execution and consensus clients as separate checks. |

Keep self-build sponsorship and compulsory inclusion as distinct scenarios.
Run sponsored FOCIL once the test fixes an eligible Frame shape under an
agreed rule and includes a policy-permitted omission control.
[EIP-8369 proposal][vops] · [Frame acceptance details][pr-plan]

## P1 · Parallel node reliability and service migrations

**Track 3 state sync.** Begin with the independent Geth and Lighthouse
late-join control for full then snap mode. Observe the native P2P and Engine
traffic, moving finality, canonical proofs, receipts, restart and a
post-pivot transaction. Then implement authenticated reuse and durable
partial account/storage cursors across A → B → C pivots. Measure request and
byte savings across an interrupted run and reject stale proofs, missing
leaves and a conflicting finalized target. Run the disk and heap scale tests
before promoting the storage path. `FileDB` currently indexes every value
in memory, and execution and serving use additional in-memory structures.
A bounded reader and replay path must be measured at download, adoption,
serving and reopen. [Track 3 acceptance][track3] · [Current FileDB][filedb]

**Ordinary explorer and workload profiles.** Qualify upstream Dora
`v1.25.0` and Spamoor `f0f83695` one at a time against a fixed Fulu/Gloas
control. Require canonical UI/API readback for Dora and included workloads,
bounded receipt failover and correct funding for Spamoor. Archive historical
EIP-7701 workloads. For Heze, first match the SSZ contract on the CL and Go
codec pins, then replay competing proposal roots, dropped events, pruning
and restart. Retire the local Go codec when the published codec covers every
active profile. Retire the retained-view adapter after the explorer gate
passes with that endpoint disabled.

**Package and CI.** Trial the newer ethereum-package pin separately and
inspect the generated genesis before promoting it. Its Frame-only handling
can disable CL Heze based on profile settings or image names. Require both
Heze and Bogotá activation in the combined profile. For the integration
jobs, record named protocol assertions and exact terminal states, including
Assertoor failures/timeouts. Use stable independent fixtures and mutation
controls for each new acceptance assertion. Run at most one capped
devnet at a time and verify removal of the owned containers, networks and
volumes after success, failure or timeout. Keep the sanitized result and
its source/image manifest. [Devnet package and verifier plan][pr-plan]

## P2 · Track 4 and longer-horizon work

Keep the merged PQ and binary-overlay implementation as the Fulu regression.
The current overlay persists a binary root and conversion cursor and
checks them against replay on reopen. For public adoption, agree the
EIP-7864 key layout and precompile addresses, implement authenticated
frozen-state reads and direct checkpoint recovery, and finish EIP-4762
witness charging. Reuse the Track 3 storage profiles to verify resource
bounds on a growing state. Keep the private profile's wire formats available
for regression. [Current checkpoint readback][checkpoint] · [Track 4 audit][track4]
· [EIP-7864][binary] · [EIP-4762][witness]

For PQ attestation adoption, source validator identity and keys from an
authenticated committee and exercise key transitions and restart. Define
separate protocol statements and independent vectors for STARK, lattice and
aggregation providers before wiring them into a deciding consumer. Standard
blob verification continues to use KZG. Keep optional proof relay, Quick
Slots and Lean consensus research separately pinned. The Hegotá meta EIP
lists EIP-8025 as declined for that fork. [Strawmap][strawmap] · [Hegotá schedule][hegota]

[track4-pr]: https://github.com/alt-research/eth2030/pull/55
[bogota-branch]: https://github.com/fyInALT/eth2030/compare/ad35748f...db91e7ef
[control-branch]: https://github.com/fyInALT/eth2030/tree/feat/devnet-upstream-controls
[controls-evidence]: https://github.com/fyInALT/eth2030/tree/929870c9/pkg/devnet/kurtosis/fixtures
[frame-branch]: https://github.com/fyInALT/eth2030/tree/integrate/bogota-deferred-frame
[frame-pool]: https://github.com/fyInALT/eth2030/blob/20b4bff13be806a7ed7067299ca80d7c6cafa380/pkg/txpool/txpool.go#L541-L543
[frame-execution]: https://github.com/fyInALT/eth2030/blob/20b4bff13be806a7ed7067299ca80d7c6cafa380/pkg/core/execution/processor.go#L817-L819
[cl-published]: https://github.com/jiayaoqijia/lighthouse/commit/1bf89a4fc933d58828c39bcda3fe6eb481d680f5
[cl-candidate]: https://github.com/jiayaoqijia/lighthouse/commit/413541ba9c0aa88f372777ede19acf961dc7ba1f
[cl-upstream]: https://github.com/sigp/lighthouse/commit/3777cfd74868563842b9a535e34498f82cb16fe1
[docs-branch]: https://github.com/fyInALT/eth2030/tree/docs/cl-iteration-roadmap
[pr-plan]: devnet-upstream-implementation-plan.md
[frame-draft]: https://github.com/ethereum/EIPs/blob/dc5d5fa688b7fe87b95a907fcc128b4fde2f4d2f/EIPS/eip-8141.md
[bogota-draft]: https://github.com/ethereum/execution-apis/blob/981aa66d79364be8322a9c358aaf0a571cca0873/src/engine/bogota.md
[hegota]: https://eips.ethereum.org/EIPS/eip-8081
[vops]: https://eips.ethereum.org/EIPS/eip-8369
[track3]: track3-remaining-acceptance.md
[track4]: CODEBASE_AUDIT.md#track-4-completion-audit-2026-11-14
[filedb]: https://github.com/alt-research/eth2030/blob/ad35748f01361483dbe4b59ad4e858021804f29c/pkg/core/rawdb/filedb.go#L39-L48
[checkpoint]: https://github.com/alt-research/eth2030/blob/ad35748f01361483dbe4b59ad4e858021804f29c/pkg/core/chain/blockchain.go#L302-L348
[binary]: https://eips.ethereum.org/EIPS/eip-7864
[witness]: https://eips.ethereum.org/EIPS/eip-4762
[strawmap]: https://strawmap.org/
