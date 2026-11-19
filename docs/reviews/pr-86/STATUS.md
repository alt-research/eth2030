# PR 86 tracking cleanup status

Original review head: `7d79bbc5cd610cacca29d72285c664a6e5d1b19c`. Merge base for preserved records: `a1008f472ca63950cfb22515ce8b795d21e913bd`. Target master: `98716284b049933888bcd68d64c1957004d759b2`. Shared tracking policy: `42a9ceb86dea2fe4a74f01fd5e2360d6f21af0e7`. The sections below preserve this branch snapshot, including any inherited feature records; their historical validation is not a new result for this cleanup. The exact original diff, including removed text, is in [legacy-tracking.patch](legacy-tracking.patch).

Cleanup validation passed on the merged source tree: four-package core/config, core/vm, core/execution and txpool tests, and `go build ./...` from `pkg/`. Source changes outside the reviewed conflict resolution match Git's automatic three-way merge. Publication targets the existing review branch without force; live GitHub verification follows publication. Draft status and dependency order are unchanged.

## Preserved feature tracking

## Stateful simulation review submitted (2026-11-19)

[PR 86](https://github.com/alt-research/eth2030/pull/86) publishes the independently qualified simulator. All selected full/tagged/race/vet/build, complete SDK commitment controls and fifteen mutations pass; final API/historical/process/resource/release gates remain explicit in the evidence. Nested tracing continues separately.

## Qualified stateful simulation implementation (2026-11-19)

Source `6daa6678` and composition `206e14ac` qualify actual stateful HTTP simulation against two complete unmodified SDK block reconstructions, fifteen detected compiling mutations and final full 212/tagged 27/affected race 48/vet/build. Five additional race packages plus vet/build qualify the PR 85 test-only follow-up. [Evidence](../../verification/glamsterdam-simulate-v1-20261119.md). Publication follows separately; nested tracing, broader fixture/process/resource and final release gates remain open.

## Stateful simulator consumer checkpoint (2026-11-19)

Actual HTTP now qualifies sequential calls/blocks, strict nonce/fee rules, uncapped versus transactional gas, default reservations, global budget, precompile relocation/warming, full transactions, gap blocks, current/side-branch BLOCKHASH and exact input errors. Two independent unmodified SDK executions match state/receipt/transaction/withdrawal/request/BAL commitments, complete header hashes, paid/peak gas and exact RLP size, including a pre-state override with refund/floor interaction. Virtual transfer logs roll back and stay outside receipts; native Amsterdam logs are not duplicated, and unknown parent slots remain unknown. Adversarial system work times out without canonical mutation. A reproduced uint256 override overflow/panic is repaired on request-owned simulation state. Fifteen compiling guard mutations are detected. Final full 212/tagged 27/affected race 48/vet/build pass after restoring all fifteen mutations. Publication and structured evidence follow separately.

## Stateful simulation implementation (2026-11-19)

Separate branch `feat/glamsterdam-simulate-v1` implements the selected execution-apis 68d76a5b method on the explicitly selected current merged LIFO baseline. Actual sequential calls/blocks must retain isolated state and complete system, fee, receipt, BAL and header consumers. Validation-enabled and disabled modes differ in nonce/fee checks and execution/reservoir policy; default uncapped calls reserve the smaller remaining execution/state dimension. Header gaps are real simulated empty blocks. No simulator conformance or release completion is claimed before actual HTTP and independent commitment evidence.

## Actual access discovery follow-up (2026-11-19)

PR 85 additionally checks the actual HTTP-discovered storage list and its exact 131253 paid gas. A computation needing more than 2^24 execution gas is rejected by access-list simulation under transaction rules. Its compiling policy mutation is detected, bringing the review to eleven detected mutations. Five affected race packages, vet/build and restored-source HTTP checks pass; full/tagged/48-package race evidence belongs to the preceding unchanged production source. The user explicitly selected current merged LIFO implementation while retaining final design review.

## Qualified RPC method and opcode consumers (2026-11-19)

Current source `28933241` passes full/tagged Go, affected race, vet/build and ten detected compiling mutations. Real HTTP verifies full call arguments, overrides, state/account estimates, creation, opcode traces, historical prefix/system replay, block traces and actual fork context. [Evidence](../../verification/glamsterdam-rpc-gas-20261119.md). Simulator, nested callTracer/two-dimensional trace fields, independent full-client/API fixtures, resource and final release gates remain open. This branch is separate from the stable EL integration until composed qualification.

## RPC method consumer implementation checkpoint (2026-11-19)

Real Node HTTP now passes intrinsic GAS observations, exact slot/new-account estimates, uncapped call versus transaction-cap behavior, creation return bytes, state/code overrides including full storage replacement, real traceCall steps and a signed two-transaction historical prefix replay with exact receipt/trace gas and canonical non-mutation. Strict full input parsing retains access lists, dynamic fees and authorizations. Affected Core/SDK/Node backend/RPC tests and local build pass. Full composed/mutation qualification, extended historical/system/application controls and simulationV1 remain pending; no API conformance claim is inferred.

## Method-specific gas consumer implementation (2026-11-19)

Separate branch `feat/glamsterdam-rpc-gas` builds on Gas draft 83. Current production call execution omits intrinsic costs and creation; estimation reuses that call path; historical tracing replays direct calls without the canonical transaction/fork/system transition. These consumers need repair before API qualification. Open execution-apis issue 977 has no comments or adopted cross-client decision; merged Geth 35845 supplies a concrete development reference, including its corrected non-strict simulation reservation. Final API and trace schema acceptance remain open.

## Current gas authorization and accounting qualification (2026-11-19)

The signed corpus now exercises ten authorization boundaries and observes LIFO/ancestor refill through GAS-dependent storage. Two actual HTTP blocks verify sender/miner settlement and advertised block value. Ordinary full Go tests, tagged Core/EIP tests and broad affected race/vet/build passed before the final added boundary controls. Historical Prague pricing controls are explicit; current-fork tests retain successful state changes, exact fee and two-dimensional header assertions. The old draft's claim that reservoirs reduce total fees is replaced by equal-price funded/unfunded controls. Ten compiling guard mutations are detected; the initial rollback mutation escaped an early-OOG fixture, so a later failure after successful authority writes was added and catches it. Final full/tagged/affected race/vet/build pass. The signed checkpoint is archived separately from the subsequent evidence commit. RPC modes, broader official/application/full-client and release qualification remain open; PR 83 stays draft.

## Canonical execution system transition qualification (2026-11-19)

Correct history address/8191 ring and shared actual-code pre-block callbacks now produce identical complete roots to unmodified SDK in deployed/missing/custom/revert scenarios. Native Node HTTP build/import/query/wrap/restart and exact runtime/genesis consumers pass. Seven owned profiles preserve existing allocations while preloading canonical code before roots. All 212 Go packages, 27 tagged Core/EIP packages, 11 affected race packages, vet/build and seven compiling detected mutations qualify this source. [Evidence](../../verification/glamsterdam-system-transitions-20261119.md). Gas draft composition, live independent-client/devnet/resource and final release gates remain open.


## Moving pivot proof binding composed regression (2026-11-19)

Source `c0df6062` composes PR 81's unchanged-account exact-proof repair with reviews 65–82. All 212 Go packages, 27 tagged Core/EIP packages, affected snap/scheduler/Node race, vet and build pass. Seven encrypted Node pivot scenarios now cover unchanged pending storage and its damaged proof. The Gas draft remains separate; final cross-client/resource/release gates stay open. [Evidence](../../verification/glamsterdam-snap2-binding-integration-20261119.md).

## Unchanged storage account pivot proof repair (2026-11-19)

Reproduced `storage account not verified` over real encrypted Node sync when an unchanged account still has partial storage after a pivot move. The verified storage consumer now obtains an exact account inclusion proof against the fresh pivot before trusting its storage root. Seven encrypted Node scenarios pass, including damaged-proof rejection for the unchanged account; two added compiling mutations are detected. Full Go, tagged Core/EIP, affected race/vet and build pass. This closes the reproduced cache-binding gap without waiving root authentication. Independent-client/resource and final release gates remain open.


## Snap version 2 socket lifecycle follow up (2026-11-19)

Explicitly stop the test-owned P2P servers before Node/database restart and on every scenario exit. Five encrypted Node cases pass again under race, with affected vet and build. This makes the test resource ownership match the manually started component setup; production protocol behavior is unchanged.

## snap version 2 state catch up (2026-11-19)

The production node negotiates snap/2 and snap/1, preserving range/code schemas and rejecting removed trie-node messages on version 2. Raw positional BAL responses distinguish missing 0x80 from available c0; the actual sender and receiver use the custom codec. Header commitments and canonical BAL structure are verified before staged state rebasing.

CL-selected moving pivots reuse partial account/storage progress. Reorg recovery restores old-only entries through exact inclusion/absence proofs and applies ordered new-fork diffs; missing orphan BALs discard partial state and restart bulk download. Final roots and CL authority gate adoption. Five real encrypted Node scenarios and an unmodified Geth SDK packet consumer qualify cursor preservation, rollback/deletion, restart and invalid-proof rejection. Seven compiling guard mutations are detected. Full Go, tagged Core/EIP, affected race/vet and build pass. [Evidence](../../verification/glamsterdam-snap2-20261119.md). Full independent-client cold sync, mainnet resource limits, peer reliability policy, eth/72, final gas and release gates remain open.

## EIP-8282 native builder requests (2026-11-18)

Canonical contracts and mandatory Amsterdam type-3/type-4 extraction now compose with the existing snapshot and Engine commitment path. Explicit owned genesis allocation precedes roots; missing production predeploys invalidate blocks. Actual HTTP and separate real-BLS Lighthouse component consumers qualify request bytes and registration/exit gates. Full Go, tagged Core/EIP, targeted race/vet/build and failing guard mutations pass. See [verification](../../verification/glamsterdam-builder-requests-20261118.md). Final gas, live full-client/devnet and release gates remain open.

## Negotiated eth/69-71 and historical consumers (2026-11-18)

Branch `feat/glamsterdam-eth71` registers eth/68/69/70/71 on the production node. Implements seven-field range status without TD for 69+, version-correct bloom-free receipts, nonblocking queued range updates and canonical raw-list BAL wire encoding. Independent golden bytes expose the old double-wrapped BAL response.

Correlated historical fetchers paginate receipts, bound decoding by requested block transaction/gas limits, verify the complete receipt root and rebuild persistent metadata only for validated blocks. BAL replies are matched to peer/request/hash, checked against headers before any restore, and reject tamper/reordering/replay/unsolicited data. Unavailable markers preserve body availability and are not trusted as arbitrary BALs.

Independent Geth RLPx/codec tests negotiate all four versions over real encrypted sockets and check status plus actual receipt/BAL formats. Real-chain eiptests validate restored receipt metadata and tampered BAL preservation; a disabled consumer hash-check mutation fails the intended tests. Affected eth/P2P/chain/node/eiptest suites, targeted race, full build and vet passed. Full independent-client/devnet, eth/72/snap/2 and complete sync/recovery qualification remain separate program gates. [Evidence](../../verification/glamsterdam-eth71-20261118.md).

## Sparse relay composed regression (2026-11-19)

At source `20b1c1f`, submitted feature reviews 65–82 compose: all 212 Go packages, 27 tagged Core/EIP packages, 97 affected race packages, vet and build pass. Real eth/72 and snap/2 consumers pass alongside prior Engine/blob/configuration/request paths. [Evidence](../../verification/glamsterdam-eth72-integration-20261119.md). Final gas, complete CL/multiclient/resource and release qualification remain open.

## Sparse blob relay qualification (2026-11-19)

[PR 82](https://github.com/alt-research/eth2030/pull/82) connects production eth/72 flat correlated cell exchange, elided signed metadata, authenticated bounded staging, provider/sampler custody requests, 64-cell reconstruction and full-data-only pool promotion. Real encrypted native-pool peers qualify the two-provider threshold, random extra, custody expansion and eager mode. Actual Node HTTP/Engine/P2P/restart consumers and independent SDK packet consumers pass. All 212 Go packages, 27 tagged Core/EIP packages, 36 affected race packages, vet/build and eight failing behavioral mutations qualify this source. [Evidence](../../verification/glamsterdam-eth72-20261119.md). Full independent-client gossip, live CL custody reconnect, resource/fairness/WAL campaigns, selected gas and final release gates remain open.


## Snap version 2 composed regression (2026-11-19)

At source `13969eaf`, feature reviews 65–81 compose successfully: all 211 Go packages, 27 tagged Core/EIP packages, affected race/vet and build pass, including five encrypted Node snap/2 scenarios. Repeated identical tracker sections are removed. [Evidence](../../verification/glamsterdam-snap2-integration-20261119.md). Final gas, eth/72 and complete CL/multiclient/release qualification remain open.

## Qualified blob schedule and settlement (2026-11-18)

The current version-1 signed HTTP scenario passes at 7/10/21 blobs through producer, independent receiver, receipts/RPC and fresh-node replay. Canonical per-fork capacity, pricing, fee burn/affordability, opcode context and persisted parameter authority agree with independent Geth formulas. Six behavioral mutations fail; full Go/tagged, affected race/vet/build pass after composing PR 76 and PR 79. The earlier legacy-wrapper race timeout is retained as failed evidence, not erased. See [verification](../../verification/glamsterdam-blob-schedule-20261118.md). Final gas, sparse relay/state sync, CL/devnet and release gates remain open.

## Ordinary blob proof validation and Engine publication (2026-11-18)

Reproduced and repaired invalid ordinary cell-proof admission through real HTTP. Preserve wrapper versions and pooled sidecars, authenticate KZG data, wait for complete pool data, derive genuine legacy proofs and expose canonical BlobAndProofV1 DATA. Shared payload collectors no longer fabricate zero proofs or publish missing data. Journal insert/rotation and fresh-node replay preserve authenticated sidecars. Frame checks remain qualified. Affected suites, targeted race/vet and build pass; four controlled mutations fail as intended. See [verification](../../verification/glamsterdam-blob-validation-20261118.md). Complete eth/72 sparse relay and release gates remain open.

## Blob schedule implementation in progress (2026-11-18)

Fork-aware block capacity, pricing, ordinary fee settlement, opcode/receipt/RPC consumers and metadata history guards are implemented. Signed HTTP producer/receiver controls pass at 7/10/21 blobs. The full module and tagged suites pass; the 21-blob race run hits the unchanged Engine timeout in the legacy proof-generation route. Compose the submitted PR 76 proof path and qualify current version-1 wrappers before completion. Final gas/release gates remain open.

## Persisted configuration recovery (2026-11-18)

Actual Node startup now checks persisted runtime configuration before history replay, rejects active or immutable changes and retains compatible future updates. Runtime overrides do not mutate network globals. Real signed HTTP/restart evidence preserves head/state/config bytes on rejection and verifies successful future updates; missing/corrupt authority is explicit. Full Go, affected race/vet/build and four failing behavioral mutations pass. See [verification](../../verification/glamsterdam-config-recovery-20261118.md). Final gas, blob schedules, unknown legacy migration and release gates remain open.

## Canonical schedule and Engine wire boundaries (2026-11-18)

Reject incoherent canonical genesis schedules in actual startup, add Osaka to fork identity and fix asymmetric activated-fork comparisons. Production Engine versions now enforce their timestamp windows; Osaka getPayloadV5 has its own correct shape. Actual HTTP tests cover required fields, DATA/QUANTITY bounds, zero values, malformed BAL, capabilities and forkchoice adoption before attribute errors. Full Go/tagged tests, affected race/vet/build and four failing behavioral mutations qualify this scope. See [verification](../../verification/glamsterdam-fork-schema-20261118.md). Persisted compatibility and broader program/release gates remain open.

## Builder-request feature composition (2026-11-18)

PR 77 composes with the previous feature set. Isolated blob readiness, factory non-injection and memory slot tests now use an explicit owned genesis with mandatory builder contracts. Existing feature obligations are preserved. At source `d97be4c3`, all 211 tested Go packages, tagged Core/EIP, affected vet and local build pass. Final gas and independent live release gates remain open.

## Glamsterdam signed candidate-gas integration checkpoint (2026-11-18)

The isolated feature integration passes the full Go suite, including the new differential command. The actual 44-case signed corpus preserves 19/22 Amsterdam candidate mismatches and separately classifies Osaka transition-versus-outer-validator checks; it does not claim release parity. See [integrated observations](../../audits/glamsterdam-gas-2026-11-18/INTEGRATED.md). All remaining gas/CL/network/release gates remain open.

## Glamsterdam independent gas corpus (2026-11-18)

Added an offline signed-transaction differential command with 22 scenarios across Osaka controls and a pinned Amsterdam reference (44 comparisons). Reports preserve complete inputs, gas dimensions, validity versus included failure, logs and final state. The independent context explicitly includes CPSB; a targeted mutation proves its sanity check detects missing state creation pricing. Scoped tests/race/vet and local build pass. This is mismatch evidence, not release parity; see [verification](../../verification/glamsterdam-gas-corpus-20261118.md).

## Glamsterdam slot commitments across Engine backends (2026-11-18)

Preserved explicit zero slots through production HTTP Engine build/import, and supplied the missing slots to memory Engine construction/import. Independent go-ethereum header hashes now agree with actual payloads, SLOTNUM execution, public block RPC and committed historical BAL bytes. Slot zero, nondefault and uint64 maximum values are covered. See [verification](../../verification/glamsterdam-slot-commitment-20261118.md); full independent-client/schema/CL and final gas gates remain open.

## Historical empty BAL cache recovery (2026-11-18)

The bounded native Engine backend now regenerates the canonical empty BAL only when its header commitment matches, even without a cached BAL object. Both historical body APIs retain the execution body; nonempty/unknown commitments stay unavailable. This extends PR 66's slot-based retention qualification.

## Glamsterdam BAL availability window (2026-11-18)

Branch `feat/glamsterdam-bal-retention` updates EIP-7928 availability to 14,299 epochs. The bounded Engine backend uses actual header slots when available, with a conservative block-number fallback for legacy headers; both ByHash and ByRange preserve the body when BAL is unavailable. The production chain backend already persists validated canonical BALs on mainline and retains them beyond the minimum window.

Tests exercise the prior-window gap, exact cutoff, outside cutoff and missed slots through both Engine body APIs. A targeted mutation restoring 3,533 epochs fails the intended availability assertions. Affected storage/Engine/node/eiptest suites, build and targeted vet passed. This closes the obsolete-window subtask of GL-040; physical pruning, complete independent CL/recovery and full network qualification remain separate gates. [Evidence](../../verification/glamsterdam-bal-retention-20261118.md).

## Glamsterdam EIP-8024 bytecode and activation (2026-11-18)

Corrected native DUPN/SWAPN/EXCHANGE immediate permutations and reserved ranges, and activated the instructions at canonical Amsterdam. Published bytecode vectors, all valid depths/pairs, stack and jump boundaries, and native transactions/fork transitions pass. Affected suites, targeted race/vet and the local build pass. See [verification evidence](../../verification/glamsterdam-stack-opcodes-20261118.md). Independent-client and full release gates remain open.

## Glamsterdam EIP-8246 execution and BAL (2026-11-18)

Implemented balance-preserving destruction, transaction-end account clearing, journaled native creations and rollback, correct CREATE2 creator nonce/prechecks, and final-state BAL reconciliation. Native and canonical Frame tests cover late credits, CREATE2 reuse, existing contracts, fork boundaries and four state backends. Affected suites, targeted race/vet and the local build pass. See [verification evidence](../../verification/glamsterdam-selfdestruct-20261118.md). Independent-client and final gas qualification remain open.

## Glamsterdam protocol ETH transfer logs (2026-11-18)

Branch `feat/glamsterdam-transfer-logs` activates 7708 at canonical Amsterdam independently of the private gas timestamp and removes extra fee-burn and self-beneficiary burn events. Specified transfers keep the ERC20-compatible system event; same-address/zero-value transfers are suppressed. Existing CREATE/CREATE2 endowment emission is retained and verified before constructor logs, with failed-call rollback preserved.

Native regressions reproduce baseline extra fees, missing Amsterdam activation and incorrect creation log counts; after correction, fork-boundary, fees/self/zero-value, ordering and rollback cases pass. The new logs exposed the modern P2P decoder zero-bloom bug, fixed in shared review #70 and included here. Affected core/P2P/Engine/node/RPC/eiptest suites, full build and targeted vet passed. Full 8246 destruction/finalization and the remaining gas model are separate gates; use a fresh experimental chain to qualify corrected historical fork semantics.

## RPC access-list discovery (2026-11-18)

Branch `feat/glamsterdam-access-list` replaces the eth_createAccessList empty placeholder with production backend discovery. Independent state copies execute native transactions until the access list converges; returned gas is measured with the returned list, without selecting an empty list merely because it is cheaper. Reverted child touches are retained, lists/keys are deterministic, and malformed addresses/keys/quantities return invalid-params errors. Unsupported backends no longer report fictitious discovery.

The actual HTTP/node-backend regression first failed on the placeholder and now passes. Additional tests cover reverted inner calls, contract-creation accesses, historical versus current state, no canonical nonce/storage mutation and malformed input. RPC/node/eiptest suites, targeted race, full build and vet passed. This delivers GL-044 access discovery for current CallArgs; final gas-model/API semantic and broader authorization/blob argument qualification remain GL-020/025/045 work. [Evidence](../../verification/glamsterdam-access-list-20261118.md).

## Glamsterdam canonical deterministic factory (2026-11-18)

Branch `feat/glamsterdam-factory` uses the EIP-7997 factory at 0x4e59b44847b379578588920cA78FbF26c0B4956C with its exact 69-byte runtime and nonce 1 for explicit genesis/test state. Removed block-processor and both builder paths that injected a different contract at a fork. Private chains must include the canonical alloc or deploy it with an ordinary transaction before activation; no fork existence check or new switch is added.

Canonical address/runtime/nonce and actual CREATE2/value/return-byte/collision tests plus a no-fork-injection regression first failed for the intended reasons and now pass. Preserved legacy private BLS 0x12 without changing the canonical 0x0b..0x11 map. Core/Engine/node/eiptest suites and full build passed. This implements the canonical factory path; live network/genesis tooling qualification remains in the broader integration gates.

## Modern receipt bloom restoration (2026-11-18)

Branch `fix/eth-receipt-bloom` repairs a receipt-root interoperability defect: eth/69+ omits bloom on the wire, but the decoder OR-ed the newly constructed receipt's zero bloom instead of computing it from decoded logs. It now uses LogsBloom. A nonzero-log typed receipt roundtrip first failed on baseline and now preserves exact consensus bytes. p2p/ethproto, eth and eiptest suites plus full build passed. The fix is also required by the negotiated protocol and transfer-log feature reviews.

## Glamsterdam feature review batch (2026-11-18)

Published independent reviews from current master: [#65 contract/initcode limits](https://github.com/alt-research/eth2030/pull/65), [#66 BAL availability window](https://github.com/alt-research/eth2030/pull/66), [#67 canonical factory](https://github.com/alt-research/eth2030/pull/67), and [#68 native RPC access discovery](https://github.com/alt-research/eth2030/pull/68). Each branch retains its affected test/build evidence and explicit remaining qualification boundaries. The negotiated eth/69-71/historical-consumer feature is prepared on its own branch and awaiting review publication.

No review is marked merged or release-qualified here. The [implementation tracker](../../plans/glamsterdam/IMPLEMENTATION.md) distinguishes merged pre-existing header/history fixes from newly submitted work. Full gas/state, remaining protocols, CL/builder and combined independent-client/recovery gates remain open; the original dirty checkout is untouched.

## Glamsterdam implementation baseline (2026-11-18)

Started the 33-task completion plan on isolated branch `feat/glamsterdam-baseline`, based on freshly fetched `origin/master` at `38ea9259`. Imported the dated plan/audits without replacing newer mainline work. [Implementation tracker](../../plans/glamsterdam/IMPLEMENTATION.md).

Revalidation finds BAL/slot canonical headers, execution-slot propagation, canonical persisted Engine bodies and the old tagged expectation already repaired on mainline. Those defects must not be reimplemented from the stale audit branch. The latest targeted core/Engine/node/eiptest suite is green; source/spec decisions and remaining size/factory/retention/RPC/gas/network work remain under qualification. Large functional changes will use separate branches and PRs.

## Current merged gas development baseline (2026-11-19)

Isolated `feat/glamsterdam-state-gas` targets the fixed 3db38a3 LIFO model. The expanded native transition implementation now matches 30 signed Amsterdam cases, including 64 KiB code, zero-base-fee pricing/GASPRICE, four delegation call families and a wide gas request. The observed zero-base-fee overcharge is fixed against EIP-1559 and independent execution. Actual Node HTTP pool/build/import/restart also qualifies a 110M-total-gas deployment plus a second transaction, and rejects a correctly rehashed header using the receipt payment sum. Native frame metering, current intrinsic/runtime separation, authorizations, slot refill and creation charging are being composed. This is an implementation checkpoint: outer block/pool/RPC consumers, larger 7702/application vectors and release qualification remain incomplete; existing old-price fixture failures are retained for semantic repair. Unmerged alternatives remain separate. The older 49beda9 read-only EIP checkout is not used as numeric authority.


