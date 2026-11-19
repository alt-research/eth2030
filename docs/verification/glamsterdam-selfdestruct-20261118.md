# Glamsterdam EIP-8246 lifecycle verification

Implementation [PR 72](https://github.com/alt-research/eth2030/pull/72), branch `feat/glamsterdam-selfdestruct`, based on `origin/master` `38ea9259`.

## Specification and behavior

The implementation follows [EIP-8246 at EIPs 3db38a3](https://github.com/ethereum/EIPs/blob/3db38a3725105cb67b8804ac3e147967113bc3ca/EIPS/eip-8246.md), read with [EIP-6780](https://eips.ethereum.org/EIPS/eip-6780) and [EIP-161](https://eips.ethereum.org/EIPS/eip-161). BAL output follows [EIP-7928 recording semantics](https://github.com/ethereum/EIPs/blob/3db38a3725105cb67b8804ac3e147967113bc3ca/EIPS/eip-7928.md#recording-semantics-by-change-type).

Ordinary EVM transactions now journal same-transaction creations, including prefunded addresses. Call and creation rollback remove the corresponding creation markers. Frame transactions retain their existing state-snapshot journal. CREATE and CREATE2 both reject insufficient endowment and nonce overflow before changing the creator nonce; successful CREATE2 increments it as required.

At Amsterdam (also the existing private Glamsterdan compatibility fork), destruction preserves the balance. Code, storage and nonce remain observable until transaction finalization. After refund and fee settlement, finalization resets the account and restores its final balance; a zero-balance result retains an empty-account deletion marker. Existing contracts retain EIP-6780 behavior. The ordinary Cancun path also tracks newly created accounts and preserves its pre-Amsterdam self-burn/deletion semantics. Stateless account existence now distinguishes a destruction marker from completed deletion, preventing a later CALL from recreating and clearing a still-observable account.

BAL collection retains the first value seen per changed field, seeds account baselines before upfront fee/nonce changes without manufacturing an access, and reconciles recorded writes against final transaction state. Reverted writes, writes restored to their initial value and storage cleared by destruction become reads. Temporary nonce/code writes disappear. CREATE/CREATE2 emit the missing nonce/code tracking events. Tracker reset clears the seeded baselines between transactions. This does not qualify every unrelated BAL access-discovery path.

## Evidence

New tests cover:

- Native top-level creation/destruction, zero/nonzero balances, prefunded targets and late fee-recipient credits.
- Internal CREATE and CREATE2, self/other beneficiaries, later CALL credits, repeated transactions and CREATE2 address reuse; addresses are checked using go-ethereum's independent address implementation.
- Code and storage visibility before transaction end, repeated execution after destruction and clearing afterward, across Memory, Trie, Stateless and Binary state backends, including commit readback.
- Reverted creation, initcode failure, creation-marker rollback, creator nonce overflow and insufficient endowment.
- Existing-contract preservation, exact Amsterdam activation boundary and prior Cancun deletion/burn behavior.
- Final BAL storage/nonce/code/balance semantics, reverted/restored SSTORE, upfront-fee baseline retention and tracker reuse.
- Real canonical Frame execution and final BAL, including balance preservation after destruction and later funding.

A controlled run restored production files to `38ea9259` while keeping these tests. A no-op `FinalizeSelfDestructs` shim only allowed the new diagnostic to compile. The run failed on native cleanup, all four backend destruction markers, internal CREATE/CREATE2 cleanup, each final BAL scenario and canonical Frame balance preservation. Production files were restored before the passing runs. The raw failure output is retained locally at `/tmp/glamsterdam-8246-baseline-failures.log`.

Validation:

```sh
cd pkg
go test -count=1 ./bal/... ./core/... ./engine/... ./node/... ./eiptest
go test -race -count=1 ./bal ./core/vm ./core/execution -run 'TestEIP8246|TestTrackerFinalState'
go vet ./bal ./core/execution ./core/vm ./core/state
go build ./...
```

All listed checks passed. No cross-client devnet, full upstream EEST permutation matrix or final 8037/8038 gas qualification is claimed. This corrects consensus execution and requires a fresh private-chain history or an explicit migration strategy for an existing experimental chain. PR 71 separately corrects protocol ETH-transfer logs; it must accompany this branch in the aggregate qualification.
