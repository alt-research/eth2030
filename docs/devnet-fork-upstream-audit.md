# Devnet fork and upstream adoption audit

Audited on 2026-11-17. Use the upstream implementation where a published
release or the maintained mainline supplies the required behavior for the
same contract. Replace the overlapping ETH2030 code on the active branch
and retain its regression cases. The [legacy Lighthouse plan][cl-legacy-plan]
maps the separate March branch history. Where the available upstream implementation
is experimental, use its module, interface and ownership boundaries to
organize the ETH2030 implementation. Preserve historical source pins for
replay.

The [devnet inventory](devnet-third-party-inventory.md) maps the worktrees,
profiles and images. This audit compares the code those profiles need with
the published release, default branch and relevant devnet branches of each
dependency. Git commits pin the comparisons. A migration becomes eligible
when the listed workload or wire checks pass on that pin. For a mainline
feature that has not reached a release, qualify its source commit. Evaluate
transport, runtime wiring, persistence and consumer behavior independently.
A mature wire codec can be adopted while its experimental runtime caller
still needs ETH2030 code.

## Executed consensus codec comparison (2026-11-18)

The [retained replay](../tools/consensus-replay/README.md) updates the codec
qualification beyond the earlier source comparison. The exact Go dependency is
ethpandaops `v0.1.11`, and the actual CL is Lighthouse `a05a56de`.

- The March maintained Go codec failed a live **Gloas** state request with
  `incorrect offset`. Its state still places `LatestExecutionPayloadBid` at
  the former payload-header position, omits the newer execution requests root
  in the bid and has no PTC window. Retaining the old Heze Eth1 fields alone
  does not make that codec compatible with the current CL.
- The published Heze decoder failed the real current Heze state at
  `HistoricalRoots`: first offset `3134845`, expected `3134753`. The difference
  is exactly 92 fixed bytes for the retained `eth1_data`, `eth1_data_votes`
  offset, `eth1_deposit_index` and `deposit_requests_start_index`.
- The compatibility candidate uses the published **Gloas** state's current
  field order and progressive SSZ tags with the published **Heze** bid. The
  actual finalized slot-192 SSZ roundtrips byte-for-byte and matches the
  original Beacon header state root, all 256 public keys and balances, and
  the 96-slot PTC window. Changing state, root, preset, epoch JSON or the
  execution requests root fails replay. Removing the root comparison is
  detected by the changed-state and unrelated-header regressions.
- This supports an upstream-based, fork-specific compatibility decoder rather
  than carrying forward the March state structs. It does not qualify a Dora
  deployment or the indexer's epoch consumer. Keep the fork/profile pin until
  those paths process the retained SSZ and match epoch statistics through a
  service restart. The published codec is tried first and remains the preferred
  path for its own released layout; no state-root agreement is inferred from
  JSON display.

The compressed real state, runtime revisions, Beacon header, spec response and
all expected epoch identities/balances are committed with the replay tool.
Its CI job runs real-fixture, mutation-derived negative and race regressions.

## Decisions and revisions

| Component | ETH2030 revision | Upstream examined | Decision |
| --- | --- | --- | --- |
| Lighthouse | [`1bf89a4fc`][local-cl] | [stable `v8.3.3`][cl-stable], [`unstable` at `1bcdcbfa0`][cl-main], [`heze-devnet-2` at `f66bd5ac1`][cl-devnet] | Use stable for the established Fulu control. Qualify `1bcdcbfa0` for upstream Gloas coverage. Continue the small ETH2030 Heze branch until the local recovery, verdict and Frame checks are available on a compatible upstream pin. |
| Dora | [`6f702f2`][dora-legacy] and [`12a0198`][local-dora] | [`v1.25.0` at `8ad536e7`][dora-release], [`master` at `5fb1ef23`][dora-main], [`bbusa/heze-devnet-3` at `25aee6e7`][dora-heze] | Trial `v1.25.0` for the generic Fulu/Gloas explorer and require the profile's Beacon API checks to pass. Use master as the Heze and Frame migration candidate. Retain the proposal-root adapter until event ingestion and restart give equivalent results. |
| go-eth2-client | Legacy `21c05ba` and Dora Heze gitlink [`79989d2`][local-go] | [Attestantio `master` at `6d6c8310`][go-canonical], [ethpandaops `v0.1.11` at `ceca3c2d`][go-panda-tag], [ethpandaops `master` at `78aca033`][go-panda] | Use the client already selected by upstream Dora. The current ETH2030 Heze BeaconState schema still needs a matched codec. Keep that compatibility pin for existing fixtures and retire it with the joint CL/Dora migration. |
| Spamoor | [`4908a05`][local-spamoor] | [`v1.2.4` at `753b08c`][spamoor-release], [`master` at `f0f83695`][spamoor-main], [`bbusa/heze-devnet-3` at `08d1959d`][spamoor-heze] | Trial a build of `f0f83695` for ordinary workloads first. Use master for the Bogotá Frame generator trial. Keep legacy raw-AA workloads as archived fixtures. Carry a focused receipt fallback change only if the paired-client test demonstrates a need. |
| ethereum-package | [`c4a8e15d`][local-package] | [`main` at `d3014d6d`][package-main] | Continue with a read-only upstream pin. Qualify `d3014d6d` in a separate pass with explicit service images and a genesis check for every profile. |

The author-maintained EIP-8141 reference implementation, revm and Foundry
checkouts serve as research fixtures at the revisions in the inventory.
Their gitlinks and captured vectors provide the reference for that draft.
Assertoor, Blockscout, Blobscan and the remaining optional services already
use upstream images through ethereum-package.

## Feature-level disposition

For each migration, start from the selected upstream source. Apply only the
ETH2030 behavior still needed for the profile, together with its regression
tests. The tables identify the local code to retire and the boundaries to
preserve for experimental features.

### Lighthouse

| Feature | Upstream status | ETH2030 disposition and placement |
| --- | --- | --- |
| Fulu, Gloas ePBS and database lifecycle | Fulu is released in `v8.3.3`. Gloas bid handling, envelope import, PTC, fork choice, PeerDAS and database recovery are maintained on `unstable`. | Adopt those upstream implementations and retire overlapping March code. Keep the previous delayed-envelope, custody, restart and fork-boundary cases. Qualify the current upstream Gloas pin with the builder, PTC and BAL gates in the CL audit. |
| Heze list data and Engine transport | Mainline provides list types, committee and dependent-root storage, gossip validation and a V6 JSON-RPC implementation. Mainline envelope import still builds a Gloas request. | Adopt upstream types, storage and the shared JSON-RPC machinery. Keep the pinned Bogotá payload shape and the V6/FCU V5 rule that satisfaction carries a Boolean exactly when the execution status is `VALID`. Place import-time Heze request selection and the timely list view in the envelope notifier. Keep the FCU/self-bid proposer snapshot in `block_production`, wire dispatch and decoding in `execution_layer`, and eligibility in the EL. [Mainline notifier][cl-main-notifier] · [V6 transport][cl-main-transport] |
| VC production and BN duties | `heze-devnet-2` has slot-clock-aware VC scheduling, a duties lookup keyed by slot and dependent root, signing and publication. | Keep ETH2030 duties and publication implementation (`4e1167314`, `4985f0110`). Organize it as a validator service consuming duties, with BN APIs sourcing candidates and the validator store owning signing. Test the same signed message across retries, remote signing and redundant BNs. [Experimental VC][cl-devnet-vc] |
| Proposer view and bid bits | The experimental producer evaluates prior-slot bits in block production and obtains list transactions independently when preparing the EL payload. The two reads can observe different store states. | Keep the locally tested snapshot (`6e5317104`). Derive the FCU transactions and self-bid bits from the same parent-scoped read and carry it through asynchronous build. Keep gossip's timely-only check in bid validation (`dfd8281d2`) and proposer selection's wider view in block production. [Experimental producer][cl-devnet-production] · [Local producer][cl-local-production] |
| Immediate/deferred Bogotá verdict | The devnet branch applies the immediate V6 verdict during envelope import. Its FCU path uses execution validity and discards the inclusion field. | Keep `865a680af` and `d7a8eb5e6` semantics. Place the verdict handoff in envelope import and FCU, and persist the independent inclusion result in fork choice on the executed ancestor. Keep `dd48a965e` and the empty-head/reload regression. The managed EL must retain and revalidate the submitted list. [Experimental notifier][cl-devnet-notifier] · [FCU path][cl-devnet-fcu] |
| Heze list Req/Resp | The experiment implements codec and inbound serving. Its network service has no consumer for an outbound response. | Keep the ETH2030 requester and validated recovery (`d43b36a0c`, `7f87e92c5`). Follow the upstream network layering. The RPC module owns protocol shape and codecs, the network service owns request lifetime and peer association, and the beacon chain authenticates and inserts lists under the original slot and root. Trigger recovery for both gossiped and embedded bids. Preserve the late status. Verify whether the local delayed-response boxing (`a11bb89b`) is still required by the integrated size/lint check. [Experimental dispatch][cl-devnet-network] |
| Frame envelope verification | The upstream devnet CL decodes via Alloy and extracts EIP-4844 hashes. The pinned Frame type is absent from that path. | Keep the narrow fork-aware Frame parser in `execution_layer/src/versioned_hashes.rs` (`e01b8737a`). Keep historical replay and KZG order coverage (`1bf89a4fc`). Route block verification and historical import through the same hash extraction. Use the original transaction bytes for execution hashing and the EL for execution. [Experimental extraction][cl-devnet-hashes] · [Local extraction][local-frame-hashes] |
| Explorer-facing view and light client | The experimental CL supplies list events. The current Dora compatibility endpoint has proposal-root semantics. Both the audited mainline and experimental light-client paths leave Heze header creation open. | Keep `5d4500270` as a compatibility query until Dora's event-led path proves parity. Adopt the merged Gloas light-client fix `e723ca55f`. Put any required Heze extension in the existing fork-specific header, bootstrap and update modules, using the selected schema and proof cases. [Local retained view][local-retained] · [Gloas fix][cl-light-pr] |

### Dora

| Feature | Upstream status | ETH2030 disposition and placement |
| --- | --- | --- |
| Gloas/Heze explorer and ordinary FOCIL | The official release has slot, bid, envelope and FOCIL indexing. Mainline has the later slot-metadata restore path and per-block evaluation. | Start from upstream. Retire the overlapping March indexer, route, transaction-preview and committee UI patches. Use upstream SSE ingestion, `inclusionlistcache`, unfinalized metadata and `SlotMeta` with the stock UI/API. Preserve the ETH2030 comparison fixtures. [FOCIL indexing][dora-index] · [Persistence][dora-storage] |
| Exact proposal read and recovery | Upstream maps a hash argument to a slot in its public API. Its event-led model can preserve distinct block outcomes once it has received the observations. | Retain only the unresolved proposal-view behavior from `353288d2`. Feed BN retained-view recovery into the upstream indexer storage model if needed. Keep the proposal hash, dependent root and observation provenance together. Compare two candidate blocks of the same slot and show missing history explicitly after pruning. Retire the adapter and CL endpoint after the replay gate. [Upstream slot API][dora-api] |
| JSON block fallback and codec pin | Current upstream Dora uses the published Go module. ETH2030 `9a9e8790` preserves source JSON for history when the old SSZ path fails, and `12a0198` pins the bid decoder. | Qualify stock decoding against current and historical saved blocks and their database reload. Retire `9a9e8790` once that history survives without its JSON fallback, and remove the custom gitlink and mapping when the released codec passes every active profile. [Local JSON fallback][dora-json-fallback] · [Dora module][dora-module] |
| Frame display and FOCIL eligibility | Mainline #920 supports Bogotá devnet 2 receipts, payer and gas dimensions. Its current inclusion evaluator places Frame outside enforcement. | Replace the legacy Type 6 renderer from `6f702f2` for compatible receipts. Keep Type 5 history archived. If the pinned Frame ABI or future FOCIL eligibility differs, implement the compatibility at the transaction/receipt decoder and list evaluation layers. Leave templates and database readback on the upstream paths. [Frame indexer][dora-frames] · [Inclusion outcomes][dora-outcomes] |

### go-eth2-client

| Feature | Upstream status | ETH2030 disposition and placement |
| --- | --- | --- |
| Gloas and current Beacon APIs | Attestantio mainline has Gloas. Upstream Dora's selected ethpandaops release also has the Heze block/state, bid, envelope and event types. | Adopt Dora's published module dependency for matching profiles and retire duplicated Gloas and JSON types. The current Lighthouse serializes `message`, so keep `79989d2` alias handling only for captured historical inputs that require it. [Dora module][dora-module] · [Bid decoder][go-panda-bid] |
| Earlier Heze draft on ETH2030 | The published ethpandaops Heze codec uses the EIP-8015 layout. The earlier dynamic-SSZ devnet branch organizes codecs per fork and predates that upgrade. | Keep a codec for the old ETH2030 schema until the CL migrates. Organize it using upstream's fork-specific SSZ types, versioned HTTP decoding and event wrappers. Select it for the pinned profile and keep any JSON compatibility in the decoder. Preserve raw fixtures for genesis, block, envelope and state. Retire this code when the released Go codec loads the same state and block and Dora computes matching epoch statistics. [Experimental codec][go-experimental] · [Released state][go-panda-state] · [Local state][local-heze-state] |

### Spamoor

| Feature | Upstream status | ETH2030 disposition and placement |
| --- | --- | --- |
| EOA, token, blob and SetCode workload | The published release and mainline ship the ordinary workload catalog. | Adopt upstream scenarios in the devnet wrapper. Remove the fork image for ordinary workload after inclusion and receipt qualification. Keep EIP-7701 raw and UserOp fixtures from `d265297`, `47ef700` and `cf6eaf9` in the archival lane. |
| Modern Frame | A maintained mainline implementation ships scenario builders, signer, per-shape assertions and typed transaction/receipt codecs. The reviewed release and the older Heze branch predate the Bogotá max-fee change. | Adopt the mainline codec and scenario as the implementation for their covered wire contract. Retire the legacy Frame builder and approver changes from `f03643f` through `4d4b439` once parity includes delegated EIP-7702 senders and sponsored payment. Use the upstream delegated-sender and sponsored recipes or equivalent deterministic fixtures. Assert authorization, approvals, payer debit and receipt. Keep only differences proven by the pinned fixtures, such as the scalar-nonce boundary. Put local policy and assertions in the scenario/test wrapper, and any proven codec gap in `txtypes`. [Frame scenario][spamoor-frame-scenario] · [Mainline Bogotá change][spamoor-frame-pr] · [Delegated and sponsored recipes][spamoor-sponsored] |
| Funding, post-confirmation reads and reorg | Mainline adds confirmation-aware proxy clients and uses the existing client pool, wallet pool and deployer. The experimental Heze branch adds reorg rebroadcast and height waits. | Adopt mainline wallet and deployer behavior first. Retire overlapping waits from `5b6828a`, `64d191a` and `0282dd2` after paired-client checks. Where needs remain, implement retry/reorg handling around the upstream client selection and `TxPool` lifecycle. [Mainline proxy][spamoor-proxy] · [Experimental rebroadcast][spamoor-heze] |
| Receipt retry across peers | Mainline samples clients with finite retries. The local `4908a05` walks the eligible client indexes. | Keep deterministic fallback only if the paired-client test requires it. Implement any retained policy through upstream `ClientPool` selection and `TxPool` receipt lookup, with one per-request timeout and a bounded retry budget. [Mainline lookup][spamoor-receipt-main] · [Local patch][spamoor-receipt-local] |

### ethereum-package and supporting repositories

Adopt the supported package launcher and generator by pinned revision. Keep
native EL wiring, fork schedule, image identities, memory budgets and test
assertions in the ETH2030 profile and verifier. The gas-limit/Prysm experiment changes the
schedule default and selects REST automatically in Heze. Use mainline's
`gas_limit_schedule` and explicit
`vc_extra_params: ["--enable-beacon-rest-api"]` in the profiles that need
them. Verify the produced block's gas limit and Prysm's duty publication
through REST. [Mainline schedule][package-gas] · [Mainline Prysm launcher][package-prysm] Preserve the combined Heze/Bogotá activation test, since
mainline's Frame-only mode changes the generated CL schedule.
[Experimental package change][package-heze] · [Mainline genesis path][package-frame]

Use upstream images for Assertoor, Blockscout and Blobscan. The Assertoor
runner's terminal-result handling belongs in ETH2030's test harness, and
the explorer comparison belongs in its verifier. Keep the external Frame
reference stack as pinned reference code. Its experimental draft can
inform state-transition and fixture organization for ETH2030's own EL
implementation.

## Lighthouse

### What upstream now covers

- `v8.3.3` and upstream unstable `1a704e58f` already ran against two ETH2030
  ELs at Fulu. Use those results for the Fulu control. The upstream Gloas
  branch owns ePBS bids and envelopes, PTC, fork choice and storage. The
  observed Gloas transition in ETH2030 used a pinned integration build.
  The upstream `e723ca55f` fix corrects light-client headers across fork
  boundaries. Its test skips Heze. The audited `1bcdcbfa0` head adds
  builder config fields and metric cleanup after that fix. [Baseline runs][cl-baseline] · [Lighthouse #10401][cl-light-pr]
- On `unstable`, the V6 RPC codec and Heze inclusion-list types and store
  are present. The [active envelope notifier][cl-main-notifier] still
  constructs a Gloas request, so that mainline path does not send the Heze
  inclusion-list view to the EL.
- The upstream devnet branch constructs [`NewPayloadRequest::Heze`][cl-devnet-notifier],
  has a [slot-aware inclusion-list VC service][cl-devnet-vc], checks the
  timely gossip bid bits and carries the immediate V6 verdict into
  envelope import. It has an integration test for V6 and the list argument.
  This branch supplies candidate code for future forward ports.
  [Bid verification][cl-devnet-bid] · [V6 test][cl-devnet-verdict-test]

### What the ETH2030 Heze branch still supplies

| Required behavior | Gap on the examined upstream branches |
| --- | --- |
| Deferred Bogotá verdict | The devnet branch FCU handler promotes the execution head by hash on `PayloadStatus::Valid { .. }` and discards the verdict field. ETH2030 binds `VALID,false` to the execution-bearing ancestor under an empty beacon head and tests persistence in fork choice. Real EL retention and revalidation remain a separate dependency. [Upstream FCU][cl-devnet-fcu] · [ETH2030 regression][local-deferred] |
| Missed-gossip recovery | The devnet branch serves `InclusionListsByIndices`; its network service treats the response as unsolicited. ETH2030 requests lists for missing bid bits, including when a self-build bid is embedded in the block. A two-pair run verified recovery and that the recovered list stayed late. [Upstream dispatch][cl-devnet-network] · [ETH2030 trigger][local-recovery] · [Run record][heze-status] |
| Proposal-root retained view | The ETH2030 BN endpoint resolves the previous-slot committee using the proposal's parent-dependent root and serves the retained signed lists to the Dora adapter. The audited upstream path exposes the list event and peer interface. Dora event-led ingestion can remove this endpoint dependency if it passes the parity run below. [ETH2030 API][local-retained] |
| Frame blob checks in local verification | The upstream devnet branch decodes `TxEnvelope` and collects blob hashes from EIP-4844. ETH2030 has a pinned seven-field Frame extractor and an ordered-commitment historical envelope test. A live Frame replay requires an executing EL and one agreed draft. [Upstream extraction][cl-devnet-hashes] · [ETH2030 extraction][local-frame-hashes] |

Keep the Heze regression cases when porting upstream VC, V6, gossip and
inclusion-list storage changes. The replacement gate exercises the same
payload with immediate and deferred true/false verdicts, two ancestors
including an empty beacon head, gossiped builder-bid and embedded self-build-bid recovery,
fork-choice reload, and Frame plus EIP-4844 blob order during fresh-node
backfill. Reuse the [CL gap audit](lighthouse-cl-gap-audit.md) for the EL and
independent-client requirements. Gloas and Heze light-client acceptance
remain separate because both inspected Heze paths still lack its headers.

## Dora and go-eth2-client

The Dora release already has the upstream FOCIL indexing path. Its master
has subsequent changes for slot metadata recovery, Heze state transition
and Bogotá Frame receipts. The master consumer subscribes to
`inclusion_list`, stores observations by slot, preserves the dependent root
and evaluation per block root, and writes unfinalized metadata for restart.
Finalized lookup uses a block database capable of slot metadata. The stock
slot inclusion-list API resolves a supplied hash to a slot, then returns the
slot view. The local adapter instead asks the BN for that proposal root and
its preceding-slot committee. [Upstream indexing][dora-index] · [Restart storage][dora-storage] · [Slot API][dora-api]

Use the upstream Dora Heze devnet branch as a draft reference. Qualify
mainline `5fb1ef23` for persistence and Frame handling. It includes the
Bogotá devnet 2 change from #920. It indexes per-frame receipt data, payer and execution/state
gas through its Spamoor module pin `d235dd5b6995`. Standalone Spamoor is
reviewed separately at `f0f83695`. Dora #920 targets Bogotá devnet 2,
which provides a closer comparison than the old local Type 6 display. That path still needs the pinned ETH2030 transaction and
receipt vectors before it can serve as the Frame oracle. The upstream FOCIL
evaluation currently labels Frame transactions outside enforcement. A
sponsored-Frame FOCIL test needs its own agreed eligibility rule and
matching display. [Dora #920][dora-frame-pr] · [Frame receipt indexer][dora-frames] · [Inclusion outcomes][dora-outcomes]

Dora master depends on `github.com/ethpandaops/go-eth2-client v0.1.11`.
Attestantio master has Gloas types and no Heze package. The ethpandaops
client includes separate Heze block and state types, execution envelopes,
and inclusion-list SSE decoding. The pinned tag's signed bid decoders read
the canonical `message` field. Current Lighthouse's Rust serialization
also names that field `message`. The local Go helper additionally supports
historical `message_gloas` and `message_heze` aliases, with a regression for
ambiguous input. Keep historical JSON as a fixture if that helper is retired.
[Upstream Dora module][dora-module] · [Upstream bid decoder][go-panda-bid] · [Lighthouse wire type][local-bid] · [Local alias decoder][local-go]

The actual SSZ boundary is the higher-priority blocker. The ethpandaops
Heze state and block body follow EIP-8015 and remove Eth1 fields and deposits.
The ETH2030 Lighthouse pin still populates `eth1_data`,
`eth1_data_votes` and `eth1_deposit_index` in the Heze upgrade and retains
the deposit list in the body. Source comparison predicts SSZ incompatibility between Dora `5fb1ef23`
with Go client `v0.1.11` and Lighthouse `1bf89a4fc`. Qualify either an
EIP-8015-aligned CL or a codec matched to the existing draft. The upstream
CL devnet branch already removes these Heze Eth1 fields and deposits, so
it is a useful wire reference. Compare both implementations against the
same genesis and BeaconState/BeaconBlock bytes. Version negotiation or
successful JSON block display cannot exercise the epoch-statistics path.
[Dora dependency change][go-panda-eip8015] · [Tag state layout][go-panda-state] · [ETH2030 upgrade][local-heze-state]

Use the master Dora candidate with `blockDb.engine: pebble` and persist
both the SQL database and Pebble directory across restart. The standalone
Dora configuration defaults to `blockDb.engine: none`. The updated
package template selects SQLite plus Pebble. Begin with the Fulu/Gloas
and ordinary Heze fixtures. [Dora defaults][dora-default-config] · [Package configuration][package-dora-config] Compare signed list
bytes, committee positions, dependent roots, late and equivocated
observations, and the target payload on a competing branch. Include a
restart after BN pruning, a list learned through recovery, a subscriber disconnect, and a fresh
explorer database. Verify which lists can be backfilled while the BN retains
them and that missing historical evidence is visible after pruning. Existing
data readback and cold-start behavior need separate assertions. Verify epoch
statistics from the CL's SSZ and compare them with the Beacon API. Retire the local Go codec when the released codec passes the wire cases on
each active profile that uses it. Retire the Dora adapter and CL endpoint
after the required proposal-root views survive the recovery and restart
cases with the endpoint disabled. Until then keep any compatibility code
specific to the wire or replay difference it covers.

## Spamoor

Upstream master already covers EOA, token, blob and SetCode workloads.
Its `frametx` scenario supports a pinned base envelope, per-frame receipt
checks and fee budgets. Its codec and fixtures cover the Bogotá devnet 2
max-fee representation, signature-length commitments, optional keyed
nonces and typed receipts. The upstream release `v1.2.4` predates
[Spamoor #312][spamoor-frame-pr]. Use the master source pin or a later
release that contains that change for the Frame trial. The examined upstream
Spamoor Heze devnet branch also predates that change.
[Frame scenario][spamoor-frame-scenario] · [Frame codec][spamoor-frame-codec]

Two independent checks remain for the product migration.

1. **Frame contract.** Select `frametx --envelope=base` for the seven-field
   envelope on ETH2030 `e628c770`. The default auto mode picks a layout from
   chain capabilities. Use the [pinned EIP-8141 revision][frame-draft] and
   the [Bogotá Engine draft][bogota-draft] to lock the test contract. Compare exact transaction RLP, signing hash and split-gas
   receipt bytes, plus sponsor balance and blob sidecars after EL support
   lands. The upstream scalar-nonce validator currently checks that keyed
   nonces are absent, then accepts without an upper-bound check.
   ETH2030 rejects the maximum `uint64` scalar nonce. Include this boundary
   and separate EIP-8250 fixtures. Upstream also probes support from
   predeploys, so a matching genesis and actual transaction admission are
   both required. Require a canonical included transaction and zero receipt
   discrepancies, since the scenario reports them as warnings. Use separate
   fixtures for sponsored and blob-bearing transactions. [Upstream validation][spamoor-frame-validate] · [ETH2030 validation][el-frame-validate] · [Capability check][spamoor-capability]
2. **Receipt and funding behavior.** The local receipt patch enumerates
   good clients on each attempt. Upstream master randomly selects a
   client for a bounded series of receipt attempts. Run a deterministic
   paired-client test where only one client has the receipt and another
   returns no receipt. Check wallet funding, reorg and post-confirmation
   reads. Preserve only the behavior the workload still requires.
   [Local receipt patch][spamoor-receipt-local] · [Upstream receipt lookup][spamoor-receipt-main]

The two broad profiles that select the old `eth2030/spamoor:v0.0.1` image
need an explicit migration. Run EOA, token, existing blob and SetCode
scenarios first. The historical EIP-7701 raw and UserOp scenarios still
have dedicated fixtures in the legacy branch. EIP-7701 was superseded by
EIP-8141. Remove old scenario selections from current profiles after
recording an archival run path. Core type `0x06` acceptance still depends
on the EL Frame state transition. [Profile inventory](devnet-third-party-inventory.md#explorers-load-generators-and-assertions) · [EIP-7701][eip7701]

## Orchestration and rollout order

The [implementation plan](devnet-upstream-implementation-plan.md) gives the
PR sequence, code entry points and required evidence for each profile.

The newer ethereum-package pin defaults to genesis generator `6.2.9` and
upstream Dora, Spamoor and Assertoor `:master` tags. It removes the
Heze-specific Quick Slots setting. In the package's local and shadow-fork
genesis path, either `network_params.frames_enabled` or `frame` or `bogota`
in any participant EL, CL or VC image name selects Frame-only preparation.
That path disables Heze on the CL side. For the combined profile, inspect
the generated CL fork epoch and EL Bogotá time, then observe both forks
activate in a single run with the selected images. Pin the effective image digests and test the native EL's
`geth` launcher flags and separate Track 4 genesis allocations.
[Package defaults][package-defaults] · [Frame profile logic][package-frame] · [Quick Slots change][package-quick]

1. Qualify Spamoor `f0f83695` for ordinary traffic and Dora `v1.25.0` for
   generic exploration using the existing stable control. Record the built
   image identities and hold the CL, EL, package and genesis fixed during
   each service swap. Qualify the package update in a separate pass.
2. Run the joint Dora/go-eth2-client Heze replay and provenance checks.
   Move to upstream master once the proposal-root, persistence and epoch
   statistics results agree with observed CL data.
3. Advance the Lighthouse Heze branch selectively. Test what the upstream
   devnet branch already covers against the existing fixtures, and retain
   the small required recovery and Engine changes. Run the Frame generator
   and historical sync gate with an EL that implements the selected draft.

The audit used upstream Git objects, local source and prior run results.
The ETH2030 Frame envelope, static rejection and receipt vector tests
passed. Isolated test runs for upstream Spamoor and the ethpandaops Go
client stopped at dependency download. The Spamoor retry used the exact
Geth tag from the local reference and then stopped at `gnark-crypto`
`v0.21.0`. The Go client stopped at its `dynamic-ssz` and crypto downloads.
Upstream runtime parity remains open.
This phase records the source-backed decisions and pending runtime gates.

[local-cl]: https://github.com/jiayaoqijia/lighthouse/commit/1bf89a4fc933d58828c39bcda3fe6eb481d680f5
[cl-stable]: https://github.com/sigp/lighthouse/releases/tag/v8.3.3
[cl-main]: https://github.com/sigp/lighthouse/commit/1bcdcbfa0e4b97f51b60212591391e956fd8a4da
[cl-devnet]: https://github.com/sigp/lighthouse/commit/f66bd5ac1051d443f11b5c701410203f4e3cf4b3
[dora-legacy]: https://github.com/jiayaoqijia/dora/commit/6f702f24e8445d9e4b764d5bf141c604b1c99d47
[local-dora]: https://github.com/jiayaoqijia/dora/commit/12a0198f237c8bf1973256ae5c2498d7cef90587
[dora-release]: https://github.com/ethpandaops/dora/releases/tag/v1.25.0
[dora-main]: https://github.com/ethpandaops/dora/commit/5fb1ef23668c13d0a23b481b2dee43993590cba2
[dora-heze]: https://github.com/ethpandaops/dora/commit/25aee6e70fc2ea1044a9db49822e25a3820021dc
[local-go]: https://github.com/fyInALT/go-eth2-client/commit/79989d2591dc7a964990097a1e048db49d4d3890
[go-canonical]: https://github.com/attestantio/go-eth2-client/commit/6d6c8310d52ef56f6382065a338a2e0bc34e5d39
[go-panda-tag]: https://github.com/ethpandaops/go-eth2-client/tree/v0.1.11
[go-panda]: https://github.com/ethpandaops/go-eth2-client/commit/78aca033aaa3ba7d4d9aeaa531d450820c3741f8
[local-spamoor]: https://github.com/jiayaoqijia/spamoor/commit/4908a05d77d3836f61524aeda6c4ee79f8c36097
[spamoor-release]: https://github.com/ethpandaops/spamoor/releases/tag/v1.2.4
[spamoor-main]: https://github.com/ethpandaops/spamoor/commit/f0f83695b24fe81ba25112e4415c2d75ef6d760c
[spamoor-heze]: https://github.com/ethpandaops/spamoor/commit/08d1959defd9e96bae4fac6494ba70cbf97dab1f
[local-package]: https://github.com/ethpandaops/ethereum-package/commit/c4a8e15d40f0965f3728f776d72f1954440ad96b
[package-main]: https://github.com/ethpandaops/ethereum-package/commit/d3014d6d23d683362025b9e23a0d69de49e77cca
[cl-baseline]: cl-iteration-roadmap.md#lighthouse-baseline-verification
[cl-light-pr]: https://github.com/sigp/lighthouse/pull/10401
[cl-main-notifier]: https://github.com/sigp/lighthouse/blob/1bcdcbfa0e4b97f51b60212591391e956fd8a4da/beacon_node/beacon_chain/src/payload_envelope_verification/payload_notifier.rs
[cl-devnet-notifier]: https://github.com/sigp/lighthouse/blob/f66bd5ac1051d443f11b5c701410203f4e3cf4b3/beacon_node/beacon_chain/src/payload_envelope_verification/payload_notifier.rs
[cl-devnet-vc]: https://github.com/sigp/lighthouse/blob/f66bd5ac1051d443f11b5c701410203f4e3cf4b3/validator_client/validator_services/src/inclusion_list_service.rs
[cl-devnet-bid]: https://github.com/sigp/lighthouse/blob/f66bd5ac1051d443f11b5c701410203f4e3cf4b3/beacon_node/beacon_chain/src/payload_bid_verification/gossip_verified_bid.rs
[cl-devnet-verdict-test]: https://github.com/sigp/lighthouse/blob/f66bd5ac1051d443f11b5c701410203f4e3cf4b3/beacon_node/beacon_chain/tests/envelope_verification.rs
[cl-devnet-fcu]: https://github.com/sigp/lighthouse/blob/f66bd5ac1051d443f11b5c701410203f4e3cf4b3/beacon_node/beacon_chain/src/beacon_chain.rs#L7363-L7384
[local-deferred]: https://github.com/jiayaoqijia/lighthouse/commit/d7a8eb5e61226636df17de30a91814e0c445122c
[cl-devnet-network]: https://github.com/sigp/lighthouse/blob/f66bd5ac1051d443f11b5c701410203f4e3cf4b3/beacon_node/network/src/service.rs#L1191-L1199
[local-recovery]: https://github.com/jiayaoqijia/lighthouse/commit/7f87e92c5cd420203cf972fa208082f39b0a594b
[local-retained]: https://github.com/jiayaoqijia/lighthouse/commit/5d450027089bc2383ce85aeb8b34597f80b58dc3
[heze-status]: ../STATUS.md#heze-consensus-validation-2026-11-16
[cl-devnet-hashes]: https://github.com/sigp/lighthouse/blob/f66bd5ac1051d443f11b5c701410203f4e3cf4b3/beacon_node/execution_layer/src/versioned_hashes.rs
[local-frame-hashes]: https://github.com/jiayaoqijia/lighthouse/blob/1bf89a4fc933d58828c39bcda3fe6eb481d680f5/beacon_node/execution_layer/src/versioned_hashes.rs
[dora-index]: https://github.com/ethpandaops/dora/commit/410cc9365fb9a0c9f217983be8bdc7678180dbf9
[dora-storage]: https://github.com/ethpandaops/dora/blob/5fb1ef23668c13d0a23b481b2dee43993590cba2/indexer/beacon/blockmetadata.go
[dora-api]: https://github.com/ethpandaops/dora/blob/5fb1ef23668c13d0a23b481b2dee43993590cba2/handlers/api/slot_inclusion_lists_v1.go
[dora-frame-pr]: https://github.com/ethpandaops/dora/pull/920
[dora-frames]: https://github.com/ethpandaops/dora/blob/5fb1ef23668c13d0a23b481b2dee43993590cba2/indexer/execution/txindexer/process_frames.go
[dora-outcomes]: https://github.com/ethpandaops/dora/blob/5fb1ef23668c13d0a23b481b2dee43993590cba2/services/chainservice_inclusionlists.go
[dora-module]: https://github.com/ethpandaops/dora/blob/5fb1ef23668c13d0a23b481b2dee43993590cba2/go.mod
[go-panda-bid]: https://github.com/ethpandaops/go-eth2-client/blob/ceca3c2d06146198f111b88f437fdef595b492d9/spec/heze/signedexecutionpayloadbid_json.go
[local-bid]: https://github.com/jiayaoqijia/lighthouse/blob/1bf89a4fc933d58828c39bcda3fe6eb481d680f5/consensus/types/src/execution/signed_execution_payload_bid.rs#L32-L44
[go-panda-eip8015]: https://github.com/ethpandaops/go-eth2-client/pull/64
[go-panda-state]: https://github.com/ethpandaops/go-eth2-client/blob/ceca3c2d06146198f111b88f437fdef595b492d9/spec/heze/beaconstate.go
[local-heze-state]: https://github.com/jiayaoqijia/lighthouse/blob/1bf89a4fc933d58828c39bcda3fe6eb481d680f5/consensus/state_processing/src/upgrade/heze.rs#L36-L49
[spamoor-frame-pr]: https://github.com/ethpandaops/spamoor/pull/312
[spamoor-frame-scenario]: https://github.com/ethpandaops/spamoor/blob/f0f83695b24fe81ba25112e4415c2d75ef6d760c/scenarios/frametx/README.md
[spamoor-frame-codec]: https://github.com/ethpandaops/spamoor/blob/f0f83695b24fe81ba25112e4415c2d75ef6d760c/txtypes/frame.go
[spamoor-frame-validate]: https://github.com/ethpandaops/spamoor/blob/f0f83695b24fe81ba25112e4415c2d75ef6d760c/txtypes/frame_validate.go#L147-L156
[el-frame-validate]: https://github.com/fyInALT/eth2030/blob/e628c770df98ba9ad269de74344afcdf4e6a92a3/pkg/core/types/frame_8141.go#L172-L175
[spamoor-capability]: https://github.com/ethpandaops/spamoor/blob/f0f83695b24fe81ba25112e4415c2d75ef6d760c/spamoor/framesupport.go
[spamoor-receipt-local]: https://github.com/jiayaoqijia/spamoor/commit/4908a05d77d3836f61524aeda6c4ee79f8c36097
[spamoor-receipt-main]: https://github.com/ethpandaops/spamoor/blob/f0f83695b24fe81ba25112e4415c2d75ef6d760c/spamoor/txpool.go#L1452-L1485
[eip7701]: https://eips.ethereum.org/EIPS/eip-7701
[package-defaults]: https://github.com/ethpandaops/ethereum-package/blob/d3014d6d23d683362025b9e23a0d69de49e77cca/src/package_io/constants.star
[package-frame]: https://github.com/ethpandaops/ethereum-package/blob/d3014d6d23d683362025b9e23a0d69de49e77cca/src/participant_network.star#L95-L111
[package-quick]: https://github.com/ethpandaops/ethereum-package/pull/1520
[frame-draft]: https://github.com/ethereum/EIPs/blob/dc5d5fa688b7fe87b95a907fcc128b4fde2f4d2f/EIPS/eip-8141.md
[bogota-draft]: https://github.com/ethereum/execution-apis/blob/981aa66d79364be8322a9c358aaf0a571cca0873/src/engine/bogota.md
[dora-default-config]: https://github.com/ethpandaops/dora/blob/5fb1ef23668c13d0a23b481b2dee43993590cba2/config/default.config.yml#L148-L179
[package-dora-config]: https://github.com/ethpandaops/ethereum-package/blob/d3014d6d23d683362025b9e23a0d69de49e77cca/static_files/dora-config/config.yaml.tmpl
[cl-main-transport]: https://github.com/sigp/lighthouse/blob/1bcdcbfa0e4b97f51b60212591391e956fd8a4da/beacon_node/execution_layer/src/engine_api/http.rs#L951
[cl-devnet-production]: https://github.com/sigp/lighthouse/blob/f66bd5ac1051d443f11b5c701410203f4e3cf4b3/beacon_node/beacon_chain/src/block_production/gloas.rs
[cl-local-production]: https://github.com/jiayaoqijia/lighthouse/blob/1bf89a4fc933d58828c39bcda3fe6eb481d680f5/beacon_node/beacon_chain/src/block_production/gloas.rs
[go-experimental]: https://github.com/ethpandaops/go-eth2-client/blob/f0131c6d811c75a7dafdf7abad37761a9684c5c4/spec/heze/beaconstate.go
[spamoor-proxy]: https://github.com/ethpandaops/spamoor/blob/f0f83695b24fe81ba25112e4415c2d75ef6d760c/spamoor/clientproxy.go
[package-heze]: https://github.com/ethpandaops/ethereum-package/commit/cc8a25bfa4eed67b7d141ac3e32bbb45b8d26430
[cl-legacy-plan]: lighthouse-legacy-branch-merge-plan.md
[dora-json-fallback]: https://github.com/jiayaoqijia/dora/commit/9a9e87905fca95c4b459a0c5f4027c94cf01bbcd
[spamoor-sponsored]: https://github.com/ethpandaops/spamoor/blob/f0f83695b24fe81ba25112e4415c2d75ef6d760c/scenarios/frametx-fuzz/README.md#L89-L94
[package-gas]: https://github.com/ethpandaops/ethereum-package/blob/d3014d6d23d683362025b9e23a0d69de49e77cca/src/package_io/input_parser.star#L570-L597
[package-prysm]: https://github.com/ethpandaops/ethereum-package/blob/d3014d6d23d683362025b9e23a0d69de49e77cca/src/vc/prysm.star#L74-L86
