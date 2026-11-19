# Glamsterdam implementation tracking

Implementation begins from fetched `origin/master` at `38ea9259`, not the older audit checkout. The original checkout and unrelated dirty submodule/generated directories are preserved. The working branch is `feat/glamsterdam-baseline` in an isolated worktree. The source plan is imported from `6f60df69`; historical audit outputs remain dated observations, not current-mainline failures.

## Revalidation

- Mainline already includes native BAL and slot header encoding (`db9f384b` and related changes). The old diagnostic now reports different BAL commitments produce different RLP/hash and native Header has SlotNumber.
- Mainline already propagates header slot into execution and persists/serves canonical hex/RLP historical Engine bodies (`e0451092`). The old body-converter-only probe cannot prove omission in the production adapter and is not treated as such.
- Mainline fixed the stale tagged BAL expectation (`49030842`). Tagged regressions are rerun on the actual baseline.
- At the `38ea9259` baseline, remaining observed gaps included 3,533-epoch BAL retention, old 7954 size limits, the invented 0x12 factory path, old gas constants/refund/reservoir semantics, eth/68-only registration and the access-list placeholder. The feature reviews below repair the specified subsets. Independent header tests additionally reproduced zero-slot loss in the production Engine and all-slot omission in the memory Engine; PR 74 fixes those actual paths.
- GL-001 remains partial: [baseline inputs](implementation-baseline.json) freeze the audit scope and live-rechecked pending choices. Exact compatible fixtures/client artifacts and open protocol decisions are release inputs, not invented defaults.

## Delivery sequence

| Feature review | Scope | State |
| --- | --- | --- |
| [PR #64](https://github.com/alt-research/eth2030/pull/64) — baseline | Import the complete plan/evidence and reconcile newer mainline fixes | Submitted; release-input closure remains open |
| [PR #74](https://github.com/alt-research/eth2030/pull/74) — supplied slot/header qualification | Preserve zero slots and memory-backend slots; independent Geth header hash, SLOTNUM and HTTP/RPC/BAL body readback | Submitted; complete schema/asynchronous/CL qualification remains open |
| [PR #65](https://github.com/alt-research/eth2030/pull/65) — EIP-7954 | Published code/initcode bounds and transaction/pool boundaries | Submitted; final fork/model qualification remains open |
| [PR #67](https://github.com/alt-research/eth2030/pull/67) — EIP-7997 | Canonical runtime/address/nonce and no fork-time mutation | Submitted; actual generated genesis/control remains open |
| [PR #66](https://github.com/alt-research/eth2030/pull/66) — BAL retention | Published slot-based window, body preservation and conservative fallback | Submitted; complete cold-sync/physical-GC gates remain open |
| [PR #68](https://github.com/alt-research/eth2030/pull/68) — RPC discovery | Actual HTTP/backend discovery, historical/reverted/creation cases and non-mutation | Submitted; final gas/schema qualification remains open |
| [PR #75](https://github.com/alt-research/eth2030/pull/75) — independent gas corpus | 22 signed scenarios under Osaka/Amsterdam, explicit reference CPSB and retained per-field mismatches | Submitted; transaction-transition evidence only; final gas/application/block/API gates remain open |
| [Draft PR #83](https://github.com/alt-research/eth2030/pull/83) — current-text gas/state execution | Versioned merged LIFO transition and first high-state block consumers | In progress; full-suite/API/application/final choice gates remain open |
| [PR #69](https://github.com/alt-research/eth2030/pull/69) — eth/69–71 | Negotiated protocols, canonical wire, correlated historical consumers and encrypted-socket checks | Submitted; independent full-client cold-sync gates remain open |
| [PR #70](https://github.com/alt-research/eth2030/pull/70) — receipt bloom | Rebuild omitted wire bloom from actual nonzero logs | Submitted; included in branches 69/71 for standalone tests |
| [PR #71](https://github.com/alt-research/eth2030/pull/71) — transfer logs | Canonical Amsterdam activation, value transfer ordering and exclusions | Submitted; final gas/cross-client composition remains open |
| [PR #72](https://github.com/alt-research/eth2030/pull/72) — destruction and BAL | Balance-preserving EIP-8246 lifecycle, journaled creation/revert, final-state BAL and CREATE2 nonce/prechecks | Submitted; native and Frame behavioral tests pass; full fixture/gas qualification remains open |
| [PR #73](https://github.com/alt-research/eth2030/pull/73) — stack opcodes | Current EIP-8024 immediate encoding, canonical activation and every valid depth/pair | Submitted; native transactions and published vectors pass |
| [PR #81](https://github.com/alt-research/eth2030/pull/81) — snap/2 | Authenticated BAL catch-up, moving/rollback pivots, missing-orphan restart and actual encrypted Node/SDK consumers | Submitted; full independent-client/resource gates remain open |
| [PR #82](https://github.com/alt-research/eth2030/pull/82) — eth/72 | Actual sparse provider/sampler/custody relay, reconstruction, Engine APIs and encrypted Node recovery | Submitted; full independent-client/CL/resource gates remain open |
| [PR #84](https://github.com/alt-research/eth2030/pull/84) — system transitions | Canonical EIP-2935/4788 actual-code callbacks, explicit pre-root genesis assets, independent roots and native Node recovery | Submitted; qualified with reviews 65–82; Gas draft/live full-client gates separate |
| [PR #85](https://github.com/alt-research/eth2030/pull/85) — RPC methods and opcode replay | Actual call/estimate/access-list with current gas, full arguments/overrides, canonical historical/system/block replay | Submitted; full/tagged/48-package race/vet/build and eleven mutations pass; simulator/nested trace work separate |
| CL lifecycle | Actual CL/builder and recovery acceptance | Pending |

Each major change receives its own branch and PR; no unrelated changes are staged. Final task completion requires the acceptance evidence in DEVELOPMENT_PLAN.md. A submitted PR is not a merged or release-qualified feature.

## Verified branch evidence

Feature branches are based independently on mainline `38ea9259`; no feature was merged upstream by this implementation run. Each has its own CHECK/STATUS entry and verification document. Current reviews retain pre-fix failures, meaningful behavioral assertions and scoped builds. The eth/69-71 consumer hash-check mutation fails tamper/reordering cases as intended. A combined local integration run has passed for the scoped production changes; the full release matrix remains open. See the exact evidence below.

Upstream PRs #12581/#12582 and #5770/#5820 were rechecked live and remain unmerged at this pass. Their concrete heads are frozen in implementation-baseline.json. No open alternative is silently selected as mainnet consensus. Independent work continues while those final semantic choices remain unresolved.


## Combined development qualification

Integration branch: `integration/glamsterdam-completion`; tested commit `96a1994cc6c007b521517cd025a70627f5b6e3b2`. This is an isolated local merge of the submitted feature branches, not a merge into upstream master. All overlapping code resolutions preserve both features: Amsterdam activation of limits and transfer logs, initcode validation followed by BAL baseline collection, and destruction semantics without fee/self-burn logs.

Passing combined checks (after the slot and empty-BAL follow-ups):

- `cd pkg && go test -count=1 ./...`: 210 tested packages pass; 18 packages report no tests.
- `go test -tags integration -count=1 ./core/... ./eiptest`: 27 tested packages pass.
- `go build ./...`: passes.
- `go vet ./bal ./core/... ./eth/... ./p2p/... ./engine/... ./node/... ./rpc/... ./txpool/... ./eiptest`: passes.
- Initial aggregate targeted race checks cover BAL, VM/execution, eth/P2P, Engine, RPC, pool and HTTP consumers. After the last Engine follow-ups, `go test -race -count=1 ./engine/... ./node/backend ./eiptest` passes across 18 tested packages.

Logs are retained locally as `/tmp/glamsterdam-integrated-final-{tests,tagged,race,vet}.log`. The earlier broader race log is `/tmp/glamsterdam-integrated-race.log`. Exact review branches, published heads and tested source ancestors are recorded in [feature-reviews.json](feature-reviews.json). Later documentation-only review commits do not change the tested production sources.

## Program task status and next implementation

The 33-task plan remains the acceptance contract. A native fix, a passing development suite and an open PR do not close an unrun release gate.

| Tasks | Current evidence and unfinished work |
| --- | --- |
| GL-001–003 | Spec/decision baseline, source-backed failures and many precise regressions are recorded. Complete compatible fixture/client manifests and registry coverage are unfinished. |
| GL-010/011 | Canonical codec existed on mainline; PR 74 adds independent hashes and repairs actual missing slot routes. Complete asynchronous/custom builder, persistence/replay and schema permutations remain. |
| GL-012 | PR 78 qualifies canonical startup dependencies, Engine version windows, Osaka V5 shape and required-field/encoding bounds through actual HTTP. PR 79 adds actual pre-replay configuration authority and registered optional/BPO compatibility. PR 80 adds configured blob values and actual pricing/settlement/restart consumers. Unregistered private extensions, broader successor schemas and release inputs remain open. |
| GL-020–025 | Draft PR 83 implements the current merged LIFO transition, validation, runtime, authorization rollback and two-dimensional settlement; 42 clean signed matches and real HTTP/import/restart/fees/system consumers pass, with full/tagged/race/vet/build and ten detected mutations. GL-025 RPC/simulation/traces is under separate implementation. Broader fixture/application/performance/full-client and final U-01/U-02 release choices remain open. |
| GL-030 | Native size limits, pool/creation boundaries and current stack opcode semantics are implemented. Final gas/estimate and cross-client cases remain. |
| GL-031 | Canonical factory implementation and real CREATE2 behavior are implemented. Actual generated genesis/deployment control and final gas qualification remain. |
| GL-032/033 | Native protocol log and destruction/state/BAL corrections are implemented, including canonical Frame composition. Full upstream scenario matrix, selected gas and independent execution remain. |
| GL-034 | PR 77 implements canonical contracts, mandatory extraction, real HTTP production/import and identical real-BLS CL component consumers (Lighthouse PR 3). Full live deployment/transport/recovery and final gas remain open. |
| GL-040/041 | Canonical persisted bodies already existed; retention, empty BAL recovery, negotiated eth/69–71 and correlated bounded history consumers are implemented. Full independent-client historical/cold-sync and recovery gates remain. |
| GL-042/043 | PR 81 implements actual snap/2 moving-pivot/reorg/cursor/restart consumers with SDK encrypted wire checks. PR 82 implements actual sparse cell relay/provider/sampler/custody/reconstruction and Engine consumers. Full independent-client cold sync/gossip, live CL reconnect and resource/WAL qualification remain. |
| GL-044/045 | Actual access-list discovery and returned-list gas behavior are implemented; final API/gas fixtures and selected multidimensional trace semantics remain. |
| GL-050–055 | Actual Lighthouse/builder transitions, selected authentication and optimistic state, payment invariants, SSZ/proofs and partial-message recovery remain open. Merged upstream transition changes can be audited immediately; these are not all blocked by unresolved alternatives. |
| GL-060–063 | The combined development gate passes. Independent gas/current-text corpus, application compatibility, real multi-client devnets, restart/cold-sync, measured limits and release qualification remain open. |

Next implementation order: finish the compatible fixture/independent current-text gas corpus; audit actual CL adoption of already merged transitions; implement the settled 8282/request and eth/72/snap/2 production flows in their own branches; complete the chosen gas/API and authentication model when the corresponding upstream inputs are recorded. Continue independent work while those final decisions remain unsettled. No final release date or completion percentage is inferred from PR count.


## Signed current-text gas follow-up

The extended integration at `24e0bd542ebb36676ab6831a33bfb43cd4d6f2db` passes all 211 tested Go packages and the local build. PR 75's independent command runs 44 comparisons with frozen signed transactions, full pre-state/config/context and an unmodified Geth v1.17.10 reference. The clean [aggregate report](../../audits/glamsterdam-gas-2026-11-18/INTEGRATED.md) retains 19/22 Amsterdam candidate mismatches. These are not converted into passing parity tests. It fixes the reference-context omission of CPSB in the diagnostic itself and uses a failing mutation to prove that omission is detected.

The remaining five Osaka high-gas comparisons are explicitly transaction-transition observations. Native outer block validation already checks the Osaka limit; that outer layer is not invoked by this corpus, so these comparisons do not prove a production block-validation bypass. The ordinary creation/destruction and value-log controls demonstrate the effect of PRs 72 and 71. Pool/block/RPC, multi-transaction, 7702/application and full-client gates remain uncovered.

The final gas alternatives were rechecked through GitHub on 2026-11-18 at this follow-up: EIPs PR 12581 remains open/unmerged at `2ba9176f360c3c95641c6c1b5428ac3113335c4b`, and PR 12582 remains open/unmerged at `19d788c896ad35fc53dc5721dbe70ed91cb40069`. The pinned merged LIFO text remains the measurement baseline; this does not close the final release-model decision.


## Actual Lighthouse fork-transition repairs

Source audit of the current paired client (`f85fa05b6`) and upstream candidate `8104a2467` reproduced two missing merged-rule adoptions. They are now separate client-repository reviews:

- [Lighthouse fork PR 1](https://github.com/jiayaoqijia/lighthouse/pull/1): unset the Gloas fork slot's payload availability; real scheduled slot transitions cover the ordinary and wrapped historical ring.
- [Lighthouse fork PR 2](https://github.com/jiayaoqijia/lighthouse/pull/2): independent builder/validator keys, credential-first funding, exact pending-deposit retention and full/cold/seeded-cache post-state equivalence, following merged consensus-specs PR 5736.

Each branch passes its 23 library tests, cargo check, formatting and complete workspace make lint-fix. Cherry-picking only the owned changes onto the actual paired-client base gives `e6ff51d52`, which passes 24 library tests, cargo check, formatting and full workspace lint-fix. [Combined evidence](../../verification/glamsterdam-fork-integration-20261118.md) and [exact client reviews](client-reviews.json) retain the source boundaries. The original CL checkout is unchanged; no upstream PR was merged.

GL-050 is therefore partially implemented and tested for these transitions. This does not close actual multi-client devnets, bid/authentication choices, optimistic invalidation, payment invariants, complete SSZ/proofs, partial-message recovery or the EIP-8282 end-to-end contract pipeline. Existing CL builder-request containers, processing and Gloas partial-group version 0x01 are source-present, with remaining behavior gates explicitly unqualified.


## Ordinary blob authentication and recovery

[PR 76](https://github.com/alt-research/eth2030/pull/76) repairs actual ordinary type-3 HTTP admission of a wrong final cell proof and qualifies complete-data pool/build/pooled-wire/Engine consumers. It also preserves sidecars in the production transaction journal and verifies fresh-node readback. Frame and ordinary cell authentication share the existing batch verifier. Canonical legacy proof kinds and BlobAndProofV1 HTTP DATA are verified, and payload collectors propagate proof failures instead of fabricating zeros.

All 210 tested Go packages pass on the independent branch, with targeted race/vet/build and four failing guard mutations. Aggregate source `3b05bdb` (all feature sources plus this repair) passes all 211 tested Go packages, tagged Core/EIP tests, build and affected vet. [Verification](../../verification/glamsterdam-blob-validation-20261118.md) records exact scope. The old sparse-admission placeholder now states the actual full-data boundary; it does not qualify sparse relay. GL-042 provider/sampler/custody/GetCells/reconstruction remains open, as do final gas, full multi-client recovery and release gates.


## Builder execution requests

[EL PR 77](https://github.com/alt-research/eth2030/pull/77) implements EIP-8282 using the exact canonical contracts and mandatory Amsterdam extraction. Explicit owned allocations precede genesis roots; supplied network state is not silently rewritten. Tests qualify fees, pre-fork queue retention, endian conversion, 64/16 drain caps, failure rollback and actual node-backed HTTP Engine production/import, including altered-request rejection.

[Lighthouse PR 3](https://github.com/jiayaoqijia/lighthouse/pull/3) independently decodes those same real-BLS records through actual builder state processing. Correct builder domain registers immediately, validator domain is ignored, top-ups retain the address, and inactive/unauthorized/pending-payment/pending-withdrawal exits are rejected before a valid exit at the exact epoch. The paired-client composition passes 25 library tests, cargo check, formatting and full workspace lint-fix (3m 24s).

The EL feature aggregate at `d97be4c3` passes all 211 tested Go packages, tagged Core/EIP tests, affected vet and local build. Composed blob/factory/slot fixtures explicitly predeploy the required contracts, preserving their original obligations. The independent EL branch additionally passes 25 targeted race-tested packages and three failing guard mutations; CL has three compiling failing mutations. [EL verification](../../verification/glamsterdam-builder-requests-20261118.md) and [CL verification](../../verification/glamsterdam-builder-request-wire-20261118.md) distinguish component evidence from an unrun full live devnet. Final gas, network deployment, actual full-client transport/recovery and release gates remain open.


## Canonical startup and Engine boundary review

[PR 78](https://github.com/alt-research/eth2030/pull/78) rejects missing canonical predecessors before genesis adoption and qualifies actual production HTTP methods against execution-apis `68d76a5b`. Osaka getPayloadV5 now has its own correct response, while old methods reject later-fork payloads. Required-field presence, exact DATA and bounded QUANTITY values are checked before permissive internal decoders; zero values remain valid. Valid forkchoice adoption survives invalid build attributes, and capability exchange no longer advertises itself.

The actual HTTP test crosses times 12/24, checks every required field missing/null, rejects truncation aliases of valid quantities and distinguishes invalid BAL RLP from invalid parameters. Four compiling guard mutations fail. The independent branch passes all 210 tested Go packages, tagged Core/EIP, affected race/vet/build. The composed feature source `1d9826c2` passes all 211 tested Go packages, tagged Core/EIP, affected race/vet/build, preserving PR 76 proof-generation errors during the legacy response rename. [Verification](../../verification/glamsterdam-fork-schema-20261118.md). Persisted runtime configuration adoption is being implemented separately; no release completion is inferred.


## Persisted configuration recovery review

[PR 79](https://github.com/alt-research/eth2030/pull/79) makes actual Node startup consume and persist resolved runtime configuration before replay/adoption. Real signed HTTP and fresh-node tests reject altered historical rules/chain identity/contract parameters without changing head or metadata, then adopt a genuinely future activation and read back the same balance/root across two restarts. Unknown executed legacy history is preserved and explicitly rejected rather than assigned a guessed old configuration.

The registry includes omitted optional/BPO/IPlus fields, compatibility checks either side of activation, and next-fork selection uses actual BPO times. Each mutable schedule field participates in compatibility authority. Four compiling mutations fail the actual startup/persistence/address/global-isolation regressions. The independent source passes all 210 tested Go packages, affected race/vet/build; the aggregate `ab365362` passes all 211 tested Go packages, tagged Core/EIP, affected race/vet/build. [Verification](../../verification/glamsterdam-config-recovery-20261118.md). Configurable blob parameter values, full network/CL epoch alignment, final gas and release gates remain open.


## Configured blob capacity and settlement

[PR 80](https://github.com/alt-research/eth2030/pull/80), stacked on PR 79 and composing PR 76, repairs the actual hardcoded block capacity, incorrect blob prices/excess, missing ordinary fee burn and missing opcode context. The [review-only delta](https://github.com/fyInALT/eth2030/compare/review/glamsterdam-blob-foundation...feat/glamsterdam-blob-schedule) separates this feature from its dependencies. Explicit EIP-7840 parameters now drive builder/import/header validation, receipts, pool head policy, Frame/payload/RPC contexts and historical authority. Official historical BPO dates are verified; final Glamsterdam activation is not inferred.

Real signed version-1 HTTP controls pass at 7/10/21 blobs across Prague/BPO transitions, including independent Geth formulas, fee debit/opcode/receipt/feeHistory, invalid header/history guards and fresh receiver restart. The current path passes race without changing timeouts; the earlier legacy-wrapper timeout remains failed evidence. Six behavioral mutations are detected. [Verification and exact results](../../verification/glamsterdam-blob-schedule-20261118.md).

The composed source `53099cbe` passes all 211 tested Go packages, 27 tagged Core/EIP packages, 54 targeted race-tested packages, affected vet and build. This settles the qualified blob foundation; it does not close execution/state gas, eth/72 custody/provider/sampler/reconstruction, snap/2, full CL/multiclient or release gates.


## Snap version 2 moving pivot and reorg recovery

[PR 81](https://github.com/alt-research/eth2030/pull/81) composes PRs 69 and 77 and implements actual Node snap/2 negotiation, ordered committed BAL exchange and staged state catch-up. Verified point proofs restore old-only fork effects; partial cursors survive and missing orphan BALs force fresh bulk synchronization. The wire distinguishes unavailable 80 from empty c0 through actual custom send/receive paths. Five real encrypted Node scenarios and unmodified Geth SDK packets qualify root/adoption/restart and corrupted proof rejection. Seven compiling behavioral mutations are detected. [Feature evidence](../../verification/glamsterdam-snap2-20261119.md).

The composed source `13969eaf` passes all 211 tested Go packages, 27 tagged Core/EIP packages, affected race/vet and build. [Composition evidence](../../verification/glamsterdam-snap2-integration-20261119.md). Full Geth cold sync, mainnet resource/retention qualification, eth/72, final gas, complete CL and release gates remain open. The PR count does not indicate release completion.


## Sparse blob relay and Engine cell consumers

[PR 82](https://github.com/alt-research/eth2030/pull/82) is a separate major feature review based on the qualified submitted composition. It connects actual eth/72 negotiation, flat correlated cells, elided metadata, bounded signed staging, KZG authentication and 64-index reconstruction before ordinary full-data pool promotion. Native encrypted peers qualify two-provider sampling, custody plus random extra, custody expansion and eager mode. Actual Nodes qualify HTTP propagation, Engine proof/data/null projections and fresh receiver reacquisition. [Feature evidence and exact results](../../verification/glamsterdam-eth72-20261119.md).

The implementation follows the formal devp2p request-ID schema and the October 30 merged blob-major correction (PR 291 / issue 290), rather than the stale alternative ordering. Eight compiling behavioral mutations are detected. Independent source passes all 212 Go packages, 27 tagged Core/EIP packages, 36 affected race packages, vet and build. The aggregate `20b1c1f` passes all 212 packages, 27 tagged packages, 97 affected race packages, vet and build. [Composition evidence](../../verification/glamsterdam-eth72-integration-20261119.md).

PR 81 also explicitly stops its test-owned encrypted servers before Node/database restart; all five snap/2 scenarios pass under race after that cleanup. Full independent-client gossip/cold sync, actual CL custody reconnect, resource/fairness and sparse WAL campaigns, final execution gas and complete release qualification remain open. No upstream PR was merged.

EIPs PRs 12581 and 12582 were rechecked at this follow-up and remain open/unmerged at the same frozen heads. The merged LIFO text remains the versioned development/measurement baseline; final release selection stays unresolved.


## Unchanged storage pivot proof and gas draft follow up

PR 81 now reproduces and fixes an actual moving-pivot cache miss for unchanged partially downloaded storage accounts. The verified consumer fetches the exact new-root account proof before trusting its storage root. Seven encrypted Node scenarios and nine compiling detected mutations qualify the expanded feature. Aggregate source `c0df6062` passes all 212 Go packages, 27 tagged Core/EIP packages, affected snap/scheduler/Node race/vet and build. [Composed follow-up](../../verification/glamsterdam-snap2-binding-integration-20261119.md).

[Draft PR 83](https://github.com/alt-research/eth2030/pull/83) implements the frozen merged LIFO development baseline in an isolated branch; no unmerged alternative is combined with it. Clean source `c6cdf15a` matches the original 22 signed Amsterdam transition cases, previously 19 differing. The next source adds a 64 KiB deployment (23 matching transition cases) and actual Node HTTP pool/build/import/restart: a 110M-total-gas transaction produces state/header gas 100,453,680 while the two-transaction receipt cumulative payment is 100,516,383. A correctly rehashed scalar-sum header is rejected. This draft is not part of the qualified aggregate.

Outer reserve/fee and broader 7702/application/system/RPC coverage and old-price fixture repairs continue. Full-suite failures remain visible; native helper checks and initial signed parity do not close all GL-020–025/060 gates. Final release-model selection and CL/multiclient/resource qualification remain open. [Draft evidence](https://github.com/fyInALT/eth2030/blob/feat/glamsterdam-state-gas/docs/audits/glamsterdam-state-gas-2026-11-19/README.md).

## Canonical system transitions and Gas draft follow-up

[PR 84](https://github.com/alt-research/eth2030/pull/84) repairs the obsolete history address/window and fabricated missing/custom/reverting-contract writes. Four complete roots agree with unmodified SDK; actual Node build/import/query/wrap/restart, seven genesis profiles, seven detected mutations and full/tagged/affected race/vet/build qualify the independent feature. [Evidence](../../verification/glamsterdam-system-transitions-20261119.md).

Composition `2f253d33` passes all 212 Go packages, 27 tagged Core/EIP packages, eleven affected race packages, vet and build. Published integration head is `b1a696d1`. [Composed evidence](../../verification/glamsterdam-system-integration-20261119.md). Draft PR 83 remains outside this qualified branch. Its local checkpoint adds nine passing signed authority scenarios and actual 8/16/17-slot system-margin controls; full-suite fixture migration, RPC/application and release qualification remain unfinished.

## Qualified current gas development baseline

Draft PR 83 now has a clean `2cde17ad` checkpoint: 42/42 signed Amsterdam scenarios match independent unmodified SDK with explicit state-byte price. Ten authority controls, observable LIFO refill, actual system margins and two signed HTTP/Engine blocks qualify nonce/code rollback, capacity, receipts, blockValue, sender/miner settlement and next base fee. Full 212 Go packages, 27 tagged Core/EIP, broad affected race, final 33 affected race, vet/build and ten compiling detected mutations pass. [Evidence](../../verification/glamsterdam-state-gas-20261119.md). The initial escaped rollback mutation is preserved alongside the stronger late-OOG fixture.

Separate branch `feat/glamsterdam-rpc-gas` repairs reproduced live HTTP call/estimate/creation/trace consumers and canonical prefix replay. It has not yet completed composed qualification or received its own PR. The Gas draft remains outside the stable reviews 65–82/84 integration, and full program/release gates remain open.

## RPC method review and continued simulation work

[PR 85](https://github.com/alt-research/eth2030/pull/85) qualifies real HTTP current-gas calls, estimates, access discovery, signed authorizations, creation, complete overrides and canonical transaction/block opcode traces. Historical fork/system/prefix consumers and eleven compiling guard mutations pass. Full 212 packages, 27 tagged Core/EIP, 48 affected race and the additional five-package access follow-up pass, with vet/build. [Evidence](../../verification/glamsterdam-rpc-gas-20261119.md). Stable integration still excludes Gas/RPC until the remaining scoped composition is qualified.

The user explicitly selected the currently merged LIFO model first, with final design review retained. Stateful simulation is being implemented in separate branch `feat/glamsterdam-simulate-v1`. Nested dimensional tracing, complete CL lifecycle, independent fixtures/processes and resource/release gates remain open.
