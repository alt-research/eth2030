# Fork-aware blob capacity, pricing and settlement

Branch `feat/glamsterdam-blob-schedule` is stacked on configuration-authority
PR 79 (`901ec063`) and composes the full-data/proof consumer from PR 76
(`bc2f5850`). Canonical Engine version/schema validation remains the separate
PR 78. These dependencies are explicit; none is represented as upstream merged.

## Normative and reproduced inputs

Read EIPs [7840](https://github.com/ethereum/EIPs/blob/3db38a3725105cb67b8804ac3e147967113bc3ca/EIPS/eip-7840.md),
[7892](https://github.com/ethereum/EIPs/blob/3db38a3725105cb67b8804ac3e147967113bc3ca/EIPS/eip-7892.md),
[7691](https://github.com/ethereum/EIPs/blob/3db38a3725105cb67b8804ac3e147967113bc3ca/EIPS/eip-7691.md)
and [7918](https://github.com/ethereum/EIPs/blob/3db38a3725105cb67b8804ac3e147967113bc3ca/EIPS/eip-7918.md)
in full, retaining the previously read 4844/7594 requirements. EIP-7594 introduces
the six-blob transaction bound at Osaka; it does not keep every block at six.
The independent reference is unmodified Geth v1.17.10, with explicitly frozen
Prague/BPO parameters and Osaka activation, not a native-to-SDK adapter that
silently drops those inputs.

The reproducible red-test commit is `0f1f5a55`, before the implementation.
Compiled regressions show contradictory schedule helpers, the wrong Prague
fraction, an unadopted 2^25 minimum/direct price floor, and reserve logic applied
before Osaka. Native ordinary execution returned BLOBBASEFEE=0; at excess
20,000,000 it reported receipt price 399 instead of the independent 54 and
deducted no blob fee. Underpriced and insufficient-maximum-balance transactions
were accepted. Logs are `/tmp/glamsterdam-blob-{schedule,accounting}-baseline.log`.

## Implementation and configuration

The runtime configuration now consumes EIP-7840 `blobSchedule` values. Missing
known entries use documented Cancun/Prague/BPO1/BPO2 defaults; explicit invalid
values fail rather than falling back. Values are positive, target does not exceed
max, and blob-gas multiplication fits uint64. Unknown schedule entries fail.

Current BPO inheritance is authoritative. Older owned genesis generators repeat
parameters under `osaka`, `amsterdam` and `bogota`: scheduled aliases are accepted
only if they match the inherited active parameters, then removed on resolution.
Unscheduled aliases have no runtime effect. This preserves compatible existing
inputs without silently allowing named forks to change blob parameters.

Official historical BPO1/BPO2 times for mainnet and Sepolia are pinned to
[mainnet](https://github.com/ethereum/go-ethereum/blob/v1.17.10/params/chainspecs/mainnet.json)
(blob `faa3a33a`) and
[Sepolia](https://github.com/ethereum/go-ethereum/blob/v1.17.10/params/chainspecs/sepolia.json)
(blob `94a6baf6`), live-rechecked against the tag. No new Glamsterdam activation
date is invented. Native-to-SDK conversion now carries the blob parameters and
BPO timestamps, with independent copies; its broader gas-model limitations are
not promoted into current-spec parity.

Node persists the resolved parameter sets. Every activated set, including a
superseded Prague set, is protected during restart. Genuinely future parameters
remain adjustable. Executed history with no resolved blob schedule is explicitly
unknown and rejected instead of assigned new historical defaults. Existing
files are preserved. Old experimental blob history that omitted fees or used
wrong prices is not made Ethereum-valid by changing its configuration: private
test chains need a fresh genesis; canonical history can use a separate verified
resync/replay. No automatic history rewrite or migration switch is introduced.

## Actual protocol consumers

- Builders and both header validators use the selected block capacity and the
  child's schedule for excess. Osaka gates EIP-7918, which changes accumulation,
  never directly floors current price. Checked arithmetic rejects overflow and
  invalid units; the unrelated experimental parent-target helper has an explicit
  legacy name and is outside canonical validation.
- Ordinary blob execution validates recipient/count/hash/fee obligations before
  mutation, reserves maximum blob affordability, burns actual blob fees and
  retains that fee on revert/exceptional halt. EVM BLOBBASEFEE, receipts, canonical
  Frame execution, pool head-fee policy, historical readback and payload contexts
  share the current-header price. The pre-Osaka historical transaction bound and
  Osaka's six-blob bound are distinguished from block capacity.
- RPC transaction/full-block formatting receives the actual fork price;
  feeHistory supplies current and forecast blob prices and capacity ratios.
  Its N+1 forecast uses the latest known fork, matching Geth's oracle policy;
  actual block construction uses the CL-provided child timestamp. Regular N+1
  base fee is computed instead of returned as an unavailable-block zero.
- Frame block value uses the built block's base fee, and send-frame asks the RPC
  for its active blob price rather than applying a Cancun fraction locally.

Standalone research blobpool/assembler helpers are not claimed as production
state-sync or sparse relay consumers. No runtime feature flag is added.

## Signed HTTP, independent formulas and restart

The actual Node test signs Geth type-3 transactions with genuine version-1 cell
sidecars. A producer and separate native receiver process configured Prague,
BPO1 and BPO2 transitions. Each transaction is within six blobs; their blocks
carry seven, ten and twenty-one. Every excess/price is compared with Geth's
independent eip4844 implementation. Sender debit, BLOBBASEFEE storage, HTTP
receipt price, feeHistory and fresh-node roots/balances agree. Over-capacity
headers have independently recomputed hashes and fail the intended maximum
guard, preserving the existing head. Changing a historical parameter is also
rejected without replacing authority. [Retained results](glamsterdam-blob-schedule-run-20261118.json):

| Child time | Blobs | Excess blob gas | Blob fee |
| --- | --- | --- | --- |
| 12 | 7 | 0 | 1 |
| 24 | 10 | 550502 | 245 |
| 36 | 21 | 987408 | 139 |

The legacy wrapper run hit the unchanged eight-second Engine budget under race
instrumentation while regenerating 21 cell-proof groups. This was not a data-race
report. Composing PR 76 and testing the current authenticated version-1 wrapper
uses its preserved proof path. The complete signed scenario passes normally
(4.28s) and under race (28.57s overall), without increasing production or test
timeouts. The earlier timeout remains recorded, not erased or called a pass.

The full Go suite and tagged Core/EIP suite pass, as do affected race, vet and
local build. Six compiling behavioral mutations fail: omit blob fee burn, clear
the opcode price context, restore the builder's six-blob block cap, select the
parent's parameters, bypass historical parameter authority or return a zero fee
forecast. Sources are restored. The four latter mutations are detected by the
actual signed HTTP/restart regression. Full-client/network capacity and release
gates remain separate. Baseline, HTTP, mutation and suite logs are retained locally under
`/tmp/glamsterdam-blob-schedule-*`. Final execution/state gas, eth/72 custody and
reconstruction, snap/2, full CL/multiclient devnets and release qualification
remain open.
