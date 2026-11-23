# ETH2030 development progress

Updated November 22, 2026 at merged master `55f7fd84`. EL PRs123–130 are merged. Current production package bytes equal functional source `2c0df12f`; original `d9201b0` measurements remain attached to that source; merge/source identity and running executable identity remain distinct. The original 33 Glamsterdam task IDs and R01–R13 are complete for the selected merged LIFO / signed Gloas development target.

## Current merged acceptance

| Area | Retained result | Evidence |
| --- | --- | --- |
| Execution/RPC |219 fresh packages,107 official files/108 requests, six affected race suites, full vet/native build, byte-identical second Go1.26.11 executable. | [Post-merge results](reviews/glamsterdam-postmerge-20261121/RESULTS.json) |
| Snap/BAL recovery | Legal BAL expiry respects actual finalized slots; indexed snap recovery authenticates original checkpoint/head/markers/canonical ancestry; malformed/failed finalized reads remain errors. | PR 123/124; `core/chain/snap_history_retention_test.go`, `snap_indexed_restart_test.go`, `core/rawdb/finality_checked_test.go` under `pkg/` |
| Original datadirs | Main2/main3/full/snap conservative Docker-start bounds189.038/212.551/219.661/176.953 seconds, all under 300s, with frozen finalized commitments/eight independent proofs. | [Recovery record](reviews/glamsterdam-postmerge-20261121/RECOVERY.json) |
| Operating windows |400/400 steady and60/60 pressure slots, original latency budgets,512-participant independent finality/offline controls and270 account proofs. | [Results](reviews/glamsterdam-postmerge-20261121/RESULTS.json) |
| Final-source blob | Actual external-builder two-row blobslot59163;544 KZG proofs, authenticated parent SSZ/signatures/ordered commitments; useful getBlobs-disabled partial cells228/114 columns and observer32/16. | Final-source archive; same original successful transaction, original API404 preserved |
| Source reproduction |245 public files, complete manifest, fresh extraction and11 independent replay checks; original nine archives/failures unchanged. | [Public replay](reviews/glamsterdam-postmerge-20261121/PUBLIC_REPRODUCTION.json) |
| Earlier application/CL scope | Signed Uniswap/Safe/7702 sequences, canonical factory, builder payments/registry, progressive SSZ, native official fixtures and independent cold sync retain exact sources. | [Task reconciliation](reviews/glamsterdam-final-campaign/TASK_RECONCILIATION.json) |

These results are retained source-specific measurements, not a new live campaign on the merge metadata. Native EF results remain distinct from Geth-backed runners. The initial failed preparations, over-budget runs and corrected harness inputs remain visible.

## Remaining work after merge

The original audit's GAP-01 and GAP-06 have separate source corrections merged in master `55f7fd84`: [127](https://github.com/alt-research/eth2030/pull/127) removes the unsupported local Builder registry, synthetic stake and signature-dropping header route; [128](https://github.com/alt-research/eth2030/pull/128) returns required bind failures, unwinds owners and requires a fresh instance for retry. The standard CL registry, signed SSZ bid and actual finalized-deposit/collateral controls remain authoritative. [130](https://github.com/alt-research/eth2030/pull/130) also corrects restored snap ancestry-cursor eligibility under moving CL targets; its finite actual non-optimistic prefix has independent signed finality.

[129](https://github.com/alt-research/eth2030/pull/129) replaces permanent value residency with a16MiB accounted cache and streams WAL replay. A655.36MB value fixture reduces measured peak RSS from939,364 to52,440KiB; a412.20MB WAL fixture from898,580 to12,784KiB. Both original and corrected runs meet their declared budgets; cold reads cost more. The corrected private million-file clone restarts in62.504/10.719seconds, preserves original finalized commitments and passes four independent MPT proofs. These are finite measurements with OS cache retained, not public full-database reproduction or multi-day acceptance.

[Current delivery and remaining gates](plans/glamsterdam/REMAINING_CODE_CLOSURE_20261122.md) preserves all original failures. Actual48-hour operation, growing key/header metadata, upstream Lodestar integration and reserved public protocol/model approval remain open. Native and Geth-backed corpus scope stays finite.

Later-fork independent Frame/Bogotá agreement, Dora production consumers, paymaster services and public binary/PQ/proof/native-rollup work retain separate gates. The paired Frame 66/165 and CL join 256 record is preserved at [its original source](verification/frame-8141-20261118-paymaster-history-final.json); its synchronized-EL boundary is not erased.

[Roadmap](ROADMAP.md) · [Next-stage plan](next-stage-work-plan.md) · [Current gaps and reproducible observations](audits/merged-glamsterdam-20261122/README.md)

## Latest runtime acceptance failure

Rounds03 and04 failed; the latter observed658 samples /5h30m before a CL syncing timeout, with noOOM and clean daemon exits. Independent BLS finality/ancestry authenticates the full observed73639–80256 span, but the runtime duration remains failed. Native zero-transaction block77623 only finished after shutdown; its actual blocking dependency is being investigated with a read-only goroutine diagnostic. New CL protocol/lifecycle fixes pass their scoped tests and complete network simulations; they do not close the native stall or48-hour gate. See [results PR132](https://github.com/alt-research/eth2030/pull/132) and [diagnostic PR133](https://github.com/alt-research/eth2030/pull/133).
