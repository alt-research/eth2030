# Schema and semantic decisions

HGT-007. Normative sources remain pinned in [source-lock.json](../../../specs/hegota/source-lock.json). The version-4 [semantic corpus](../../../specs/hegota/semantic-goldens.json) now contains concrete inputs and expected outputs for all nine former trace seeds: seven native VM/executor cases and two isolated conditional rule oracles. There are no unexecuted natural-language trace seeds. A further 25 concrete conditional boundary vectors cover key shape/exhaustion/rollback and root window/domain/reorg rules; expected root-domain hashes were frozen with independent upstream Geth Keccak. This completes the bounded A fixture/ADR deliverable without claiming all later native/independent-client conformance.

## ADR-A01: encoding, hashes and signature commitments

Use pinned base 8141's seven fields: `[chain_id,nonce,sender,frames,signatures,fees,blob_versioned_hashes]`. Frames have six fields, limits are `[execution,state]`, and fees are `[max_fee,max_priority_fee_per_gas]`. Transaction hash is Keccak of `0x06 || rlp(payload)`. Canonical signing hash replaces each empty-message signature's bytes with its original byte length encoded as an RLP integer; explicit-message signature bytes remain committed.

[frame-schema.json](../../../specs/hegota/frame-schema.json) fixes the value tree/static widths; mode/flag/batch and stateful rules remain normative. Seven [codec goldens](../../../specs/hegota/codec-goldens.json) come from independent minimal Python RLP/Keccak and are cross-checked with upstream Geth RLP/Keccak by `TestHegotaIndependentEncodingGoldens`. Coverage includes same/changed signature length, explicit messages, maximal base nonce and long 64-frame RLP. Arbitrary encoding bytes are not authenticated execution transactions. D09 width disagreement stays open; no EELS type is silently substituted for EIP validity.

## ADR-A02: approval identity, payment and nonce

Scopes are bitmasks: payment 1, execution 2, combined 3. Reject zero/out-of-flags scope and wrong executing address/target. Execution requires the sender target and no earlier approval. Payment requires sender approval, no payer and sufficient funds. A successful payment transition consumes the sender nonce once and collects max cost; execution-only approval does not. APPROVE charges RETURN-style memory expansion, without extra opcode base gas.

The four `approval_calls` goldens execute the existing canonical `vm.RunFrame8141`: zero scope, wrong target, payment first, then execution/payment/repeated payment. They assert errors, approval context, payer, nonce, both balances and exact execution gas. The wrong-target case deliberately isolates the opcode precondition that whole-envelope validation can reject earlier; it is not asserted to be a valid network envelope. The once-only case independently fixes escrow at 103,425 and preserves the first payer/nonce after repeated payment reverts.

Default-account canonical signatures and delegation are already implemented separately; code size is not a substitute for authentication. Public admission, block validity and IL eligibility remain different contracts.

## ADR-A03: independent gas dimensions and settlement

Frames use separate execution/state budgets; the scalar reservoir model does not apply inside canonical Frames. Intrinsic cost starts at 12,000 plus 475 per frame and data/signature/value charges. Frame reservation, calldata floor, state budget and blob fee jointly constrain max gas/max fee. Payment collects max cost at approval; settlement burns base/blob fees, credits proposer tip and refunds the payer. Block execution counts pre-refund/floor usage; state usage is net. Refills never reopen a different frame's spendable pool.

Five independent arithmetic goldens recompute numeric intrinsic/floor/reservation/escrow/refund/block-accounting outcomes, including capped positive refunds, floor saturation and blob fees with a clipped upfront tip budget. `TestHegotaGasOracleGoldens` consumes all five through the existing native `GasBudget`, `Payment` and `Settle` methods. Execution usage is an oracle input in this layer, not an extra VM execution claim. Native transaction goldens additionally fix per-frame execution/state usage across rollback. This does not replace later exhaustive independent gas, resource and large-corpus qualification.

## ADR-A04: journal, batches and receipts

Snapshots journal approval, warm sets, state and state-charge ownership/earlier receipt changes. Failed VERIFY makes the transaction invalid and restores all transaction effects and gas reservations. Atomic body failure restores batch-entry state/logs; executed frames retain execution status/gas with zero state gas, and later batch members are skipped status 2. Earlier successful payment/nonce survives body failure. Transient state is cleared between frames.

Three concrete `transaction` goldens use shared native `ApplyTransaction`: nested successful APPROVE followed by enclosing VERIFY revert; three-frame atomic body failure; and a later batch that deletes an earlier frame's slot then fails. Assertions cover whole-state root restoration for invalid VERIFY, both gas pools, nonce, payer, storage, logs, receipt statuses, exact execution gas and net state-gas ownership. Expected values are frozen from specification reasoning; tests never regenerate them from native outputs.

Canonical receipt is `[cumulative_gas_used,payer,[frame_receipt...]]`, each frame receipt `[status,[execution,state],logs]`, with ordered frame logs. Existing codec/persistence/restart tests are reused; the frozen legacy scalar receipt is not an oracle for this format.

## ADR-A05: isolated keyed-nonce and recent-root rules

8250 replaces the base scalar nonce with `(nonce_keys,nonce_seq)`; keys must be strictly increasing, bounded, and uniquely select domains. Empty keys select the legacy nonce. Nonempty keys at `MAX_NONCE_SEQ` select unused single-use domains and consume them to MAX exactly once with successful payment approval. Failed approval rolls that transition back; a later body revert does not. Fresh-key charge is 97,920 state gas per key and access is 12,100 execution gas per key under the pinned dependencies.

The two-key single-use oracle and 17 additional nonce boundary cases check MAX writes, preservation of the current approval-time legacy nonce for keyed transactions, the 16-key valid boundary, once-only replay rejection, 195,840 state charge and 24,200 access cost. Negative Python regressions alter expected nonce/cost/replay, state-budget sufficiency and approval rollback. This is an isolated specification oracle; D01's conflicting fee/envelope prose prevents claiming combined network activation. In these isolated existing-sender cases, `valid` means nonce pre-state validity and `approval_applied` reports whether consumption committed. A later body revert preserves the prefix; a wholly invalid transaction discards it. `execution_access` is the declared keyed access cost, not a claim of paid fees for invalid transactions.

8272 uses canonical VERIFY root references, fixed-width source/entry/storage domains and explicit header/Engine `slotNumber`, never timestamp-derived slots. A write at S is usable only at S+1 through S+8191. The domain golden fixes source ID, ring index, entry hash and storage key. The core window oracle and eight storage/domain boundary cases check S, S+1, S+8191, S+8192, plus reorg to absent/orphaned entries; wrong boundary/reorg expectations fail regressions. System-contract deployment, bytecode and public-pool/native extension integration remain HGT-018/022 under their admission gates. This oracle does not claim those paths are activated.

## ADR-A06: provenance, consumers and updates

`codec_goldens.py` only compares unless `--write` is explicitly requested. `semantic_goldens.py` checks arithmetic/domain, structured fixture coverage and both conditional rule oracles; the Go suite actually executes all seven base cases. Criterion acceptance requires their individual pass events and current artifact/source fingerprints. Altering expectations without executing the corresponding native cases cannot be justified by a completion label.

Upstream changes retain old/new commits/hashes, date, affected tasks and admission/defer rationale. Baseline expected results never come from the native run. The [original ADR snapshot](https://github.com/alt-research/eth2030/blob/6a17479/docs/plans/hegota-a/semantic-adrs.md) stays immutable; current fixtures do not retroactively turn historical seeds into executed evidence.
