# ETH2030 development progress

Updated November 18, 2026 against mainline `911afb0c`, which merged [PR 61](https://github.com/alt-research/eth2030/pull/61).
This report records scoped implementation and acceptance evidence. The active
[roadmap](ROADMAP.md), [next-stage plan](next-stage-work-plan.md) and
[checklist](../CHECK.md) retain the remaining work.

## Latest merged implementation

| Area | Current result | Evidence |
| --- | --- | --- |
| Native Frame execution | Pinned seven-field envelope, scalar nonce, protocol signatures, approvals, independent execution/state gas, refills, atomic rollback and settlement. | [Execution record](frame-8141-execution.md), `pkg/core/execution/frame_8141.go` |
| Public admission/paymaster | Bounded prefix simulation, head/fee refresh, replacement exposure and a reproducible local paymaster v1 with delayed withdrawal reservations. | `pkg/txpool/frame_8141.go`, `pkg/core/framepay/README.md` |
| Blob admission | EIP-7594 pooled wrapper accepted through the actual public RPC; ordered KZG cell proofs validated. | `pkg/node/frame_blob_rpc_test.go` |
| Historical Engine data | Hex V1/V2 body responses and validated RLP BALs persist with blocks; canonical hash/range lookup and reopen preserve original bytes. | `pkg/node/frame_8141_integration_test.go`, `pkg/core/chain/block_access_list_test.go` |
| Heze checkpoint import | Trusted-anchor satisfaction avoids the unavailable pre-anchor parent; live list lookup, signatures, bids and custody checks remain enforced. | Lighthouse `a4ceb86e`, published `eth2030` branch |
| Build source identity | Cargo rebuilds version metadata after worktree, packed-ref and detached-HEAD changes. | Lighthouse `common/lighthouse_version/tests/worktree_version.rs` |
| Service codec candidate | Published Go client comparison plus minimal experimental-layout replay matches real finalized state and block roots, signatures, identities and commitments. | [Consensus replay](../tools/consensus-replay/README.md) |

## Live acceptance

The qualified EL `e0451092` / Lighthouse `a4ceb86e` pair finalized sponsored
FOCIL at slot 66 and a blob Frame at slot 165. A new CL database joined at slot
256 and read both original finalized envelopes. Receiver restart preserved
both workloads' receipts, storage and sponsor settlement. The CL observer used
a synchronized EL; independent EL sync is a separate gate.

The [retained record](verification/frame-8141-20261118-paymaster-history-final.json)
contains raw transactions, the signed list and bid, receipts, roots, commitments,
version/image pins, memory caps and resource removal. Failed controls are
retained separately in the [execution record](frame-8141-execution.md).

## Validation scope

- Full native Go tests, builds, affected vet and race checks passed.
- CI on the merged PR had seven successful jobs: full build/vet/test, pinned
  Geth-backed EF fixtures, integration, blst, go-eth-kzg, race and consensus/control
  regressions. EF results qualify that runner/backend and fixture pin.
- Real EVM and HTTP/FileDB tests cover authorization, state changes, nonce bounds,
  fee dimensions, journal rollback and historical readback. Targeted mutations
  detect missing owner/timelock/reservation, BAL persistence and wire-codec guards.
- Twelve Python control assertions and standalone consensus SSZ/block replay
  passed. Frozen paymaster artifacts rebuild with the checksum-verified compiler.
- Lighthouse Heze and Git-version regressions, cargo check, full lint-fix and
  release build passed before the paired control.
- The live enclave, containers/volumes and engine were removed. Seven owned
  runtime images and temporary binary contexts were removed afterwards.

## Remaining acceptance

| Work | Current boundary |
| --- | --- |
| Independent Frame/Bogotá client | Newer Geth experiments have a different keyed-nonce wire contract. A compatible reference pin or separately versioned protocol is required. |
| Dora and workload services | Codec replay is qualified; the actual production epoch/receipt consumer, retention and restart cutover are not. |
| CL reliability / Gloas builders | Skipped-slot checkpoint variants, redundant BN/remote signing, alternate-peer recovery, external payment, withheld envelopes and PTC need their own controls. |
| Track 3 | Independent full/snap late join, authenticated moving-pivot reuse/restart and bounded storage/memory need acceptance. |
| Public Track 4 / proofs | Public binary layout, witness charging, authenticated committee/key lifecycle and application-specific proof statements require independent qualification. |

The historical package/EIP labels remain available in the
[traceability matrix](EIP_SPEC_IMPL.md) and [site tracker](EIP_TRACKER.md).
They are not a current aggregate conformance or production-readiness score.
