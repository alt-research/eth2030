# Devnet upstream adoption implementation plan

This plan implements the [feature-level dispositions](devnet-fork-upstream-audit.md#feature-level-disposition).
Adopt upstream code for behavior covered by a released or qualified mainline
implementation. Keep ETH2030 implementations for features that are still
experimental, using the upstream component and interface boundaries. Preserve
the tests that establish each behavior when the old implementation is retired.

The [Lighthouse and Hegotá plan](lighthouse-hegota-implementation-plan.md)
defines the original protocol contracts. The [CL gap audit](lighthouse-cl-gap-audit.md)
records the existing regressions and live evidence. This plan defines the PR
order, source changes and acceptance required to move the devnet profiles.

## Starting revisions

| Work | Starting point |
| --- | --- |
| Lighthouse | Published ETH2030 `eth2030` at [`1bf89a4fc`][cl-local]. Integrate upstream unstable [`1bcdcbfa0`][cl-main]. Keep `v8.3.3` as the Fulu control. Use [`f66bd5ac1`][cl-experiment] for Heze architecture comparisons. |
| ETH2030 EL | Start the Engine work from [`42e7ada9`][el-bogota] on `feat/bogota-inclusion-list-wire`. Start Frame execution from its descendant [`e628c770`][el-frame] on `feat/frame-8141`. Carry the Engine fixes forward into the Frame integration branch for the combined run. |
| Dora and Go client | [Dora `v1.25.0`][dora-release] for the generic control. [Dora `5fb1ef23`][dora-main] and its [Go client `v0.1.11`][panda-go] for migration. Keep [Dora `12a0198`][dora-local] and [its Go gitlink `79989d2`][go-local] as the Heze comparison pair. |
| Spamoor | Pin upstream [`f0f83695`][spamoor-main] for the ordinary workload and current Frame trials. Use the local [`4908a05`][spamoor-local] for the historical workloads and receipt fallback comparison. |
| ethereum-package | Keep [`c4a8e15d`][package-current] on the control during the first comparison. Evaluate [`d3014d6d`][package-candidate] separately after recording its effective generator, arguments and images. |

Refresh upstream references before opening implementation PRs. Record any
change to a candidate pin and rerun the affected contract tests. Use isolated
worktrees and separate PRs for each repository. Merge upstream into a staging
branch descended from the published Lighthouse branch so an accepted update can
fast-forward `eth2030`. Start Dora and Spamoor candidates directly from upstream
and add a focused compatibility change only where the audited contract requires it.

## Existing evidence and pending qualifications

The stable Lighthouse `v8.3.3` control ran two ETH2030 ELs at `89cfff90`
with package `c4a8e15d` and generator `5.3.5`. The prior runs recorded
Track 4 payload agreement and Fulu finalization. ETH2030's Heze run recovered
a signed list after dropped gossip and reopened the receiver. The pinned
Frame encoding and receipt vectors passed locally. The upstream CL, Dora,
Spamoor and newer package combinations in this plan still need their
profile-level gates. Earlier upstream Spamoor and Go codec runs stopped
during dependency downloads. [Control evidence][baseline] · [Heze evidence][heze-evidence]

## PR sequence

The IDs below are plan references. Each PR reports its source pins and results
for the gates it owns.

| ID | Repository and deliverable | Depends on |
| --- | --- | --- |
| 1 | ETH2030 devnet control, pinned inputs and verifiers | Existing two-pair fixtures |
| 2 | Lighthouse upstream sync and Heze architecture alignment | 1 for the live gate |
| 3 | ETH2030 EL deferred Bogotá V6 and FCU V5 validation | 1 for fixtures, 2 for the paired gate |
| 4 | ETH2030 profiles for upstream Dora and Spamoor ordinary workloads | 1 |
| 5 | Dora and Go codec migration for Heze | 1 and 2, plus a matched Heze schema |
| 6 | ETH2030 EL execution for the pinned Frame draft | 1 for vectors, 3 for the combined gate |
| 7 | Cross-client Frame generator, execution and CL historical replay | 2, 3 and 6. Use 5 when Dora is the explorer under test. |
| 8 | ethereum-package qualification and devnet profile promotion | 1 and 4 for ordinary Fulu, 2 for Gloas, 2/3/7 for combined Heze/Bogotá, and 5 for profiles switching Dora. |

Begin the Engine and Lighthouse changes in parallel. Run the ordinary
service swaps independently of Heze and Frame development. The Frame execution
PRs can use targeted fixtures while the deferred Engine changes are in review.

## 1. Freeze the control and make failures observable

**Repository:** ETH2030 devnet, under [`pkg/devnet/kurtosis`][devnet].
Use the existing [Track 4 wrapper][track4-run], [Track 4 verifier][track4-verify]
and [Heze isolated-mempool verifier][heze-verify]. Use the existing
[Track 4 profile][track4-profile] for Fulu and the [Heze profile][heze-profile]
for combined staging. Derive separate `gloas-genesis.yaml` and
`gloas-transition.yaml` candidates from [full-feature-eth2030][full-eth2030].
Keep their minimized inputs and verifier version alongside the results.

- Commit a sanitized fixture manifest with EL, BN and VC source pins and
  image identities, package and genesis-generator revisions, the fork
  schedule, genesis hashes and chain IDs. Pin the [Bogotá Engine draft][bogota]
  and [EIP-8141 draft][frame] for the wire inputs. Record generator settings used
  for Fulu, Gloas at genesis, scheduled Gloas and Heze with Bogotá. Use
  the recorded generator `6.2.8` as the starting candidate for Gloas at
  genesis. Recheck the Heze/Bogotá genesis before starting the combined run.
- Use [Buildoor `aa3bbaf5`][buildoor] as the upstream ePBS builder candidate.
  Pin the image and configure an enrolled builder with a funded test balance
  in the Gloas fixture. Verify its signed bid and envelope are accepted before
  counting it as the external builder reference. Keep the external-builder
  gate open if the candidate's API or fork is incompatible.
- Resolve the previously blocked dependency downloads for the exact upstream
  Spamoor and Go client pins. Record successful candidate builds separately
  from codec and live acceptance.
- Keep the control and candidate launch inputs comparable. For each swap,
  hold genesis, clients and other service versions fixed. Wait for applicable
  protocol transitions and capture the finalized Fulu checkpoint before
  checking the Gloas head. Record the active fork and the methods actually
  observed on the Engine wire.
- Extend the existing verifiers where assertions are missing. Retain real
  inputs, sanitized method responses, roots, peer counts, receipts and any
  error that prevented the scenario. Record Assertoor's actual terminal
  result when it is a gate, including failures and timeouts.
- Run two EL/BN/VC pairs within the existing Heze fixture budget of 2 GiB
  per EL or BN and 1 GiB per VC. Give additional services a measured budget
  when a scenario needs them. Reuse one bounded enclave at a time and
  remove it after evidence collection.

**Gate:** Both ELs and Beacon APIs agree on finalized execution from the
Fulu control. Save a Track 4 PQ transaction hash, execution block and state
root before restart. Restart one EL with its database intact, resolve its
current RPC binding and read the same receipt and historical state. Extend
the verifier for this readback because its normal entry point submits new
transactions. A launch that omits a
required workload or fails to capture its outputs is incomplete.

## 2. Integrate Lighthouse upstream and carry the required Heze behavior

**Repository:** Lighthouse. Start with the upstream merge and a source diff
against `1bcdcbfa0`. Review the architecture changes separately from the
merge. The [feature-level dispositions][ownership] identify overlapping
mainline code and ETH2030 patches to keep.

1. Use upstream Gloas proposal, PTC, envelope, custody, database and
   light-client paths. Retain the old cases as regressions and exercise
   the fork-boundary header fix. Close Gloas at genesis and scheduled
   transition runs with the signed external builder bid, payment, withheld
   or late envelope, PTC decision, bad BAL rejection and data-column cases.
2. Organize the ETH2030 Heze VC as a validator service that consumes the
   duty and dependent root, asks the BN for the candidate, and signs through
   the validator store. Cover restart, skipped slots, competing heads and
   retry to a second BN with the same signed message. Use the upstream
   types, duty representations and list store.
3. In `block_production/gloas.rs`, keep one parent-scoped list snapshot for
   FCU payload attributes and the self-build bid bits. In the envelope
   notifier, select the import-time timely list view. Keep V6/V5 transport
   and decoding in `execution_layer` and the deferred verdict on the
   execution-bearing ancestor in fork choice.
4. Keep Req/Resp codecs and serving in the P2P RPC layer. Keep peer and
   request lifecycle in the network service and signature, root and
   timeliness decisions in the beacon chain. Exercise both the gossiped
   builder bid and the embedded self-build bid triggers. Add partial-response
   and peer failover coverage. Keep the proposal-root BN view until PR 5
   demonstrates equivalent readback with the endpoint off.

Run the existing merge regressions by name before the broader build.
Keep the inclusion-list test module wired into the selected target when
merging. Confirm the names are present and that the Fulu-to-Gloas test
actually runs. The Req/Resp and HTTP tests are compiled in release mode.

```sh
cargo test --release --locked -j 2 -p beacon_chain --test beacon_chain_tests heze_
cargo test --release --locked -j 2 -p network --lib heze_reqresp_imports_and_serves_previous_slot_inclusion_list
cargo test --release --locked -j 2 -p http_api --test bn_http_api_tests heze_inclusion_view_is_bound_to_the_proposal_parent
cargo test --locked -j 2 -p beacon_chain --lib heze_historical_envelope_binds_frame_blob_commitments_in_order
FORK_NAME=fulu cargo test --release --locked -j 2 -p beacon_chain --features spec-minimal,fork_from_env --test beacon_chain_spec_minimal_tests light_client_finality_update_across_next_fork
cargo check --locked -j 2 -p beacon_chain -p execution_layer -p network -p validator_services
```

**Gate:** Capture both correct and incorrect fork-choice verdicts, including
the executed ancestor under an empty beacon head and reload. Drop the
receiver's list gossip, recover the exact signed list through Req/Resp,
retain its late status, and restart the receiver. A capped live run must
produce matching finalized block and receipt evidence on both pairs. The
real deferred EL case is gated jointly with PR 3.

## 3. Retain Bogotá inputs in the native EL

**Repository:** ETH2030 EL. Work in
[`pkg/engine/bogota.go`][el-engine] and
[`pkg/node/backend/engine_bogota.go`][el-backend],
using the native Engine backend and its fork-choice/recovery path.

- Add a native deferred staging path for an off-canonical payload with
  known well-formed ancestry when full execution is deferred. Apply the
  mandatory wire, hash and fork checks before recording `ACCEPTED`. Keep
  `SYNCING` for missing ancestry or validation data and enforce immediate
  validation for a canonical extension whose requisite data are available.
  The [inherited Engine status rules][paris] define these cases.
- Stage the exact payload and its list in the native backend. Wire FCU's
  execution head lookup to complete validation using the same native state
  transition and the existing post-state inclusion check. Persist the input
  before replying `ACCEPTED` and retain the result through the existing
  database owner. Bind both to the immutable execution block and the view
  the authenticated driving CL actually supplied.
- A retry with identical bytes is idempotent. For the single driving CL in
  this profile, serialize a later successful V6 for the same payload as a
  replacement of its active view. Validate that choice against a compatible
  reference EL before qualifying the cross-client profile. FCU evaluates a
  coherent retained view and cannot publish an older result after replacement.
- Represent an explicit empty list independently from missing or unresolved
  input. For a post-fork head restored or imported through P2P without its
  CL view, rebuild the result by replaying that head's applicable V6 input
  from the CL. Exercise the wire outcome before resubmission against the
  pinned contract. A cache miss never establishes satisfaction. Keep
  execution validity independent and return a satisfaction Boolean exactly
  for `VALID`. Bound retained branch state and check its lifetime at reorg.
- Keep `newPayloadV5` and `forkchoiceUpdatedV4` on the pre-Bogotá
  Amsterdam path. Activate `newPayloadV6` and `forkchoiceUpdatedV5`
  for Bogotá. Verify the advertised methods and payload retrieval through
  `engine_getPayloadV6`, typed response fields and BAL, method and
  parameter order, null satisfaction, tolerated malformed list items, and
  wrong-fork method errors.

**Gate:** Through authenticated HTTP, send the same execution payload with
a satisfying and an unsatisfying list. Obtain native `ACCEPTED` for an
eligible off-canonical payload and bring it to a real FCU `VALID` verdict.
Restart between staging and FCU, exercise two views and a competing head,
and verify the same result after database reopen. On the CL check the next
proposer's execution parent and the reloaded fork-choice decision. A missing
parent run also has to demonstrate `SYNCING` and later recovery. Record
independent-EL `ACCEPTED` interoperability as a separate gate until a
candidate proves the same deferred status path.

## 4. Replace ordinary workload and explorer images

**Repository:** ETH2030 devnet. Use the pinned upstream Spamoor build and
Dora release in staged candidates derived from [full-feature][full-profile]
and [full-feature-eth2030][full-eth2030]. Run the replacements separately
against the same Fulu/Gloas control. Move
the old EIP-7701 raw and UserOp selections to a historical fixture before
removing them from present-day profile inputs.

**Gate:** Submit EOA, token, blob and EIP-7702 traffic where the fork enables
them. Prove inclusion and canonical receipt readback. For Spamoor funding,
deployment and receipts, use two EL endpoints where only the second can
serve the receipt at first. Verify wallet progression and a bounded retry.
Where upstream misses required receipt fallback semantics, isolate the fix
behind the existing `ClientPool` and `TxPool` interfaces. For Dora, compare
slots, envelopes and canonical execution data with the Beacon and EL APIs.
Record the staged results here. PR 8 promotes each passing profile.

## 5. Migrate Heze Dora and its Go codec

**Repositories:** Dora, go-eth2-client if compatibility is required, and
Lighthouse for the endpoint retirement.

- Begin from upstream Dora's service event, `inclusionlistcache`,
  `blockmetadata.go` and slot-metadata readback. Use SQLite plus Pebble in
  the devnet profile and preserve both across restart. Run the existing
  `TestUnfinalizedMetadataCrashCopy` regression.
- Replay actual compatible CL BeaconBlock and BeaconState SSZ and their
  JSON representations. The upstream Go dependency expects the EIP-8015
  Heze layout. Keep the compatibility pin for the old CL schema, or match
  the CL and Dora to a new schema in one documented profile revision. First
  establish the old-profile replay with the known compatibility pin. Qualify
  the upstream-only pair on its matched schema in a separate fixture. Put any
  retained compatibility at the fork-specific decoder and keep it out of
  the indexer and UI.
- Build fixtures for two proposal roots in one slot, observed late and
  equivocated lists, subscriber disconnect, Req/Resp recovery, BN pruning
  and Dora restart. Compare signed bytes and dependent roots as well as
  the display. Check state-root and epoch-statistics readbacks through the
  normal consumer. Use a fresh Dora database to check what can be rebuilt.

**Gate:** Retire the Go fork when the released decoder passes current and
historical fixtures required by each active profile. Retire the Dora
source-JSON fallback when stored blocks reopen correctly without it.
Retire `353288d2`'s adapter and the Lighthouse retained-view endpoint only
after replay with the endpoint disabled preserves the required proposal
views through disconnect, recovery, pruning and restart. Keep the prior
release and databases available for rollback during migration.

## 6. Implement the pinned Frame state transition

**Repository:** ETH2030 EL. Start from `e628c770` and split state
transition and admission/block-path changes so each can be reviewed.

- Add the canonical Frame route at `applyTransactionInternal` in
  `pkg/core/execution/processor.go` where it currently rejects the
  seven-field format. Use the call and journal boundaries in
  `pkg/core/eips/frame_execution.go` as architectural reference for the
  pinned state transition. Reuse `pkg/core/types/frame_8141*.go` for canonical
  bytes, signatures, budget and receipt encoding. Implement frame execution,
  scope and approval handling, escrow, sponsor settlement, state-gas
  ownership and refills, and atomic rollback at the execution/state layer.
  Keep the historical executor on the original fork-specific route.
- Wire fork activation and public admission in the native
  [`pkg/txpool` path][el-pool]. Reuse the same execution route for building,
  Engine validation and history replay. Add blob sidecar and versioned
  hash checks and keep the typed receipt intact through storage and RPC.
- Run the existing `TestFrame8141` vectors and add state-root, balances,
  gas dimensions and ordered frame outcomes against independent fixtures.
  Include sponsor insufficiency, wrong authorization, maximum scalar
  nonce, and partial failure/atomic rollback. Compare a legitimate
  reverted transaction with a consensus-invalid one. Verify the same
  receipt and storage after restart.

**Gate:** A native node admits and executes the pinned transaction format
with exact fixture outputs across construction, Engine validation and
replay. Invalid Frame, carrier or sidecar input must fail at the applicable
boundary. Preserve the Fulu Track 4 regression on the same candidate.

## 7. Qualify Frame workloads and historical verification

**Repositories:** ETH2030 devnet, Lighthouse, and Dora/Spamoor only where a
remaining compatibility difference warrants a focused patch.

- Use upstream Spamoor `frametx --envelope=base` against the pinned draft.
  Compare the transaction bytes, signing hash and split-gas receipt with
  the frozen vectors. Exercise deterministic delegated EIP-7702 sender,
  sponsor and blob-bearing cases. Require real inclusion and matching
  receipts and payer debits. Treat verification warnings as a failed gate.
- Keep self-build and FOCIL sponsorship separate. The sponsored FOCIL gate
  stays blocked until the paired implementations pin a rule admitting the
  chosen non-blob Frame shape. The [VOPS proposal][vops] is a reference for
  that decision. After pinning it, compare the listed transaction bytes with
  the proposal and include a policy-permitted omission. Keep eligibility in
  the EL.
- Import a block with ordered Frame and EIP-4844 blob hashes on both the
  Engine-assisted and CL-local paths. Test omitted, reversed and erroneous
  commitments separately from unavailable data. Verify historical backfill,
  optimistic finalized sync and late join on a fresh Lighthouse database.
  For independent execution parity, qualify [Geth Bogotá devnet 2
  `f71f4be8`][geth-bogota] against the pinned seven-field draft first.
  Compare receipt bytes, balances and state roots only if it runs the
  selected contract. Independent Heze CL wire parity remains open until
  a matching client is pinned. Apply the independent full/snap gates in
  the [Track 3 acceptance plan](track3-remaining-acceptance.md) where applicable.
- Read the Dora transaction, payer and gas dimensions from its normal
  indexed endpoint. Compare them with canonical receipt bytes and the
  execution client's state. Track any remaining dependency of the Heze
  explorer instead of counting the Frame path as historical parity.

**Gate:** Record separate codec, native and independent EL state, Lighthouse
historical sync, independent CL and Dora results. The finalized transaction,
sponsor accounting, restart and late-join readbacks are required to replace
an active Frame workload. Existing draft fixtures remain available for replay.

## 8. Qualify the package and promote one profile at a time

**Repository:** ETH2030 devnet. Use the existing `KURTOSIS_PACKAGE` override
to test the upstream package separately with the same client images. Keep
native launch arguments, fork settings and workload assertions in ETH2030.
Avoid image selections that silently route the combined profile through
[Frame-only genesis preparation][package-genesis]. Inspect generated CL
fork settings and EL Bogotá activation before launch and confirm both on a
running chain.

Promote `track4.yaml`, the staged ordinary variants, the two Gloas
fixtures and the combined Heze fixture separately. For each profile, record
the prior and candidate manifest plus the date and acceptance results. Promote only the profile whose gates passed. Retire the
replaced source overlays in the owning repository and preserve their
regression tests. Roll back by restoring the previous supported pins and
the corresponding data when storage formats differ. Record which tests
still lack a compatible independent client.

Quick Slots, optional execution proofs and Lean/PQ validator experiments
follow their separate [CL roadmap lanes](cl-iteration-roadmap.md). They have
their own specifications and devnet fixtures.

[cl-local]: https://github.com/jiayaoqijia/lighthouse/commit/1bf89a4fc933d58828c39bcda3fe6eb481d680f5
[cl-main]: https://github.com/sigp/lighthouse/commit/1bcdcbfa0e4b97f51b60212591391e956fd8a4da
[cl-experiment]: https://github.com/sigp/lighthouse/commit/f66bd5ac1051d443f11b5c701410203f4e3cf4b3
[el-bogota]: https://github.com/fyInALT/eth2030/commit/42e7ada99372650111b63b38bd2606c3eab19ece
[el-frame]: https://github.com/fyInALT/eth2030/commit/e628c770df98ba9ad269de74344afcdf4e6a92a3
[dora-release]: https://github.com/ethpandaops/dora/releases/tag/v1.25.0
[dora-main]: https://github.com/ethpandaops/dora/commit/5fb1ef23668c13d0a23b481b2dee43993590cba2
[panda-go]: https://github.com/ethpandaops/go-eth2-client/tree/v0.1.11
[dora-local]: https://github.com/jiayaoqijia/dora/commit/12a0198f237c8bf1973256ae5c2498d7cef90587
[go-local]: https://github.com/fyInALT/go-eth2-client/commit/79989d2591dc7a964990097a1e048db49d4d3890
[spamoor-main]: https://github.com/ethpandaops/spamoor/commit/f0f83695b24fe81ba25112e4415c2d75ef6d760c
[spamoor-local]: https://github.com/jiayaoqijia/spamoor/commit/4908a05d77d3836f61524aeda6c4ee79f8c36097
[package-current]: https://github.com/ethpandaops/ethereum-package/commit/c4a8e15d40f0965f3728f776d72f1954440ad96b
[package-candidate]: https://github.com/ethpandaops/ethereum-package/commit/d3014d6d23d683362025b9e23a0d69de49e77cca
[devnet]: ../pkg/devnet/kurtosis/README.md
[track4-run]: ../pkg/devnet/kurtosis/scripts/run-track4.sh
[track4-verify]: ../pkg/devnet/kurtosis/scripts/verify-track4.sh
[heze-verify]: https://github.com/fyInALT/eth2030/blob/e628c770df98ba9ad269de74344afcdf4e6a92a3/pkg/devnet/kurtosis/scripts/verify-heze-focil.py
[bogota]: https://github.com/ethereum/execution-apis/blob/981aa66d79364be8322a9c358aaf0a571cca0873/src/engine/bogota.md
[frame]: https://github.com/ethereum/EIPs/blob/dc5d5fa688b7fe87b95a907fcc128b4fde2f4d2f/EIPS/eip-8141.md
[ownership]: devnet-fork-upstream-audit.md#feature-level-disposition
[el-engine]: https://github.com/fyInALT/eth2030/blob/42e7ada99372650111b63b38bd2606c3eab19ece/pkg/engine/bogota.go
[el-backend]: https://github.com/fyInALT/eth2030/blob/42e7ada99372650111b63b38bd2606c3eab19ece/pkg/node/backend/engine_bogota.go
[el-pool]: https://github.com/fyInALT/eth2030/blob/e628c770df98ba9ad269de74344afcdf4e6a92a3/pkg/txpool/txpool.go#L541-L543
[package-genesis]: https://github.com/ethpandaops/ethereum-package/blob/d3014d6d23d683362025b9e23a0d69de49e77cca/src/participant_network.star#L95-L111
[baseline]: cl-iteration-roadmap.md#lighthouse-baseline-verification
[heze-evidence]: ../STATUS.md#heze-consensus-validation-2026-11-16
[track4-profile]: ../pkg/devnet/kurtosis/configs/track4.yaml
[heze-profile]: https://github.com/fyInALT/eth2030/blob/e628c770df98ba9ad269de74344afcdf4e6a92a3/pkg/devnet/kurtosis/configs/heze-focil.yaml
[full-profile]: ../pkg/devnet/kurtosis/configs/full-feature.yaml
[full-eth2030]: ../pkg/devnet/kurtosis/configs/full-feature-eth2030.yaml
[buildoor]: https://github.com/ethpandaops/buildoor/commit/aa3bbaf543dd2072e2c6026f60af122727810355
[paris]: https://github.com/ethereum/execution-apis/blob/981aa66d79364be8322a9c358aaf0a571cca0873/src/engine/paris.md
[vops]: https://eips.ethereum.org/EIPS/eip-8369
[geth-bogota]: https://github.com/ethereum/go-ethereum/commit/f71f4be8027bc0828148ae03424231092ed7ab66
