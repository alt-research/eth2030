# Pinned EIP-8141 execution

## Source boundary

This implementation extends the seven-field codec merged in ETH2030
`cc567f93`. It follows EIP-8141 at EIPs commit
`dc5d5fa688b7fe87b95a907fcc128b4fde2f4d2f`, including the EIP-8037 and
EIP-8038 parameter tables at that revision. References remain read-only.
The active native fork is `HogotaTime` / `bogotaTime`.

## Implemented paths

| Area | Production path and behavior |
| --- | --- |
| Activation | `execution.applyTransactionInternal` selects the seven-field transition at Bogotá. Historical Frame envelopes are rejected at that fork. |
| Authorization | Protocol signatures validate before execution. Default-code VERIFY uses signature index 0 for execution approval and index 1 for payment-only approval. APPROVE scopes are bitmasks: payment 1, execution 2, both 3. |
| Journaling | EVM snapshots include approvals, the payer, state-pool spending, storage ownership and prior frame receipt state charges. The Frame BAL journal retains executed accesses while emitting only final changes; skipped targets are excluded. Reverted calls and atomic groups restore all of these alongside state and logs. |
| Gas | Each frame has explicit execution and state pools. Account creation costs 183600 state gas, a new slot 97920, and deployed code 1530 per byte. Storage-write, account-access and creation execution costs follow the pinned EIP-8038 table. |
| State refills | Clearing a slot originally zero credits the frame that allocated it. A later reverting frame restores that earlier receipt's state charge. Execution gas remains spent. |
| Settlement | Payment approval checks and escrows the total fee budget before consuming the sender nonce. Settlement returns unused funds and credits the actual capped tip. Block value uses the exact tip rather than multiplying the rounded GASPRICE by gas used. |
| Block accounting | Builders reserve the Frame execution and state dimensions independently. Header gas uses the maximum of the cumulative dimensions; receipt cumulative gas remains the sum paid by transactions. Import uses the same accounting. |
| Opcodes | APPROVE, TXPARAM, FRAMEDATALOAD, FRAMEDATACOPY, FRAMEPARAM, SIGPARAM and SIGDATACOPY use the pinned mappings. Future receipt reads and protocol-signature witness reads halt exceptionally. |
| Public admission | A copied StateDB executes only the bounded validation prefix. Admission checks the exact approval scopes, trace dependencies, expiry, current nonce, fee budgets, both replacement bumps and per-payer max-fee exposure. Head/state/fee changes revalidate pending Frames. Node startup and production head recording supply the actual fee context. P2P, sync and native fork-choice state resets refresh it under the pool lock. |
| Blob networking | Blob-bearing Frames carry the EIP-7594 version-1 pooled wrapper. Shape, commitment hashes and every cell proof are checked. The production backend authenticates a blob's 128 cells with one batch verification. Block bodies retain the plain envelope. Full Transactions messages reject blob-bearing Frames. |
| Receipts and RPC | Canonical receipts retain the payer, per-frame statuses, logs and both gas dimensions. Disk readback derives block-dependent fee/blob metadata. eth/70 pages retain Frame fields and pagination metadata. RPC exposes the seven-field transaction data and computes transaction gas price using the including block's blob base fee. |
| Native Engine | HTTP regressions execute sponsored Frames through inclusion-list attributes, newPayloadV6, forkchoiceUpdatedV5 and node restart. A trusted genesis head can start Bogotá without a submitted payload/list record. |
| Workload tool | `pkg/cmd/send-frame` signs the canonical envelope, supports a default-code sponsor and optionally creates a real blob sidecar. |

Ordinary transaction formats retain the existing gas schedule. This change does
not qualify a complete migration of those formats to the latest EIP-8037
reservoir model.

## Public paymaster boundary

The pinned EIP includes `assets/eip-8141/CanonicalPaymaster.sol`, but does not
freeze a compiler, compiler settings or deployed runtime for the required exact
code match. The reference also places a signature of the canonical signing hash
in frame calldata, even though that calldata is itself committed by the signing
hash; bespoke witnesses must instead use ARBITRARY signature entries under the
same EIP. A local `solc 0.8.38+commit.1835d247` invocation with
`--via-ir --optimize --bin-runtime` rejected both inline `verbatim` calls as
unknown functions; the [compiler evidence](verification/frame-8141-canonical-paymaster-compile.json)
retains the command and diagnostics. A compatible compilation path and a
consistent witness convention must be frozen before granting the exception.

ETH2030 now freezes the [v1 candidate](../pkg/core/framepay/README.md) as a
local experimental runtime. It preserves the reference owner and twelve-hour
withdrawal layout and uses protocol-validated signature entry 1 with empty
`msg`. Exact runtime matching, actual prefix execution and withdrawal-adjusted
aggregate reservations qualify this one implementation. Unknown code retains
generic trace rules and the one-pending coded-paymaster cap. This explicitly
records the witness/ABI/compiler differences from the uncompiled upstream asset;
formal cross-client runtime agreement remains a promotion gate.

## Verification

Behavioral regressions exercise real EVM bytecode and inspect state, gas,
receipts, logs and balances. They cover sponsorship, sender-account creation,
code deposit, state exhaustion, cross-frame refills, deployed-sender nonce increments and overflow rollback, nested/delegate approvals,
atomic rollback, introspection, transient isolation, signature restrictions,
public-prefix simulation, replacement and exposure release. Cross-transaction slot originals and canonical receipts are checked on both memory and binary-overlay state backends.

Integration tests build and import sponsored and blob-bearing blocks with real
KZG data, switch canonical branches, reopen FileDB and run the production startup
replay. Native HTTP tests exercise signed RPC admission, inclusion-list payload
construction, Engine import, FCU and restart readback on a second node.

Mutation checks independently break dimension reservation, state-receipt
snapshotting and caught-child trace enforcement. Each corresponding behavioral
regression must fail. Full Go, race, build and live-run outcomes are recorded in
STATUS.md after execution.

## Remaining promotion gates

- Qualify the frozen paymaster v1 candidate on live traffic and align its runtime/witness ABI with independent clients.
- The sponsored FOCIL/finality/restart gate passed on EL `1c20e3c7` and
  Lighthouse `a05a56de`; see the [retained wire and settlement evidence](verification/frame-8141-20261118-final.json).
  Fresh CL historical backfill remains a separate gate.
- Exercise blob commitment ordering and Frame historical backfill through a fresh
  CL, including an independent compatible execution source.
- Qualify upstream workload/explorer consumers against the signed envelope and
  RPC fields before retiring the maintained service modifications.
- Keep the independent-client, state-sync, external-builder/PTC and remaining Heze
  gates in [the next-stage plan](next-stage-work-plan.md) open until measured.

## Independent-client revision audit (2026-11-18)

The upstream Geth `bogota-devnet-2` branch resolves to
`f71f4be8027bc0828148ae03424231092ed7ab66`. Its
[Frame type](https://github.com/ethereum/go-ethereum/blob/f71f4be8027bc0828148ae03424231092ed7ab66/core/types/tx_frame.go)
incorporates EIP-8250 nonce keys and nonce sequence in place of the base scalar
nonce. It also prices their encoded bytes as transaction data. An empty key set
uses the account nonce, but does not restore the base seven-field envelope or
its signing hash. The installed released Geth v1.17.10 type/VM sources contain
no Frame type implementation.

Consequently neither is recorded as an independent execution oracle for this
pinned base envelope. Geth's experimental implementation is an architecture
reference for a separately scoped protocol-family migration; accepting its
wire format silently would change signatures and fee accounting. Fresh CL
history against an existing compatible EL and independent EL execution remain
distinct gates. The pinned upstream source and the
[Bogotá test release](https://github.com/ethereum/execution-specs/releases/tag/tests-bogota-devnet%40v2.0.0)
make the additional EIP-8250/7906/8298 scope explicit.

### Coded sponsorship live RPC follow-up

The first coded-paymaster run (EL `387c7a6d`, Lighthouse `a05a56de`)
finalized transaction `0x50eceea816a34b2ec7493a02a8e00765982d0ee0489d464db921cc2376477653`
at slot 65 on both peers. Its subsequent blob submission failed because the
production `EthAPI` decoded a pooled Frame wrapper as a plain transaction.
The [failed-run record](verification/frame-8141-20261118-paymaster-rpc-failed.json)
retains the successful sponsored result and complete enclave cleanup; it does
not establish blob or fresh-CL acceptance.

Both public transaction entrypoints now use the pooled decoder. The actual
HTTP regression submits an incorrect final-cell KZG proof, checks rejection,
then submits the corrected wrapper with the same transaction identity. It
checks admission, retained sidecar, and all 128 ordered hex proofs obtained
through authenticated `engine_getBlobsV2`. This also exposed the missing
`BlobAndProofV2` Engine DATA JSON codec, which now follows the Osaka wire
fields. The direct facade regression uses a real encoded legacy transaction
and checks decoded identity rather than accepting arbitrary bytes.

### Consensus service state fixture

The second coded/blob run finalized sponsored Frame slot 66 and blob Frame
slot 162 on both peers. The observer process rejected the mutually exclusive
checkpoint/genesis-sync options. The [failed observer record](verification/frame-8141-20261118-blob-observer-failed.json)
retains both finalized workloads. The controller removed the failed observer,
both pairs, labeled volumes and its engine. Its archived Heze state at slot
192 is retained in the [consensus replay module](../tools/consensus-replay/README.md).
The byte-for-byte fixture and its original state root qualify a minimal
upstream-shaped compatibility type, including all validator keys and balances.
Removing the root comparison makes the changed-state and unrelated-header
regressions fail. This is decoder evidence; Dora's actual epoch consumer and
service restart remain unqualified.

### Upper scalar nonce admission

The final scalar transaction nonce allowed by EIP-2681 and the pinned Frame
validation is `2^64 - 2`. A shared pool guard previously added its future-nonce
gap to the current state nonce and wrapped near that bound. The pool regression
failed with `nonce too high` before changing it to compare the nonnegative
difference. The actual HTTP/Engine/restart integration now runs both at nonce
zero and at this upper nonce, checking the post-approval nonce through both
nodes and the reopened database. The live `38bbcff5` image uses ordinary nonce
values; the later guard correction has separate local regression evidence.

### Historical body serving after CL payload pruning

The [failed fresh-CL record](verification/frame-8141-20261118-history-serving-failed.json)
retains finalized coded and blob Frames on EL `38bbcff5`. The source CL's own
historical envelope endpoint returned `PayloadBodiesByHashV2Failure` with
`hex must have 0x prefix` for the sponsored Frame and
`PayloadBodyMissingBlockAccessList` for the blob and checkpoint blocks. The new
observer remained at checkpoint slot 192 until the bounded timeout. All owned
enclave resources were removed and its engine stopped.

EL `e0451092` repairs the actual wire path: V1/V2 body transactions are hex
DATA, native validated RLP BALs persist atomically with blocks and are served
through both hash and canonical-range retrieval, and nullable/pre-fork fields
follow the Engine contract. HTTP integration compares exact original Frame and
BAL bytes on both nodes and after FileDB reopen. Separate mutations omitting
BAL persistence and the V2 DATA codec each fail this integration. Range controls
cover invalid quantities, bounds, error codes, overflow and trailing heights.

### Heze checkpoint IL view

On the repaired EL, source historical APIs return the original finalized
sponsored, blob and checkpoint envelopes. The [next failed observer record](verification/frame-8141-20261118-checkpoint-parent-failed.json)
also retains an actual blob-receiver restart with unchanged receipt, parent/post
storage and sponsor balances. Fresh range sync still failed in the CL: the
Heze notifier looked up the checkpoint's IL view using an unavailable pre-anchor
parent (a debug node at anchor slot 224 requested its missing slot-223 parent).
The Heze `get_forkchoice_store` specification initializes the trusted anchor's
IL satisfaction to true. This is a separate Lighthouse bootstrap repair; source
history serving success does not establish fresh CL acceptance.
