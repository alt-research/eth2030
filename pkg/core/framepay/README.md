# ETH2030 Frame paymaster v1

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
rtk proxy python3 pkg/core/framepay/contracts/generate.py --check
```

`runtime.hex` is 459 bytes with Keccak-256
`ca4aa4d8e001959081cc44528b5aeec6dbda2e5adfc0c5544f70da1e622c3f32`.
`InitV1` appends the owner's ABI word to the compiled creation code.

The pool grants the pay-frame trace exception only by complete runtime byte
match and still requires normal execution to approve payment. Availability is
balance minus all pending max-fee reservations and the entire pending withdrawal
amount, including a matured request that has not executed. Unknown or changed
code follows generic trace rules and the non-canonical paymaster cap.
