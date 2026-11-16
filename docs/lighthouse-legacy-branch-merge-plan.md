# Lighthouse legacy branch integration plan

Code and remote refs checked on 2026-11-16. The delivery target is
`jiayaoqijia/lighthouse:eth2030` at `3e3978d4e`, based on upstream
`unstable` at `1a704e58f`. The historical source is
`feat/fix-native-aa` at `6552c7124`. The EL integration target is
ETH2030 `origin/master` at `ad35748`.

The Lighthouse fork already contains `65b0186da` (the iFlow symlink)
and `6e5317104` (the adapted Heze self-build inclusion-list snapshot).
The old CL branch has 68 commits outside the new branch and changes 119
files from the shared ancestor. Git's patch comparison found no exact
equivalent for those 68 commits. That comparison does not account for
upstream replacements or the adapted self-build change. An in-memory
trial merge reported 76 conflicts, including 73 content conflicts, two
modify/delete conflicts and one add/add conflict. Preserve the current
upstream consensus types, storage and networking. Port the outstanding
behavior by feature on branches cut from `eth2030`.
[Current fork](https://github.com/jiayaoqijia/lighthouse/tree/3e3978d4eacda648b32ddc45ac32222b00afc6ba) · [Historical fork](https://github.com/jiayaoqijia/lighthouse/tree/6552c7124c25f9feca79f4deffcd36375a8a0b40)

## Where the live paths are

| Area | Current implementation and integration consequence |
|---|---|
| Lighthouse VC duties and publication | `duties_service.rs` fetches and caches IL assignments with their dependent root and exposes `get_il_duties_for_slot`. BN `POST /eth/v1/validator/inclusion_list` verifies, imports and gossips already-signed lists. `ExecutionLayer::get_inclusion_list_v1` has no production caller outside the execution layer. Connect the duty, BN candidate retrieval, signing and publication through the current VC and BN APIs. |
| Lighthouse Heze list and bid handling | The list store keys entries by slot, dependent root and validator. Commit `6e5317104` already binds local bid bits to the same previous-slot transaction view sent to Engine. `BeaconChain::is_inclusion_list_bits_inclusive` and `InclusionListStore::get_signed_inclusion_lists` have test call sites but no production bid or P2P consumer at this revision. Connect the existing helpers to incoming verification and current Req/Resp. |
| Lighthouse Engine and fork choice | The Engine client implements the Heze request variant and parses the satisfaction Boolean. Live envelope notification in `payload_notifier.rs` still constructs Gloas. The historical envelope hash helper in `payload_envelope_verification/mod.rs` is a separate local path, also built around Gloas. Status conversion reduces a matching `VALID` response to plain validity, and the fork choice has no consumed Heze satisfaction result. Wire the live request and fork choice state, and review local backfill separately. |
| ETH2030 production Engine | `pkg/node/node.go` constructs `nodebackend.NewEngineBackend` and passes it to `engine.NewEngineAPI`. Its imports run through `pkg/node/backend/engine_backend.go` to `Blockchain.InsertSyncedBlock`. The legacy `pkg/engine/backend.go` global list and `INCLUSION_LIST_UNSATISFIED` return belong to a different backend. The node backend itself rejects nonempty `engine_newInclusionListV1` list storage. Implement the new request-context path at the node backend. |
| ETH2030 live payload build | Canonical camelCase attributes already pass the list via `ForkchoiceUpdatedV4` into `BuildBlockAttributes.InclusionListTxs`. `generatePayloadID` already includes parent, gas limit, list and slot. Preserve that path and thread the caller's target gas limit, since it currently uses the parent's gas limit. The old snake_case parser drops list content. The current wire handler returns an object for `getInclusionListV1`, and dispatch and capability advertisement omit `newPayloadV6`. |
| ETH2030 Frame and blobs | The existing Go Frame representation is an earlier draft. The live `EngineAPI.NewPayloadV5` already compares the expected hashes against transaction hashes through `validateBlobHashes` before invoking `ProcessBlockV5`. `Transaction.BlobHashes`, the builder's blob classification and the node's blob bundle construction are currently tied to EIP-4844's type. Extend the shared wire check and build/sidecar paths to include Frame hashes, and reuse the ordered comparison for V6. |
| Dora and test senders | Dora `6f702f2` loads Gloas/Heze execution data from EL by bid hash. Its list view calls the March `/blocks/{block_id}/inclusion_lists` route and treats response order as committee order. Replace the list consumer using the upstream event and actual committee position. Spamoor's `scenarios/frametx` and the reference implementation's `e2e/tests` supply historical test scenarios for re-encoding against the chosen draft. |

[Production Engine setup](../pkg/node/node.go#L911) · [Payload wire validation](../pkg/engine/server.go#L233) · [Import and list backend](../pkg/node/backend/engine_backend.go) · [Build attributes](../pkg/node/backend/engine_fcu.go) · [Payload IDs and blob bundles](../pkg/node/backend/engine_payload.go) · [Block builder](../pkg/core/block/builder.go) · [Lighthouse envelope notifier](https://github.com/jiayaoqijia/lighthouse/blob/3e3978d4eacda648b32ddc45ac32222b00afc6ba/beacon_node/beacon_chain/src/payload_envelope_verification/payload_notifier.rs)

Both historical EL branch tips, `fy/feat/track2-hegota` at `c603fc5`
and `origin/feat/focil-8141-integration` at `f585028`, are ancestors of
`ad35748`. Their work is already reachable from the EL target. Start
from the current EL branch and repair its consumers. The separate
EIP-8141 reference repository provides earlier scenario intent and test
inputs, with the pinned draft and independent vectors as authority for
current expected results.

## Old CL commits by disposition

These groups cover the 68 old commits outside `eth2030`. A commit that
touches several features is listed by its primary purpose.

| Group | Historical commits | Decision on the current branch |
|---|---|---|
| Heze types, fork upgrade and build repairs | `ef61f0646`, `158eae369`, `1cd74dc96`, `ad2633012`, `c372247c7`, `20c4192e0`, `d008618a1`, `e7b53cc82`, `18057e2e5`, `d14f2cc83`, `c4f209a5a`, `b13c9256d`, `e9c05fbad`, `9a19332e2`, `0f2bcf5b2` | Retain upstream Heze SSZ, fork upgrade, lookahead and `HEZE_FORK_EPOCH`. Use the old serialization and fork-transition cases as tests after translating their fixtures. Handle the old epoch alias in a consumer only if a released profile still needs it. |
| List committee, gossip, store and timing | `03e83a2e4`, `d59e02ce5`, `c39cd5d61`, `efdd96683`, `d02bb60cc`, `ca970c6ee`, `7c17c510b`, `a54edf5aa` | Reuse upstream committee verification, dependent-root store, gossip and metrics. Carry over equivocation, deadline, deduplication and wrong-branch cases where current tests lack them. The old message uses a committee root and its old timing literals predate the pinned spec. |
| Validator list production and delivery | `8a0c2c431`, `5f20ef5c5`, `02e4ac799`, `7fb46a833`, `9108417e2`, `6552c7124` | Port this behavior first on the CL side. Reuse current duty cache, slot clock, validator signing conventions, BN fallback and publication route. Port the assigned-slot, head arrival, deadline and redundant-BN regressions. Derive the deadline from the pinned 7,500 basis points and bind signing and retry to the same dependent root. |
| Beacon API and explorer reads | `242aac219`, `7832fa06e`, `d6e30a691`, `a4f60250a`, `d13f6c10d` | Use upstream signed-list submission, inclusion-list events and `/eth/v1/beacon/execution_payload_envelopes/{block_id}`. Move Dora list reads to the new consumer path. The old envelope commit also contains a skipped-slot recovery fix. Preserve that scenario against upstream envelope retrieval. |
| Engine wire model | `7a287c8f2`, `c9b438de5`, `7ca581d91`, `0fe6536ed`, `2df596172` | Port the intended flow using Bogotá `forkchoiceUpdatedV5`, Amsterdam-inherited `getPayloadV6` and Bogotá `newPayloadV6`. The old `PayloadAttributesV4` list field and `INCLUSION_LIST_UNSATISFIED` enum use earlier wire rules. Preserve `VALID` separately from `inclusionListSatisfied=false`. |
| P2P list retrieval | `6a13ad284`, `50bf344a6` | Reuse request lifecycle and quota test ideas. The old wire method uses singular `inclusion_list_by_committee_indices` with slot and bitvector. Implement pinned `inclusion_lists_by_indices/1` with slot, dependent root and indices. Include one signed list per successful chunk and apply the pinned limits and fork context. |
| Proposal bits and bid checks | `7d7f3dae5`, `3e198341e`, `208442fc8` | Retain `6e5317104` for self-build and port the remaining production call sites for inclusivity. The old block error assumes peer penalty. The pinned gossip-bid rule ignores a bid missing bits from the timely view. Derive block, direct-bid and local selection checks from their own pinned rules. |
| Fork choice satisfaction | `7872ca52a`, `b18b208ec`, `6109e1f0f` | Adapt the non-extension behavior to current fork choice, proto-array and persisted state. Verify `should_extend_payload` and the next proposer's actual parent. Exercise pruning, optimistic-to-verified transitions and restart. |
| Gloas PTC and full/empty branches | `d618dba48`, `d7502bff2`, `499d88646`, `935af7c6e`, `17b037992`, `b4079e2c8`, `db1e232b4`, `0b6a079e4`, `6eedae1e1`, `f6534d46e`, `64e6ef114`, `ae37e720a`, `4ac2ae6ea` | Keep upstream `on_payload_attestation`, timeliness/data-availability votes and full/empty execution head handling. Re-run old duplicate-vote, late envelope, parent hash and finalized hash regressions. |
| Envelope imports and fork boundaries | `230622966`, `ac4b12d1c`, `3a2b8bdc5`, `ae9deda56`, `35271545e` | Keep upstream pending-envelope and blob paths, envelope streaming and fork digest handling. Test data pending then arriving, a skipped parent, sync by root and range, and restart. |
| Light client | `168f42957`, `9977951a8` | Keep upstream Gloas execution-block-hash proof. Heze constructors still return `HezeNotImplemented`. Plan a separate proof/vector-backed Heze change when the corresponding contract is available. |
| Mixed AA-named change | `3b5fc06f6` | Read by behavior while implementing the groups above. It mixes Heze and FOCIL refactors and has no Frame decoder. Implement the Frame CL and EL changes against the pinned EIP-8141 draft. |

[Heze inclusion-list spec](https://github.com/ethereum/consensus-specs/blob/313cbfe561517a81eb11607fea7c6985e562eaa9/specs/heze/inclusion-list.md) · [Heze validator spec](https://github.com/ethereum/consensus-specs/blob/313cbfe561517a81eb11607fea7c6985e562eaa9/specs/heze/validator.md) · [Heze P2P spec](https://github.com/ethereum/consensus-specs/blob/313cbfe561517a81eb11607fea7c6985e562eaa9/specs/heze/p2p-interface.md) · [Bogotá Engine draft](https://github.com/ethereum/execution-apis/blob/981aa66d79364be8322a9c358aaf0a571cca0873/src/engine/bogota.md) · [Frame draft](https://github.com/ethereum/EIPs/blob/dc5d5fa688b7fe87b95a907fcc128b4fde2f4d2f/EIPS/eip-8141.md)

## Pull-request sequence

Create each CL topic branch from the current `eth2030` target and each
EL topic branch from current `origin/master`. Start from the code paths
above. Use `git show <old-commit> -- <path>` to extract behavior and tests.
Name the source commits in the PR and explain changed wire or fork
semantics. Each PR pairs its changed behavior with a targeted test and
the relevant existing suite.

A dependent PR can use pinned wire fixtures before its counterpart lands.
Once the prerequisite merges, update the dependent branch from its own
target and run the paired acceptance with both committed revisions.

| Work package | Depends on | Implementation and acceptance |
|---|---|---|
| Shared fixtures and baseline | Published branch and current EL | Record image, spec, genesis and fork references. Freeze Amsterdam `forkchoiceUpdatedV4`, `getPayloadV6`, `newPayloadV5` and Bogotá `forkchoiceUpdatedV5`, `getPayloadV6`, `newPayloadV6` with versioned golden requests and responses. Cover calls without attributes separately. Re-run Fulu and Fulu-to-Gloas against the published branch. The Amsterdam genesis header fix and Gloas recovery scenarios proceed alongside the Heze changes. |
| EL list selection and build attributes | Shared fixtures | Use the node backend's mempool selection. Return a JSON array, including `[]` when empty. Enforce the byte bound and nonempty entries. Give `forkchoiceUpdatedV5` its own Bogotá attribute validation and thread the target gas and list through the existing node build path while reusing payload identity. Confirm which old profiles still use snake_case. Verify `getPayloadV6` retrieves each candidate by the ID returned for its build inputs. Complete the V5 status response with the Bogotá validation PR. |
| CL validator publication | List method wire fixture. Live acceptance also needs the EL implementation. | Use the existing duty producer and add a BN candidate retrieval path backed by its actual EL. Sign using `ValidatorStore` and the inclusion-list domain. Test local signing, configured remote-signer behavior, activation, missed head, changed dependent root, slot deadline and failover without a conflicting signature. Observe signed publication and peer receipt with distinct beacon nodes. |
| CL incoming bid checks and list retrieval | Shared fixtures and existing `6e5317104`. Live retrieval uses a real signed-list producer. | In the live gossip validator, compare bid bits against the previous-slot timely view keyed by the bid parent's shuffling dependent root. Test that a bid missing a locally known required bit is ignored and the matching bid passes. Wire the store into `InclusionListsByIndices`. When the proposer or builder identifies a missing list from a candidate bid for the relevant branch, try recovery by committee index within the slot deadline. Test the wire path across two nodes, incorrect roots, retention, equivocation and bounded responses. |
| EL Bogotá validation and forkchoice response | Shared fixtures and EL build attributes | Add V6 to production dispatch, capability negotiation and the node backend. Share reconstruction, branch-relative state execution, ordered blob-hash validation and publication with V5. Return V2 for Bogotá `newPayloadV6` and `forkchoiceUpdatedV5`. Use payload post-state from its own branch and retain the original request list for `ACCEPTED` revalidation. Check the same block against two different lists and verify that they do not leak across calls or restart. |
| CL Bogotá proposal, envelope and fork choice | Bogotá wire fixtures. Integration acceptance needs the EL Bogotá methods. | Use the existing V5 attribute selection for Heze builds and `getPayloadV6` on the resulting payload ID. Keep the full previous-slot non-equivocating view under the proposal parent for build inputs and bid bits. On envelope import derive the previous-slot timely view for the block's shuffling dependent root and send it via V6. Carry execution validity and satisfaction separately through fork choice. Keep local historical verification and EL-assisted import independently testable. In the controlled omission case, the valid envelope remains valid and the next proposer chooses a parent without extending its execution payload. |
| Frame EL and CL | Pinned Frame fixture. Work can run in parallel with ordinary FOCIL. | Migrate EL semantics and the type-aware blob path. Add CL type-aware hash extraction and preserve original bytes for header hashing. Assert plain, sponsored and blob-bearing Frame cases against independent expected results. Test mixed blob ordering and wrong-digest rejection over the live Engine and CL-local history import. |
| Combined devnet, Dora and generator | Ordinary FOCIL and Frame pass, including the live V5 build, V6 payload retrieval and V6 validation calls. Dora can start with pinned event fixtures. | Run sponsored non-blob Frame with an isolated proposer mempool. Withhold list delivery for the control and assert that the selected proposer block omits the transaction. Deliver the signed list in the matching run and trace the only delivery path through list gossip, Engine build, bid and final receipt. Compare payer/sender balances and state roots on two pairs. Repeat after restart and verify Dora's displayed committee positions and transaction data. Keep blob Frame as a separate scenario. |

Two implementation pitfalls need tests on the real node backend.

1. `pkg/focil/il_satisfaction.go` contains reusable inclusion checks and
   historical tests. The legacy Engine wrapper passes a nil post-state.
   The optional Frame verifier treats a code-less VERIFY target as an
   exemption, whereas the current draft supports default-code accounts.
   Exercise an otherwise valid, sponsored Frame with a zero-balance
   sender, a default-code VERIFY, and omitted transactions with distinct
   balance, gas and nonce outcomes. Use the selected draft's execution
   and includability rules for expected results.
2. The legacy handler test asserts a Go response type with a
   `Transactions` member. Add a real JSON-RPC assertion to distinguish
   an object from an array. For V6, distinguish payload invalidity from
   malformed inclusion-list entries and from a valid payload omitting a
   required transaction. Verify same-hash/different-list results and
   the `ACCEPTED` to `VALID` revalidation path through forkchoice. Treat
   `engine_newInclusionListV1` as a historical extension with separate
   compatibility tests for any real caller.

[Satisfaction helper](../pkg/focil/il_satisfaction.go) · [Legacy list handler test](../pkg/engine/handler_test.go) · [Full implementation and devnet plan](lighthouse-hegota-implementation-plan.md)

## Audit reproduction

Run the branch comparison in the Lighthouse repository. The merge-tree
command reports conflicts without moving branches or changing the
checkout.

```bash
git merge-base eth2030 feat/fix-native-aa
git rev-list --left-right --count eth2030...feat/fix-native-aa
git diff --shortstat eth2030...feat/fix-native-aa
git merge-tree --write-tree --messages eth2030 feat/fix-native-aa
```

In the EL repository, run `git merge-base --is-ancestor` for each old
feature ref against `origin/master`. Use the current node backend and
the planned wire and live devnet checks to establish implementation
status. The trial merge and existing unit tests establish the migration
inputs. New cross-client behavior requires its own acceptance evidence.

At the audited revisions, all 68 commit IDs were checked against the
historical range. The existing EL focused tests passed with
`go -C pkg test ./engine ./focil ./node/backend -run 'InclusionList|ILSatisfaction|ForkchoiceUpdatedV4|PayloadV5' -count=1`.
Those tests exercise the existing contracts, including the legacy object
result. The new wire and devnet checks remain in the implementation plan.
