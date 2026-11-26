# ETH2030 authenticated Frame simulation v1

This client API is selected on 2026-11-26 before implementation. EIP-8141 defines transaction execution and dimensional estimates, but no complete JSON simulation schema. This extension does not change transaction encoding, signing, admission or consensus.

`eth_simulateFrameV1(options, blockSelector?)` replays one authenticated canonical Frame envelope (base EIP-8141 or EIP-8250 when selected by EIP8250Time) against a disposable copy of the selected block's post-state. The optional selector follows existing historical selectors (default latest); it does not implicitly mine the next block. Options:

- `rawTransaction`: required strict hex DATA containing the plain canonical typed envelope. Blob sidecars are not needed to execute versioned hashes; network wrappers are rejected by this API. This is distinct from public submission, which requires valid sidecars.
- `slotNumber`: optional strict uint64 QUANTITY setting the exact simulation slot; otherwise use the selected header slot. Slots are never calculated from timestamps.
- `blockOverrides`: optional existing trace block override schema for number, time, gasLimit, feeRecipient, prevRandao, baseFeePerGas and blobBaseFee. Withdrawals and beaconRoot do not apply to this single-transaction API. Overrides are explicitly hypothetical and affect no canonical state.
- `stateOverrides`: optional existing account/state override schema. Moving precompiles is rejected in v1; such an override would require explicitly installing the changed runtime map for Frame execution.
- `trace`: optional boolean (default false) enabling a bounded opcode trace with per-frame identity and return/error outputs. `traceLimit`: optional uint64 QUANTITY, 1..10000 (default 1000).

The transport cancellation context applies, with a maximum five-second execution timeout. Raw payloads are capped at 128 KiB, both declared dimensions at 50 million, retained frame return data at 1 MiB in aggregate, and opcode capture at existing trace byte bounds. Invalid JSON/options/envelope/authentication returns invalid-params. Timeout, unavailable state or exhausted capture limits returns an explicit execution error. A consensus execution failure returns `valid: false`, an error and the attempted frame results; it is never misreported as successful estimation.

A valid result identifies the original transaction hash and immutable signing hash, payer, intrinsic gas, calldata floor, declared dimensional reservations, actual charged and pre-refund execution/state usage, blob gas/cost, escrow and settlement refund/tip, ordered per-frame measured gas/status/logs/return data and optional opcode trace. Estimates are measurements under these exact authenticated limits and context. They are not promised minimum valid limits. Signatures, sender, approvals, nonce and fee settlement use the production executor without relaxed validation. No simulation result can be submitted as a replacement for a signed transaction.

`eth_call`, `eth_estimateGas`, access-list creation, ordinary `eth_simulateV1` calls and ordinary `debug_traceCall` explicitly reject `type: 0x6`; callers use the extension for complete Frame input. This prevents silent ordinary-CALL fallback. Existing transaction replay/receipt trace routes retain their own complete envelope dispatch.

The receipt RPC transaction-level status is client-derived, never encoded into canonical Frame receipt RLP: success iff no executed frame failed (skipped frames are not independently failures). Per-frame status and gas remain authoritative. All scalar per-gas display values are derived using the original block's base and blob base fees; `fees.maxFee` remains the signed total budget.

Keyed transaction query objects replace nonce with nonceKeys (an array of uint256
QUANTITYs, including an explicit empty array for the legacy domain) and nonceSeq
(a uint64 QUANTITY). The original raw hash/signing hash and two-field total fee
budget are preserved across pending, mined and restart queries. The typed direct
API uses the shared Frame formatter, including the original block's fee context.
eth_getTransactionCount continues to return the legacy account nonce.
txpool_content/inspect identify a nonempty keyed domain by keyed:<transaction-hash>
inside its sender map; ordinary entries retain their scalar nonce keys. This
local indexing avoids dropping a same-integer-sequence ordinary transaction.
