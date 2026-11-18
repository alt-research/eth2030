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
| Gas/state execution | Selected coherent intrinsic/runtime/state/refund/floor model; upstream choice gate retained | Pending |
| [PR #69](https://github.com/alt-research/eth2030/pull/69) — eth/69–71 | Negotiated protocols, canonical wire, correlated historical consumers and encrypted-socket checks | Submitted; independent full-client cold-sync gates remain open |
| [PR #70](https://github.com/alt-research/eth2030/pull/70) — receipt bloom | Rebuild omitted wire bloom from actual nonzero logs | Submitted; included in branches 69/71 for standalone tests |
| [PR #71](https://github.com/alt-research/eth2030/pull/71) — transfer logs | Canonical Amsterdam activation, value transfer ordering and exclusions | Submitted; final gas/cross-client composition remains open |
| [PR #72](https://github.com/alt-research/eth2030/pull/72) — destruction and BAL | Balance-preserving EIP-8246 lifecycle, journaled creation/revert, final-state BAL and CREATE2 nonce/prechecks | Submitted; native and Frame behavioral tests pass; full fixture/gas qualification remains open |
| [PR #73](https://github.com/alt-research/eth2030/pull/73) — stack opcodes | Current EIP-8024 immediate encoding, canonical activation and every valid depth/pair | Submitted; native transactions and published vectors pass |
| eth/72, snap/2 and CL lifecycle | Full remaining protocols, actual CL/builder and recovery acceptance | Pending |

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
| GL-012 | PR 78 qualifies canonical startup dependencies, Engine version windows, Osaka V5 shape and required-field/encoding bounds through actual HTTP. PR 79 adds actual pre-replay configuration authority and registered optional/BPO compatibility. Configurable blob values, unregistered private extensions, broader successor schemas and release inputs remain open. |
| GL-020–025 | Coherent gas model, validation/pre-execution/runtime/settlement and method-specific simulation remain open. Current merged LIFO text is the measurement baseline; final release selection is still U-01/U-02 dependent. No partial constant patch is presented as a coherent model. |
| GL-030 | Native size limits, pool/creation boundaries and current stack opcode semantics are implemented. Final gas/estimate and cross-client cases remain. |
| GL-031 | Canonical factory implementation and real CREATE2 behavior are implemented. Actual generated genesis/deployment control and final gas qualification remain. |
| GL-032/033 | Native protocol log and destruction/state/BAL corrections are implemented, including canonical Frame composition. Full upstream scenario matrix, selected gas and independent execution remain. |
| GL-034 | PR 77 implements canonical contracts, mandatory extraction, real HTTP production/import and identical real-BLS CL component consumers (Lighthouse PR 3). Full live deployment/transport/recovery and final gas remain open. |
| GL-040/041 | Canonical persisted bodies already existed; retention, empty BAL recovery, negotiated eth/69–71 and correlated bounded history consumers are implemented. Full independent-client historical/cold-sync and recovery gates remain. |
| GL-042/043 | Actual eth/72 cell relay/provider/sampler and snap/2 state-sync flows remain open. Existing helper names do not satisfy these tasks. |
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
