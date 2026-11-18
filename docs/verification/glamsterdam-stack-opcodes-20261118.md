# Glamsterdam EIP-8024 bytecode verification

Branch: `feat/glamsterdam-stack-opcodes`, based on `origin/master` `38ea9259`.

## Specification and changes

Read the local EIP-8024 reference and [the pinned upstream specification](https://github.com/ethereum/EIPs/blob/3db38a3725105cb67b8804ac3e147967113bc3ca/EIPS/eip-8024.md) (file blob `b15042f65157134c398a8dd0f2d6759893f2f81e`). The native implementation used an incorrect immediate permutation despite its opcode-presence tests passing. For example, `e680` must be DUPN 17; the old implementation decoded 0x80 as 108. EXCHANGE must use the specified XOR permutation and accept 0x50 and 0x51. Both were rejected by the old range check. Canonical Amsterdam also lacked these opcodes unless the private gas fork was enabled.

DUPN/SWAPN now decode using the specified modulo permutation. EXCHANGE uses XOR 143 and reserves 82 through 127. All three are installed in the Amsterdam jump table and inherited by later private forks. Existing JUMPDEST analysis remains the ordinary PUSH-only analysis, as required for backward compatibility. Old tests that asserted incorrect wire values are updated to the published encoding. The native transaction test no longer contains unused bytecode or accepts mere opcode presence as evidence.

The separate, unactivated EOF validator retains its own experimental instruction-format handling; this review qualifies the legacy bytecode instructions specified by EIP-8024, not a future EOF fork.

## Verification

- Every execution vector published in the EIP is exercised as raw bytecode, including zero-padded code-end EXCHANGE, forbidden immediates and jumps past invalid opcodes. Published final stacks are checked explicitly.
- Every DUPN/SWAPN depth 17 through 235 and every valid EXCHANGE pair is assembled using the EIP's inverse encoding, then checked for the exact affected values and minimum required stack depth.
- Every reserved immediate is rejected; DUPN over a full 1024-entry stack fails. JUMPDEST immediates remain valid targets while JUMPDEST bytes inside PUSH data stay masked.
- Real native Amsterdam transactions deploy and execute all three opcodes, then read back storage. A timestamp-boundary test fails execution before Amsterdam and succeeds at activation without enabling the private Glamsterdan gas fork.
- Restoring the two production files to `38ea9259` causes the published vectors to fail with stack underflow or wrong stack values. The files were restored before the final passing run. The local raw baseline log is `/tmp/glamsterdam-8024-baseline-failures.log`.

Passing checks:

```sh
cd pkg
go test -count=1 ./core/... ./engine/... ./node/... ./eiptest
go test -race -count=1 ./core/vm ./eiptest -run TestEIP8024
go vet ./core/vm ./eiptest
go build ./...
```

Full upstream fixture/cross-client and final gas-model acceptance remain part of the program release gates. This branch completes the native current-byte-encoding correction and its scoped consumer tests.
