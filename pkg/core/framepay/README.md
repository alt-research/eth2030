# ETH2030 Frame paymaster v1

The exact upstream PR-12041 proposal is separately available as
`Runtime12041`/`Init12041`, selected with
`--experimental-frame-paymaster-12041`. Its CC0 `contracts/pr12041.asm`
reproduces the upstream 355-byte runtime and pinned hash in a native test.
It uses empty PAY data, any protocol-verified signature scheme at index 1,
slots 0/1/2/3 = signer/amount/unlock/new signer, a 24-hour delay, cancellation
and rotation. It cannot be selected together with v1. See the full
[source reconciliation](../../../docs/reviews/pr-170/CANONICAL_PAYMASTER.md).
The remainder of this file describes the distinct historical local v1.

This is ETH2030's experimental implementation of the EIP-8141 paymaster
architecture at EIPs revision `dc5d5fa6`. It is a frozen local candidate;
it does not assert that the upstream draft has standardized these bytecode bytes.

The owner, pending recipient, amount and ready timestamp occupy storage slots
0, 1, 2 and 3, matching the reference contract. Funds leave only through Frame
payment approval or an owner-requested withdrawal after 43200 seconds. The owner
has no setter and runtime code is identical across deployments.

`validateFrame()` requires a VERIFY frame with flags 1 and signature entry 1
using protocol-validated SECP256K1, empty `msg`, and resolved signer equal to the
stored owner. Protocol validation performs the same canonical low-s ECDSA
recovery before execution. Signature bytes stay outside frame calldata and are
elided by the canonical signature hash. P256, ARBITRARY, explicit-digest and
contract-signature authorization are rejected.

This replaces the reference's circular calldata witness and its non-compilable
inline `verbatim` bindings. Other ABI entrypoints remain `owner()`,
`WITHDRAWAL_DELAY()`, `pendingWithdrawalTo()`, `pendingWithdrawalAmount()`,
`pendingWithdrawalReadyAt()`, `requestWithdrawal(address,uint256)` and
`executeWithdrawal()`. Empty calldata accepts deposits.

The standard-JSON compiler request is reproducible with the recorded
`solc 0.8.38+commit.1835d247.Linux.g++`, optimizer runs 200 and Osaka EVM target:

```sh
python3 pkg/core/framepay/contracts/generate.py --check
```

`runtime.hex` is 459 bytes with Keccak-256
`ca4aa4d8e001959081cc44528b5aeec6dbda2e5adfc0c5544f70da1e622c3f32`.
`InitV1` appends the owner's ABI word to the compiled creation code.

The pool grants the pay-frame trace exception only by complete runtime byte
match and still requires normal execution to approve payment. Availability is
balance minus all pending max-fee reservations and the entire pending withdrawal
amount, including a matured request that has not executed. Unknown or changed
code follows generic trace rules and the non-canonical paymaster cap.

CI downloads the pinned Linux compiler from the official solc-bin distribution,
checks SHA-256 `65c3180a336eabb6bdc4bbd49cc405aeb233976964ae6c11fba2a6c8be8f3243`,
and reruns the artifact comparison. The generator requires only Python and the
recorded `solc` on PATH; it does not depend on workspace-specific command wrappers.
[Compiler distribution](https://binaries.soliditylang.org/linux-amd64/list.json)
