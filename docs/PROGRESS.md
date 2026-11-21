# ETH2030 development progress

Updated November 22, 2026 at merged master `9a094830`. EL PRs 123/124/125 are merged. The production package and code-CI workflow equal qualified composition `d9201b0`; merge/source identity and running executable identity remain distinct. The original 33 Glamsterdam task IDs and R01–R13 are complete for the selected merged LIFO / signed Gloas development target.

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

The [current audit](audits/merged-glamsterdam-20261122/README.md) confirms a real code gap in legacy Engine builder extensions: unauthenticated registration/bids, synthetic local stake and dropped header signature. This does not establish an unsigned-envelope bypass in the qualified CL consensus route. It needs a separate contract/authentication or interface-retirement PR. A second code gap remains in required-interface startup failure propagation: binding errors can be logged without making `Start` fail and unwind. The CI-discovered ephemeral metrics binding race is fixed separately in this review.

Lodestar 10693 remains open upstream despite local/fork qualification. Larger-state multi-day operating/indexing capacity remains unmeasured. Public protocol/model approval is reserved. Finite fixture/format coverage remains explicit.

Later-fork independent Frame/Bogotá agreement, Dora production consumers, paymaster services and public binary/PQ/proof/native-rollup work retain separate gates. The paired Frame 66/165 and CL join 256 record is preserved at [its original source](verification/frame-8141-20261118-paymaster-history-final.json); its synchronized-EL boundary is not erased.

[Roadmap](ROADMAP.md) · [Next-stage plan](next-stage-work-plan.md) · [Current gaps and reproducible observations](audits/merged-glamsterdam-20261122/README.md)
