# Project Status

## EIP and Consensus Reference Update (2026-11-25)

### Current Status: COMPLETE

Updated the two reference submodules to their upstream default branch (`master`) heads, as explicitly requested:

- `refs/EIPs`: `49beda901dcf3747ed4b91f853a333595fb2cbc4` → `ade9e9cab7e148bba4cd293f916a4a189cd7a8aa` (2026-11-25 UTC).
- `refs/consensus-specs`: `14e6ce5a5334910b0d0575d782e468305ad92373` → `d2021d2719235f0df4726a1d330502cb15aa8ea3` (2026-11-23 UTC).

Both submodule working trees were clean before and after the update. The new pins match the Hegotá research snapshot; all 70 published EIP file hashes in that manifest match the updated checkout. Updated the development plan's current-reference note while preserving historical research provenance. No upstream specification text or client implementation was edited.

Verification: each submodule HEAD equals its fetched `origin/master`; `cd pkg && env -u GOROOT go build ./...`; `git diff --check`.

## Hegotá Executable Development Plan (2026-11-25)

### Current Status: PLAN COMPLETE; IMPLEMENTATION TASKS NOT STARTED

Created [the complete development plan](docs/plans/hegota-development-plan-2026-11.md) with 77 dependency-ordered tasks and individual dispositions for all 72 reviewed proposals. It defines specification decisions, native-client integration, Frames/state/gas/receipts, public mempool/P2P/RPC, conditional PQ and migration work, FOCIL/CL integration, candidate admission, differential and cross-client tests, recovery, release gates, and maintenance. The 18–28 week initial scheduling range explicitly assumes two EL engineers, one CL/integration engineer, shared testing support, stable core scope and limited mature candidates; it must be re-estimated after the baseline audit and is not an official fork date.

Additional repository findings: native and geth-wrapper binaries follow distinct paths; Docker's native entrypoint differs from parts of the devnet README; existing Frames/FOCIL smoke checks and the feature-runner verdict aggregation are insufficient for conformance acceptance. The expected sibling checkouts are absent locally. The plan makes these prerequisite tasks and does not claim that implementation, official devnet interoperability, fixture conformance, or external deployment has been completed.

Verification: `cd pkg && env -u GOROOT go build ./...`; task-ID/dependency DAG validation; 72-EIP coverage and local-link checks; `git diff --check`. Only documentation and task tracking changed.

## Hegotá Macro Planning and EIP Discussion Review (2026-11-25)

### Current Status: RESEARCH AND PLANNING COMPLETE; IMPLEMENTATION NOT STARTED IN THIS TASK

Created a Chinese macro development plan, an individual review register for 72 proposals, and a source manifest with 77 EIP/context records. The research compares the current Hegotá Meta EIP with EF priorities, public proposal discussions, and dated ACD/breakout records through November 25. It records stale Meta entries, the Quick Slots withdrawal, continuing FOCIL scope review, evolving Frames/PQ interfaces, and inter-EIP specification inconsistencies. Public topic streams were fetched completely; the analysis reviews selected consequential replies and scope history, not every historical PR, video, private discussion, or implementation rule.

The implementation baseline is `dbe10116b703f88a97a6c45f1443b489e8e3b4ce`. Read-only reference submodules remain pinned to February 2026. The plan identifies old FrameTx/APPROVE semantics, the node backend's unsupported inclusion-list storage path, and the need for a versioned specification composition and conformance matrix before feature work.

Deliverables:

- [Macro development plan](docs/plans/hegota-macro-plan-2026-11.md)
- [Individual EIP/discussion register](docs/plans/hegota-eip-review-2026-11.md)
- [Pinned sources and discussion coverage](docs/plans/hegota-sources-2026-11.json)

Verification: `cd pkg && env -u GOROOT go build ./...`; `cd pkg && env -u GOROOT go test ./core/types ./core/eips ./core/vm ./txpool/frametx ./focil ./engine/... ./node/backend/...`; document/source coverage, snapshot hashes and local-link checks. These establish the local baseline only; current Hegotá conformance and cross-client devnet validation remain future implementation gates. No protocol code, reference submodules, or sibling repositories were changed.

## Website Strawmap Refresh (2026-11-05)

### Current Status: LOCAL UPDATE COMPLETE; FURTHER AUDIT AND DEPLOYMENT PENDING

Updated the site against the EF Strawmap drawing dated 2026-11-02, separating proposed Ethereum milestones from selected ETH2030 code evidence. Replaced unsupported completion claims with links to code, tests, specifications, and known limits. Reviewed selected EIPs and implementation paths at repository revision `14497b5`; this is not a full EIP-by-EIP conformance or cross-client interoperability audit. In particular, PQ registration verification uses a stub and ePBS bid validation has a zero-signature testing path.

Verification: `cd site && npm run build && node scripts/check-export.mjs`; desktop and mobile browser previews; `cd pkg && env -u GOROOT go test ./bal ./epbs ./focil ./core/eips ./crypto/pqc ./rollup/...` (run as two commands). Publishing remains pending; the deployment workflow still names `eth2030.com`, while GitHub Pages currently binds `eth2030.altllm.ai`.


## Website Domain Recovery (2026-11-05)

### Current Status: BLOCKED ON REPOSITORY ADMIN

GitHub Pages is configured for GitHub Actions but currently binds `eth2030.altllm.ai`, while the website and deployment workflow specify `eth2030.com`. The latter resolves to GitHub Pages and returns 404. A repository administrator must change the Pages custom domain to `eth2030.com` and verify DNS and HTTPS; this replaces the existing custom-domain binding. The privacy page now identifies GitHub Pages as the hosting provider.


## Full-Feature Devnet Bring-Up (2026-03-25)

### Current Status: IN PROGRESS

**Scope:** Build the local Lighthouse and ETH2030 images as needed, launch the Kurtosis `full-feature` devnet in the `eth2030-devnet` enclave, and verify block production plus basic EL/CL health.

**Planned Verification:**
- `cd ../lighthouse && docker build -t eth2030cl:local .`
- `cd pkg && docker build -t eth2030:local .`
- `cd pkg/devnet/kurtosis && ./scripts/cleanup.sh eth2030-devnet`
- `cd pkg/devnet/kurtosis && ./scripts/run-devnet.sh full-feature`
- `cast bn -r http://$(kurtosis port print eth2030-devnet el-1-geth-lighthouse rpc)`
- `kurtosis service logs eth2030-devnet cl-1-lighthouse-geth 2>&1 | grep -E "verified|ERROR" | tail -20`
- `kurtosis service logs eth2030-devnet el-1-geth-lighthouse 2>&1 | grep -E "WARN|ERROR" | tail -20`

## Inclusion List JSON Refactor (2026-03-21)

### Current Status: COMPLETE

**Scope:** Simplify the local inclusion-list JSON parsing helpers in `pkg/engine/inclusion_list.go` without changing the wire format or validation behavior.

**Completed Work:**
- Extract small helper functions for repeated hex-prefix stripping and byte-slice conversion.
- Reuse the helpers from the custom JSON marshal/unmarshal paths.
- Add focused parsing tests and rerun the relevant engine test suites.

**Verification:**
- `cd pkg && go test ./engine/...`

## Blob Bundle Refactor (2026-03-21)

### Current Status: COMPLETE

**Scope:** Deduplicate the cell-proof expansion logic shared by the V2 and V6 blob-bundle builders in `pkg/engine/backend.go` without changing output formats.

**Completed Work:**
- Extract shared blob sidecar collection and cell-proof expansion logic.
- Reuse the helper from the V2 and V6 blob-bundle builders.
- Verify the relevant engine test suites after the refactor.

**Verification:**
- `cd pkg && go test ./engine/...`

## Payload Retrieval Refactor (2026-03-21)

### Current Status: COMPLETE

**Scope:** Deduplicate the repeated payload retrieval paths in `pkg/engine/backend.go` without changing API-specific response contents or async eviction behavior.

**Completed Work:**
- Extract shared helpers for async payload waiting and mutex-protected payload lookup.
- Reuse the helpers from `GetPayloadByID`, `GetPayloadV4ByID`, `GetPayloadV5`, and `GetPayloadV6ByID`.
- Verify the relevant engine test suites after the refactor.

**Verification:**
- `cd pkg && go test ./engine/...`

## Inclusion List Selection Refactor (2026-03-21)

### Current Status: COMPLETE

**Scope:** Deduplicate inclusion-list transaction selection across engine backends and make oversized pending transactions skippable instead of aborting the whole selection pass.

**Completed Work:**
- Extract shared inclusion-list transaction selection into a helper in `pkg/core/eips/inclusion_list.go`.
- Use the helper from both `pkg/engine/backend.go` and `pkg/node/backend/engine_backend.go`.
- Add a focused test for oversized-transaction skipping.
- Verify the relevant Go test suites after the refactor.

**Verification:**
- `cd pkg && go test ./core/eips ./engine/... ./node/backend/...`

## Engine Review Follow-Up (2026-03-21)

### Current Status: COMPLETE

**Scope:** Recheck branch review findings, confirm whether `engine_getPayloadV5` JSON encoding is already fixed, and address the remaining safe bugs in inclusion-list handling.

**Completed Work:**
- Verified that the `engine_getPayloadV5` response encoding issue was already fixed on this branch by `fdd0820`.
- Reject oversized hex input in `flexibleUint64` instead of truncating it.
- Make the node backend return an explicit error for `ProcessInclusionList` until real storage/wiring exists.
- Add tests for the inclusion-list fixes and rerun the relevant engine/node test suites.

**Verification:**
- `cd pkg && go test ./engine/... ./node/backend/...`

## Engine Lock Wrapper Refactor (2026-03-19)

### Current Status: COMPLETE

**Scope:** Refactor `pkg/engine/backend.go` so accesses guarded by `stateMu`, `blocksMu`, `payloadMu`, and `ilMu` go through helper functions instead of open-coded lock/unlock pairs.

**Completed Work:**
- Add backend accessor methods that encapsulate `stateMu`, `blocksMu`, `payloadMu`, and `ilMu` reads/writes.
- Migrate existing call sites in `pkg/engine/backend.go` to the accessors without changing lock ordering or behavior.
- Follow up on the accessor refactor by removing remaining unsynchronized forkchoice field reads in `ForkchoiceUpdated`.
- Extend the accessor refactor into `pkg/engine/backend_bodies.go` to remove direct mutex usage there as well.
- Verify the `pkg/engine` build and tests after the refactor.

**Verification:**
- `cd pkg && go test ./engine/...`

## Go Test Repair (2026-03-19)

### Current Status: COMPLETE

**Scope:** Reproduce and fix failing or hanging `go test ./...` runs from the Go module root at `pkg/`.

**Resolved Issues:**
- `pkg/core/vm` JSON-vector and precompile fixture tests no longer fail when the optional external fixture corpus is absent from the checkout; they now skip cleanly on missing files.
- `pkg/engine` no longer deadlocks while storing processed blocks: `ProcessBlock` and `ProcessBlockV5` were calling `evictOldBlocks` with `blocksMu` already held, while the helper tried to lock `blocksMu` again.

**Verification:**
- `cd pkg && go test ./...`

## Devnet Testing (2026-03-16)

### Current Status: WORKING

**Devnet Configuration:** `full-feature-prysm.yaml` with PeerDAS/Fulu

**Test Results:**
- Slot: 367+ (continuing to advance)
- Finalized Epoch: 9+ (continuing to finalize)
- Block Processing Time: 15-200ms (normal)
- Data Availability Check: ~5 microseconds (instant)

### Issues Fixed

1. **engine_getPayloadV5 "build timed out" (CRITICAL)**
   - Root Cause: `buildMu` mutex held during `StateAtBlock` call which could take tens of seconds on cache miss
   - Death Spiral: state cache miss → state rebuild blocks buildMu → all builds timeout → chain stalls
   - Fix: Pre-fetch state BEFORE acquiring `buildMu`, use singleflight for deduplication
   - File: `pkg/node/backend.go`
   - Status: ✅ RESOLVED

2. **Race condition in computeBlobsV2**
   - Root Cause: `len(b.blobCache)` accessed after `blobCacheMu.RUnlock()`
   - Fix: Capture cache size before releasing lock
   - Status: ✅ RESOLVED

### Known Non-Critical Issues

1. **engine_getBlobsV2 "JSON-RPC response has no result"**
   - This is expected behavior in PeerDAS
   - Blobs are propagated via gossip (data column sidecars), not through Engine API
   - Prysm CL attempts to reconstruct data columns from EL as a fallback
   - EL doesn't have blobs for blocks built by other nodes
   - **Impact: NONE** - Chain continues to advance and finalize normally
   - These errors are just log noise from Prysm's gossip message handling

### Architecture Notes

**PeerDAS Data Flow:**
1. Block proposer (CL-2) broadcasts data column sidecars (128 columns per block)
2. Other CL nodes receive data columns via gossip
3. Data availability is verified from gossip-received columns
4. CL may attempt to reconstruct from EL as fallback (fails but non-fatal)
5. Block processing succeeds regardless

**State Cache Behavior:**
- `StateAtBlock` uses singleflight for deduplication
- Hot cache (`sc`) and permanent cache (`memSC`) provide state caching
- Re-execution needed when cache misses (can take tens of seconds)

---

## Formal Verification

- Formal Lean workspace is present under `formal/lean`.
- Initial EVM/VM model (`Lean2030/VM`) and richer `Lean2030/EVM` model are now linked in `Lean2030/Lean2030.lean`.
- EVM model now includes:
  - arithmetic and bitwise ops,
  - DUP/SWAP,
  - POP/JUMP/JUMPI/JUMPDEST,
  - compiler and theorem suites.
- Current status: still a semantic subset only (toy control-flow, no memory/tracing, no Go cross-checking).
- Added bytecode-offset execution module `formal/lean/Lean2030/EVM/Bytecode.lean` with `decodeAt`/`decodePush`/`runBytecode`.
- Added executable mismatch lemmas showing current op-index `run` diverges from byte-offset EVM behavior for `PUSH`-preceded jumps.
- Lean 4 test driver is now configured in `formal/lean/lakefile.lean` with `Lean2030/Tests.lean`.
- Added `.gitignore` entries for Lean outputs (`.lake`, `*.olean`, `*.ilean`) and linked the Lean VM strategy doc in README.
- Next major deliverable: full `compile`→`runBytecode` refinement theorem and stronger instruction/stack invariants.
