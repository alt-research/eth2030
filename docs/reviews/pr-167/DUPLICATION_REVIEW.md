# PR #167 conflict and duplicate-work review

## Compared revisions and result

Reviewed upstream `98d4317784b755ab75eefbec1ae7d8dca813ae96` and integrated head `dca42829c9bc059244d10efc0a835b220944a6aa`. Fetching both remotes confirmed that master is an ancestor: seven PR-exclusive commits, zero missing upstream commits, and no textual merge conflicts. All 15 hosted checks passed on that integrated head; branch protection requires one approval.

The review uses the complete local diff: **117 files**, not the first 100 files returned by one GitHub PR-files response. Older source/audit records are historical evidence, not a second current implementation.

## Findings and disposition

| Finding | Evidence | Disposition |
| --- | --- | --- |
| The development plan can cause implementation to be repeated. | `docs/plans/hegota-development-plan-2026-11.md` still describes the pre-integration processor, toolchain and a broad planned backlog. Master already provides canonical Frames, persistent receipts, prefix admission, Bogotá wire processing, FCU selection and extensive qualification infrastructure. | Add a prominent current-baseline/reuse policy and the task-group reconciliation below. Existing code is the starting point; only demonstrated specification or qualification gaps become new implementation tasks. |
| Duplicate pool fork-context storage. | `Config.ChainConfig`/`HeadTime` mirrored `TxPool.frameConfig`/`frameTime`; `SetForkRules` wrote both. Node and simulator also set derived flags before calling the same setter again. | Keep one pool fork context. Use the existing `SetForkRules` plus full-header `Reset` in startup, simulator and tests; keep the block number alongside that context for the London prerequisite. |
| Test-only production reset API. | Every `ResetAtHead` caller was in tests. Native FCU/reorg paths already use `SetForkRules` and `Reset(state, header)`. The wrapper did not refresh header fees. | Remove the extra production API. Tests exercise the production methods and provide a real header, preserving fee updates, canonical Frame revalidation and legacy-profile eviction. |
| Obsolete/two-state IL wrapper and test-only digest in production. | The helper Engine now handles the three-state IL result directly; `focilCheckILSatisfaction` had no callers. `registrationDigest` was used only to construct rejected legacy PQ pseudo-signatures in tests. | Remove the unused boolean wrapper and move the digest fixture helper into `_test.go`; retain unavailable/error handling and forgery rejection. |
| Equivalent compatibility rewrite. | Master already rejects retroactive activation in either local or remote schedules. The PR replaced its name lookup with positional indexing, without additional compatibility behavior. | Restore master's algorithm; retain only the separately necessary `ExperimentalLegacyFrames` compatibility check and regression coverage. |
| BLS overlap needs accurate attribution. | Master already supplies canonical gnark decoding, hash-to-curve and pairing primitives. In the final PR, the fixed-array verification functions and `PureGoBLSBackend` delegate to the one consensus backend; they are compatibility entry points, not independent signature-verification oracles. | Reuse the shared primitives. Retain the PR's distinct strict aggregation/secret checks, independent blst vector and fail-closed backend selection; do not describe all canonical cryptography as newly implemented by this PR. |
| The small historical baseline overlaps test infrastructure but has a distinct contract. | It calls the existing native runner and direct upstream Geth, enforces checksums/exact counts/zero skips, and compares root, logs and gas for 12 older-fork cases. Master runs broad official Osaka/Amsterdam corpora. | Keep the manifest/differential layer; do not create another processor or relabel 12 cases as full-fork conformance. |

## Exact EIP-8141 source comparison

Master's native implementation declares EIP-8141 pin `dc5d5fa688b7fe87b95a907fcc128b4fde2f4d2f`. This PR's research pin is `ade9e9cab7e148bba4cd293f916a4a189cd7a8aa`. The EIP file at those two repository revisions is **byte-identical**, SHA-256:

`1c0fe44b0829f587a20c2e7575b55498f579a17b92d683bb73be94395ea16639`

The dependent gas specifications are byte-identical at these pins as well:

| EIP | SHA-256 at both pins |
| --- | --- |
| 8037 | `37abe522c5dee68a47a27db6f7b88a71da5b66c8a72ebc601c467940b577b715` |
| 8038 | `ad09e7c9a7c78855b00e793109056e6d30022bf4bbf0e237e757226de0e544e2` |

The older pin was fetched read-only from `https://raw.githubusercontent.com/ethereum/EIPs/dc5d5fa688b7fe87b95a907fcc128b4fde2f4d2f/EIPS/eip-8141.md` and compared with the locked local reference. Different repository SHAs alone do not establish a specification change. This identity does not establish correctness of every implementation rule or resolve the extension-composition decisions.

## Reconcile all 77 planning tasks before development

This is an implementation-reuse inventory, not a claim that all 77 tasks passed latest-spec or independent-client acceptance.

| Task IDs | Existing foundation / required next action |
| --- | --- |
| HGT-001, 004–009 (7) | A delivery remains the specification, admission and executable baseline. Preserve source provenance and update impact decisions as implementation is qualified. |
| HGT-002, 003, 010–016, 019 (10) | Reuse `core/types/frame_8141*.go`, `core/vm/frame_8141*.go`, `core/execution/frame_8141.go`, `core/chain/frame_8141_receipt_test.go` and `node/frame_8141_integration_test.go`. Inventory missing vectors and selected-spec differences; do not rebuild encoding, approval, context, gas, transition or persistence from scratch. |
| HGT-017, 018 (2) | Keyed-nonce/recent-root extensions retain their conditional specification gates. Canonical base Frames do not prove those extensions are complete. |
| HGT-020, 021, 023–029 (9) | Reuse `txpool/frame_8141.go`, signed prefix/reservation/replacement/expiry tests, native block builder, node HTTP/restart tests and shared simulation. Qualification still needs exact adopted-profile coverage, invalidation/propagation and independent consumers. |
| HGT-022 (1) | Depends on the conditional nonce/root extensions, not just base Frame pool support. Implement only admitted deltas. |
| HGT-030–037 (8) | Keep PQ/dependency/migration/assertion admission and ADR gates. Inventory existing `core/execution/pq_draft_test.go`, `core/vm/pq_draft_state.go` and crypto draft primitives; draft precompiles and metadata registries are not a complete 8504/8288 composition. |
| HGT-040 (1) | The scope/CL responsibility decision remains an explicit prerequisite, not a missing Go module. |
| HGT-041–045 (5) | Reuse `core/chain/bogota_payload.go`, `engine/bogota.go`, `node/backend/engine_bogota.go` and `node/bogota_deferred_integration_test.go`. Storage, deferred import, wire processing, list verdicts and restart have implementations; qualify against the selected current IL contract instead of adding a second backend/store. |
| HGT-046–048 (3) | Profile 2 claims, independent CL/gossip agreement and PQ resource qualification remain separately gated. Existing native paired controls do not close them. |
| HGT-050–061 (12) | Optional candidates stay conditional. Check exact pinned proposal text and existing VM/P2P primitives before admitting a delta; an EIP-named prototype is not conformance evidence. |
| HGT-070–076 (7) | CL work remains owned by the separately pinned CL repository. EL helper types and a missing sibling checkout are not evidence that CL code must be recreated in this repo. |
| HGT-080–087 (8) | Reuse the existing CI, official native signed/block corpora, Geth reference, consensus replay, independent snap2, application tests, fuzz/race, recovery and observability infrastructure. Add selected-Hegotá cases and honest verdicts. Legacy smoke/feature-aggregator weaknesses, independent Frame/FOCIL agreement and long-duration/resource gates remain real work. |
| HGT-090–093 (4) | Reuse existing runbooks and build/release machinery; update version-specific configuration and qualify the actual RC, image, upgrade, external-network and handoff gates. Host binary hashes are not a reproducible container/RC result. |

The row counts sum to 77. Relevant source boundaries are also described in `docs/frame-8141-execution.md` and `docs/plans/ROADMAP_REMAINING_20261124.md`: delivered native implementation and qualification infrastructure must not be confused with independent-client or full composed-fork acceptance.

## Work that must remain distinct

- Canonical `FrameTx8141` and the explicitly selected frozen `FrameTx` prototype have different formats and approval scopes. They must not be folded into one permissive decoder or activation gate.
- The actual node Engine backend and the separate helper backend are different runtime paths. A helper result does not substitute for node/HTTP tests.
- The old snapshot under `evidence/hegota-a/history/6a17479/` remains historical provenance. Current evidence is not counted twice by retaining it.
- The remaining legacy ePBS/PQ/IL fail-closed guards, strict differential manifest and exact source/admission checks provide behavior absent from master's historical helper APIs. Removing them merely because related modules exist would weaken the baseline.

## Cleanup verification

The cleanup preserves master's compatibility algorithm plus the distinct legacy-profile guard, removes the duplicate public pool context fields and test-only `ResetAtHead`, and initializes/revalidates pool context through the same setters and full-header reset used by native FCU. No second Frames, IL, BLS oracle or state-transition engine was introduced.

Local build and vet passed. The fresh full Go suite completed all 240 packages with 23,520 named test/subtest pass events, zero failures and 124 separately inventoried skips. Focused race checks for the changed configuration, pool, node, EIP integration, Engine and PQ fixture paths passed with 2,615 pass events, zero failures and six skips. The 12 required historical baseline cases passed with zero skips; source/admission checks, independent goldens and 12 Python checker regressions passed. The previous broader qualification remains available at immutable commit `dca42829`; it is not mislabeled as a fresh result for the cleanup. New-head hosted outcomes are tracked directly on PR #167.
