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
| Public admission | A copied StateDB executes only the bounded validation prefix. Admission checks the exact approval scopes, trace dependencies, expiry, current nonce, fee budgets, both replacement bumps and per-payer max-fee exposure. Head/state/fee changes revalidate pending Frames. |
| Blob networking | Blob-bearing Frames carry the EIP-7594 version-1 pooled wrapper. Shape, commitment hashes and every cell proof are checked. Block bodies retain the plain envelope. Full Transactions messages reject blob-bearing Frames. |
| Receipts and RPC | Canonical receipts retain the payer, per-frame statuses, logs and both gas dimensions. Disk readback derives block-dependent fee/blob metadata. eth/70 pages retain Frame fields and pagination metadata. RPC exposes the seven-field transaction data. |
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
same EIP. Its inline `verbatim` operations require a compatible compilation path.

No arbitrary contract is granted the canonical-paymaster exception. Coded
sponsors use the generic validation trace and the one-pending non-canonical
paymaster limit. Default-code sponsors use aggregate per-payer exposure. Runtime
pinning, a consistent witness convention, delayed-withdrawal accounting and the
canonical exception remain an explicit promotion gate.

## Verification

Behavioral regressions exercise real EVM bytecode and inspect state, gas,
receipts, logs and balances. They cover sponsorship, sender-account creation,
code deposit, state exhaustion, cross-frame refills, nested/delegate approvals,
atomic rollback, introspection, transient isolation, signature restrictions,
public-prefix simulation, replacement and exposure release.

Integration tests build and import sponsored and blob-bearing blocks with real
KZG data, switch canonical branches, reopen FileDB and run the production startup
replay. Native HTTP tests exercise signed RPC admission, inclusion-list payload
construction, Engine import, FCU and restart readback on a second node.

Mutation checks independently break dimension reservation, state-receipt
snapshotting and caught-child trace enforcement. Each corresponding behavioral
regression must fail. Full Go, race, build and live-run outcomes are recorded in
STATUS.md after execution.

## Remaining promotion gates

- Pin and qualify the canonical-paymaster runtime and withdrawal accounting.
- Finalize sponsored Frame traffic through the current Lighthouse pair and retain
  exact receipts, balances and historical readback after an EL restart.
- Exercise blob commitment ordering and Frame historical backfill through a fresh
  CL, including an independent compatible execution source.
- Qualify upstream workload/explorer consumers against the signed envelope and
  RPC fields before retiring the maintained service modifications.
- Keep the independent-client, state-sync, external-builder/PTC and remaining Heze
  gates in [the next-stage plan](next-stage-work-plan.md) open until measured.
