# Independent Glamsterdam gas corpus

Branch: `test/glamsterdam-gas-corpus`, based on `origin/master` `38ea9259`.

## Scope and reproducibility

`pkg/cmd/glamsterdam-compare` executes 22 frozen scenarios under both Osaka control and Amsterdam candidate rules (44 comparisons). Each client decodes the same signed dynamic-fee transaction. The report retains complete pre-state, native/reference chain configurations, block context inputs, signed bytes, validation errors, included success/failure, user and block gas dimensions, logs, exact observed accounts and final roots. It records the source revision, dirty state, reference module/version and EIPs revision. The corpus has no live-chain side effects.

Reference transition: unmodified `github.com/ethereum/go-ethereum v1.17.10`, pinned by the existing module dependency (`h1:b3WO+xux3pItpm13kqT5Xddv01zOxBFZxQgk/Ate4/M=`). The command rejects an unknown/replaced/different reference version until requalification. This is an independent implementation observation, not normative authority or a claim of full SDK/EIP conformance. The candidate context follows [EIPs 3db38a3](https://github.com/ethereum/EIPs/tree/3db38a3725105cb67b8804ac3e147967113bc3ca): EIP-2780 base 12,000/value 6,000; EIP-8037 CPSB 1,530 and 120/64-byte account/slot creation; EIP-8038 access/write repricing and the current merged LIFO model. Open alternatives #12581/#12582 are not mixed in.

The reference EVM uses SDK CanTransfer/Transfer and an explicit complete block context, including CPSB 1,530 and SLOTNUM input. It does not use the native transaction processor, custom Geth precompile injection or an adapter that omits Amsterdam activation. Existing `geth.MakeBlockContext` omits CPSB: using its default zero would remove state creation costs. The new-account runtime-OOG test detects that mistake; temporarily replacing the explicit CPSB with zero makes that test fail. The local mutation log is `/tmp/glamsterdam-gas-corpus-oracle-mutation.log`.

Run from any checkout containing this command:

```sh
cd pkg
go run ./cmd/glamsterdam-compare > /tmp/glamsterdam-gas-observations.json
```

A report with mismatches is a successful measurement, not a passing release gate. Invalid transaction and included failed transaction remain separate (`valid` and `receipt_status`); status/gas fields are meaningful only for valid transactions. Error text is retained for investigation but not compared as a consensus byte format. Whole-state roots and account balances can differ solely because gas/fee accounting differs; inspect raw fields before attributing a separate state-transition defect.

## Cases and acceptance

Cases include empty, funded and partially exhausted reservoirs, nested refills and exceptional exits. They cover existing/new accounts, zero/nonzero value, low-budget new-account runtime failure, calldata floor, access lists, new/cleared/restored/reverted storage, exceptional halt, successful/reverted nested DELEGATECALL refills, successful/reverted creation, created-account selfdestruction, execution-cap overflow and total-cap overflow. The reference new-account failure must be an included transaction whose target remains absent; the signed Osaka transfer control must match exactly, including its root, and use 21,000 gas.

Passing checks:

```sh
cd pkg
go test -count=1 ./cmd/glamsterdam-compare
go test -race -count=1 ./cmd/glamsterdam-compare
go vet ./cmd/glamsterdam-compare
go build ./...
```

The archived baseline report and a summary are stored in `docs/audits/glamsterdam-gas-2026-11-18/` after a clean source commit. An aggregate implementation report is collected separately on the integration branch. Neither is a waiver of a mismatch.

Uncovered: full EEST permutations, 7702 authorization/application invariants, mixed multi-transaction settlement/next base fee, RPC method semantics, full client processes/devnets/recovery and all unmerged gas alternatives. GL-002/020/060 remain partial until those dependencies and the selected model are qualified. The command does not modify or enable consensus gas rules.
