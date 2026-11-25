# PR #167 master integration audit

## Why the conflicts accumulated

The PR targets `alt-research/eth2030:master`, not the fork's master. Its merge base with upstream `98d43177` was `dbe10116b703f88a97a6c45f1443b489e8e3b4ce`. Before integration the PR had five exclusive commits, while upstream had **1,204** exclusive commits. Upstream changed **2,340 files**, with 285,645 insertions and 54,036 deletions, relative to that merge base. These are upstream divergence statistics, not the size of this PR.

The earlier `origin/master` comparison observed the fork at `5ffa711`, not canonical upstream master. It could not establish whether this PR merged with `alt-research/eth2030:master`. This integration explicitly fetches `upstream/master` and keeps the PR target fixed.

The two earlier website commits are already ancestors of upstream. They introduce no remaining website diff. The principal cause is the old shared baseline and overlapping implementation work, including APIs and placeholders that upstream has since retired. Textual conflict resolution alone would retain obsolete behavior and break the newer integration tests. The unmerged PR had no check runs; GitHub does not run ordinary pull-request workflows for a conflicted merge.

## All 19 conflict paths

| Paths | Resolution |
| --- | --- |
| `CHECK.md`, `STATUS.md` | Preserve both histories; add a separate integration result. |
| `pkg/core/config/chain_config_forks.go`, `chain_config_forks_test.go` | Keep all 24 upstream registered forks, SSZBlocks and upstream schedule behavior; retain symmetric retroactive activation and explicit legacy-profile compatibility checks. |
| `pkg/core/eftest/state_test_runner.go` | Keep upstream signed transactions, slot/header support and fork logic; retain gas/error evidence and historical fork-correct fee fields. |
| `pkg/core/execution/processor.go` | Keep canonical FrameTx8141 dispatch, tracing, original-state finalization and shared ConfigureBlockEVM. Gate only the distinct frozen legacy envelope; never gate canonical Frames on its opt-in. |
| `pkg/core/vm/interpreter.go`, `jump_table.go` | Preserve Amsterdam/Osaka gas and instructions. Isolate legacy opcodes behind their explicit profile; canonical Frame execution installs its own opcodes/context. |
| `pkg/crypto/bls/bls_aggregate.go`, `bls_integration_test.go` | Keep upstream canonical decoding/hash-to-curve; retain validated gnark aggregation/signing and secret-range checks. Preserve test cleanup and a real default verifier. |
| `pkg/engine/builder_api.go`, `builder_api_test.go` | Respect upstream deletion of local builder-authority extensions (`76427149`); do not resurrect the APIs or their capability entry. |
| `pkg/engine/server.go` | Preserve all upstream versioned methods and capability negotiation; keep the retired builder method absent. |
| `pkg/node/backend/engine_backend.go` | Preserve upstream import/validation, deferred Bogotá and FCU-owned canonical selection/fees. Remove obsolete pre-import short circuits and duplicate pool/oracle updates. |
| `pkg/node/backend/rpc_backend.go` | Preserve shared native simulation, request contexts, tracing, overrides and execution fork setup; do not restore the old separate EVM path. |
| `pkg/node/genesis_loader.go`, `genesis_loader_test.go` | Combine aliases/conflict rejection with upstream Osaka, SSZBlocks, blob schedules, PQ draft and binary overlay settings/tests. |
| `pkg/node/node.go` | Keep real snap/sync implementations and forkchoice-controlled imports. Retain canonical BLS selection and initialize pool context from the actual restored head. Do not restore retired stub writers. |
| `pkg/txpool/txpool.go` | Preserve upstream authentication, canonical Frame prefix validation, fee checks and revalidation. Keep old AA disabled by default and legacy-envelope opt-in/eviction separate from canonical Frames. |

## Semantic integration repairs

- Backend fork predicates remain monotonic, as expected by the upstream Engine dispatcher. Version-specific schema/epoch rejection belongs to that dispatcher: FCU V3 is reused after Prague/Osaka, and Bogotá V6/V5 processing needs Amsterdam execution rules to remain true. The old exclusive backend predicates broke both cases and have been removed.
- The native regression now proves that P2P announcements cannot publish an optimistic PoS head, then selects and reverts the head through the actual native FCU path. It supplies the explicit empty Bogotá CL-view fixture required by upstream; this is not a CL interoperability claim.
- Canonical signed Frames remain admitted across legacy-profile resets. Existing native execution, Engine/restart and application tests exercise the current canonical implementation. The EIP integration registry now also executes signed canonical Frames through a snapshot-capable StateDB, pool, builder and replay rejection. The two old integration tests are explicitly named legacy prototypes and opt in; AA pool fixtures also opt in instead of weakening production defaults.
- Snapshot negative tests exercise the real upstream backend with unknown roots; removed stubs are not recreated. Runtime dispatcher tests now recognize implemented Bogotá methods and require invalid parameters to fail, while unknown versions return method-not-found.
- Geth dependency identity is updated to v1.17.10 and the historical 12-case differential corpus is rerun. The dependency lock records old/new SHA-256 and the upstream revision. Normative EIP/CL source pins and the 72-proposal research snapshot remain unchanged.
- The VDF dependency panicked at iterations 1 and 2 because its optimization estimator evaluates a logarithm outside its domain. Those two cases now compute exact class-group squarings and the identity Wesolowski proof (zero quotient); the unchanged upstream verifier checks them, wrong difficulty and corrupted proofs. No digest substitute is accepted.
- The legacy filename-alias regression requires two distinct case-sensitive filenames and explicitly skips on a filesystem that aliases them. Linux continues to execute the regression. The P2P bind-failure fixture now reserves the same wildcard address family as the production listener. Cancellation tests synchronize request launch rather than assuming a finite gas budget lasts 50 milliseconds on every host.

## Evidence and limits

The previous `6a17479` evidence is retained under `evidence/hegota-a/history/6a17479/`. Current runtime fingerprints and verification results describe the integrated tree. Initial failing runs are retained in the integration evidence summary; successful reruns supersede them, without representing historical failures as passes.

Earlier A documents describe the old branch snapshot. In particular, their assertions that canonical Frames, native Bogotá APIs, snap state persistence or FCU head selection are unimplemented **do not describe integrated master**. The original prerequisite findings need requalification against the selected latest specification composition. PRE-3675/R16 is closed for its specific missing native FCU/head propagation issue by upstream plus the actual backend regression. The other historical prerequisite findings are not automatically declared conformant by this merge.

This integration does not establish a complete Hegotá fork release, resolve all D01–D09 composition decisions or prove official CL/devnet interoperability. The independent hosted workflows retain their original corpus, proof and interoperability assertions.

Local verification completed with 23,520 named test/subtest pass events, zero failures and 124 separately recorded skips across all 240 packages. Focused race and exact CI race selections passed (6,652 and 3,916 pass events respectively; these selections overlap). Build/vet, all three build-tag variants, 12 mandatory baseline cases, 12 source-checker regressions, 70 Python 3.14 control tests, independent goldens and the finalized consensus fixture replay passed. Hosted results are tracked on [PR #167](https://github.com/alt-research/eth2030/pull/167); local runs do not substitute for its Linux/native corpus/real-proof jobs.
