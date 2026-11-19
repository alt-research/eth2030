# EIP-8282 bytecode provenance

These are exact hex assets from [ethereum/sys-asm at
83f9801245ff56878a450b5625801101b9a225a1](https://github.com/ethereum/sys-asm/tree/83f9801245ff56878a450b5625801101b9a225a1),
the reference implementation linked by [EIP-8282 at the selected EIPs revision](https://github.com/ethereum/EIPs/blob/3db38a3725105cb67b8804ac3e147967113bc3ca/EIPS/eip-8282.md).

| Local asset | Upstream path | Runtime/init bytes |
| --- | --- | --- |
| deposit_runtime.hex | bytecode/builder_deposits/main.hex | 628 |
| deposit_init.hex | bytecode/builder_deposits/ctor.hex | 638 |
| exit_runtime.hex | bytecode/builder_exits/main.hex | 458 |
| exit_init.hex | bytecode/builder_exits/ctor.hex | 503 |

The native EVM executes both constructors in tests and checks their output,
account nonce and exit inhibitor. Runtime bytes also match the independent
Geth v1.17.10 constants. No bytecode is rewritten at activation. Network operators
must arrange deployment before the fork; owned test/devnet genesis allocations
are explicit inputs before state roots are calculated.
