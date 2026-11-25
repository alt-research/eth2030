# Current-spec Glamsterdam audit

## Result and source boundary

One reproducible active-path defect was confirmed and corrected: valid mixed-case slot QUANTITY digits were rejected by both Engine backends (F01). A CI installer defect (F02) was also corrected. Additional review reproduced a prose/detailed-spec discrepancy for EIP-8282 top-ups (G05): the client agrees with the current detailed consensus reference. This is not a proof that every possible Glamsterdam interaction is correct, and the five explicit follow-ups below must not be replaced by the page's Complete labels.

Merged EL baseline: `5fedb7e5733fbef69767a711115ef9925e7bd769`, including PR159. The audit-only baseline commit `8caa00cd` changes CHECK/STATUS documentation, not Go code. Corrected functional source: `e9174cbe`. Further commits attach audit evidence only.

Official scope is EIP-7773 at EIPs repository `d27448e08f60513856bbe228ea7ce04f156209dc`: twenty scheduled core EIPs, five separate networking references and two informational references. [SOURCE_MANIFEST.json](SOURCE_MANIFEST.json) pins all 28 documents by source URL and content fingerprint. Sixteen of the seventeen compared contracts have identical bodies to the previously adopted `3db38a3` source after front matter is removed. EIP-8024's differences correct step numbering and a step cross-reference, without changing its decoding or execution semantics. EIPs 7495/7688/7916 were reviewed separately for their progressive-type contract.

The primary checkout's unrelated changes and existing devnets were preserved. No EL merge, CL source change, public activation parameter or new live acceptance campaign is included.

## F01: valid slot quantities rejected

EIP-7843's slot is a uint64, transported as an Engine QUANTITY. The request validator and the independent Geth `hexutil.DecodeUint64` accept `0x2A`, but the memory and native backends compared the input against lowercase-only formatted text. The actual native `engine_newPayloadV5` returned JSON-RPC `-32602` for the same slot and block that were accepted with `0x2a`.

Both backends now compare hex digits without regard to case, while still requiring the lowercase `0x` prefix, no leading zeroes, valid hex and the uint64 range. Tests retain the actual pre-fix failures in [REGRESSION.json](REGRESSION.json). They cover zero, 42, the maximum uint64, uppercase/mixed-case digits, seven malformed encodings, authenticated native Engine HTTP, an independently decoded Geth header and hash, canonical block RPC output, SLOTNUM execution and persisted restart. All twenty complete Engine/Node race package suites pass after correction; affected vet and native/Geth-wrapper builds pass.

This is a valid-input interoperability defect, not an observed state-root divergence or an explanation of the historical import stall.

## F02: proof CI installer used the anonymous GitHub API budget

The final documentation-head proof job 138568582860 failed in `Install pinned local proving components`: GitHub returned an anonymous API-rate-limit 403 while rzup looked up the pinned Rust release. Neither the client build nor proof execution had started; the missing artifact error was a consequence of that setup failure. The workflow now supplies its existing read-only `github.token` through rzup's documented `GITHUB_TOKEN` variable at the installer step only. Toolchain versions, installer checksum, guest image and proof acceptance assertions are unchanged. [CI_BOOTSTRAP.json](CI_BOOTSTRAP.json) records the original failure; the corrected hosted run is a separate gate, not a retroactive pass.

## Executed evidence

- Full Go baseline: 220 passing packages, zero failing packages; eighteen packages contain no tests. The 123 conditional test skip events are retained in [TEST_RESULTS.json](TEST_RESULTS.json), rather than counted as passed cases.
- Broad race run: 64 passing execution/core, Engine, SSZ, BAL, P2P, ETH and txpool packages, zero failures.
- Actual native signed-state executor against verified independent EELS `tests@v21.0.2`: all 14,665 Osaka and 16,908 Amsterdam cases pass with no errors. The archive fingerprint is verified. This invokes the native signed transition, not the Geth-backed state-test adapter.
- Unchanged Lighthouse tree `9fba2d60`, identical to merged eth2030 `983f62e9`: 345 type-library tests, 67 state-processing tests, four captured/progressive integration checks, four fork-choice tests and 41 proto-array tests pass. No fake-crypto feature is enabled. State-processing unit execution endpoints are mocks; these tests are not presented as a new live EL/CL network.
- Independent additional EIP-8045/8061 comparison: the unchanged selected Python consensus reference generates complete proposer vectors; the literal EIP-8061 functions generate churn/weak-subjectivity expectations. An external probe links the actual unchanged Lighthouse library. All 96 proposer indices across three slashing distributions agree, existing current/next lookahead remains unchanged, and ten churn/rounding/cap/weak-subjectivity cases agree. Deliberately changing either proposer or churn expectations fails real assertions. See [CL_CHECKS.json](CL_CHECKS.json), [CL_ORACLE.json](CL_ORACLE.json) and [negative controls](CL_NEGATIVE_CONTROLS.json).

The proposer probe executes at the actual dependent decision slot, after deriving the test state from the captured 384-validator state. A request outside that window is correctly rejected by the client. The large aggregate balances used for churn are explicitly injected helper-cache inputs; they do not describe a captured 36M/120M ETH state or a scale admission result. Initial probe setup errors (a reference constant's config location, a missing generated file and an out-of-window query) were corrected before successful qualification and are not classified as client defects or passing cases.

## Per-EIP review

[EIP_MATRIX.json](EIP_MATRIX.json) records requirements, actual source paths, executed checks and boundaries for every scheduled core item. Native counts below use explicit EIP numbers in official fixture paths. A zero does not mean there are no tests: some rules are CL-only or require block/header checks. Test names/counts are not coverage percentages.

| EIP | Reviewed contract and executed evidence | Additional result/boundary |
| --- | --- | --- |
| 2780 | Decomposed intrinsic/runtime charges, authorization failure and rollback; 269 native cases | No additional deviation confirmed |
| 7495 | Immutable schema, canonical encoding, independent roots/proofs, actual CL state | Merged PR159 corpus runs in the full suite |
| 7688 | Converted containers/lists and runtime limits; type/captured checks | Complete upstream CL fixture replay is not claimed |
| 7708 | Transfer log format/order, zero/self/fee exclusions and rollback; 74 native cases | No additional deviation confirmed |
| 7732 | Signed authority, request commitment, deferred parent processing, payments/registry; CL suites | Full fresh fork-choice/gossip reference campaign remains separate |
| 7778 | Gross block dimension versus user refund, floor and receipt semantics; block/replay tests | Signed state fixtures do not qualify all block fields |
| 7843 | Slot opcode/header/Engine/RPC/persistence; 18 native cases | F01 corrected with actual native regression |
| 7916 | Progressive shape, basic/bit packing, length mix-in and stable indices | Independent SSZ corpus and race pass |
| 7928 | Canonical RLP, indices, exact accesses/diffs, rollback, commitment/retention; 66 native state cases | Complete official blockchain/BAL corpus not replayed |
| 7954 | 64KiB code / 128KiB initcode, activation and failures; 58 native cases | No additional deviation confirmed |
| 7976 | Equal 64-gas byte floor and decomposed base; 543 native cases | Active canonical path reviewed separately from legacy helpers |
| 7981 | Data surcharge on both intrinsic/floor plus access repricing; 189 native cases | No additional deviation confirmed |
| 7997 | Exact runtime/address/nonce and explicit allocation, no fork injection; 40 native cases | Network deployment policy remains separate |
| 8024 | Immediates, reserved ranges, stack/EOF/JUMPDEST; 288 native cases | Current EIP changes are editorial only |
| 8037 | Fixed CPSB, dimension caps, LIFO/baseline/authorization rollback, create and system margin; 497 native cases | Whole-block corpus remains separate |
| 8038 | Access/write/refund/delegation ordering and prices; 263 native cases | Legacy experimental helpers remain a maintenance issue |
| 8045 | Pre-shuffle exclusion and fixed lookahead | 96 independent expected selections agree |
| 8061 | Independent churn dimensions, caps/rounding and asymmetric weak subjectivity | Ten actual helper boundary comparisons agree |
| 8246 | Preserve balance, clear nonce/code/storage, repeated destruction and rollback; 12 native cases | No additional deviation confirmed |
| 8282 | Real runtime/system queues, fees/caps/inhibitor, request wire, domain and exit authorization | G05: top-up prose versus detailed prefix precondition requires clarification |

## Separate networking and informational review

The broad race/full suites execute `eth/70` receipt pagination, `eth/71` BAL exchange, `eth/72` sparse sampling/custody and `snap/2` raw positional BAL tests. Source inspection traced requested receipt budgets, header commitments, missing/pruned entries, removed trie messages, custody updates, provider/sampler roles and actual cell validation. EIP-8136 is a CL Gossipsub partial-message extension: its implementation is in Lighthouse and the pinned libp2p dependency, not the EL's independent sparse-relay experiment. A fresh independent mixed-capability CL peer campaign was not executed here.

EIP-7904 and EIP-8261 are informational. They do not authorize silently installing speculative compute prices or a new network gas-limit schedule in the canonical Amsterdam transition. EIP-8025 and the executable Ethereum STF guest remain later scope, outside the Glamsterdam denominator.

## Remaining follow-ups, with explicit limits

1. **G01 — Complete native blockchain/BAL corpus replay.** No native official blockchain-fixture runner was found under `pkg/`; the supplied command is a signed state-test executor. Add a separate native blockchain runner that checks provided headers, BAL commitments, multi-transaction cumulative receipts, system request failures and block-dimension gas against independent expected results, including the 7778/7928/8037 interactions and builder queues. Existing block integration tests remain useful but are not relabelled as that full independent corpus.
2. **G02 — Full current CL reference fixture admission.** The fresh CL suites and additional independent probes are bounded. Replay the complete selected upstream fork/state-transition/fork-choice and converted-type SSZ fixtures at explicitly compatible pins, retain unsupported/failed cases, and add cross-client negative messages for bids, envelopes and parent execution requests. This is a coverage/qualification gap, not a newly confirmed missing builder transition.
3. **G03 — Independent mixed-peer network regression.** Repeat partial/full-column fallback and scoring, sparse custody reconnect/expansion/contraction, and positional/truncated BAL recovery against an independently selected compatible peer. Current local tests and earlier dated campaign receipts are not a new independent-network pass. Upstream Lodestar approval is still separately excluded at the user's direction.
4. **G04 — Retire misleading legacy gas-helper authority.** No active canonical Amsterdam caller was found for the obsolete `DefaultGlamsterdamGasTable`, `IntrinsicGasGlamsterdam` and old floor-helper family. They retain the old 4500-base/experimental schedules, and several tracker entries still point at them. Deprecate or remove that compatibility surface after checking external consumers and update source pointers to `amsterdam_intrinsic.go`, `amsterdam_message.go` and `amsterdam_gas_table.go`. Do not route production back through the old helpers or infer a consensus defect merely from their stale constants.
5. **G05 — Clarify builder top-up credential semantics before changing consensus.** EIP-8282 prose says an existing builder's amount is credited while credentials/signature are ignored. Its linked `d9663f85` consensus code and current master `d2021d27` instead validate the withdrawal-credential prefix before any registry lookup. The actual unchanged Lighthouse transition and independent current reference agree: against the captured registered builder, a 1 ETH request with prefix `0xb0` and zero signature credits 1 ETH; changing only the prefix to `0xff` credits zero. Neither changes the execution address. See [the discrepancy record](SPEC_CONFLICTS.json), [reference results](BUILDER_TOPUP_REFERENCE.json) and [actual client results](BUILDER_TOPUP_CLIENT.json). Either the prose must explicitly retain the common validity precondition, or a protocol revision must change the detailed transition and vectors together. This is not a demonstrated client/reference divergence and is not fixed by silently changing the CL.

48h/larger-scale acceptance and upstream Lodestar approval remain excluded, not passed. Public activation is a network rollout decision. The original-04 / 5.5-hour actual waiting-chain attribution remains **UNKNOWN**. No source/test result here changes those boundaries.

## Reproduce the additional independent CL checks

Use an unchanged generated consensus reference at `d2021d27`, remerkleable at `2f0baeef`, and a clean Lighthouse checkout with tree `9fba2d60` (merged eth2030 `983f62e9`). The Rust probe is an external qualification program; it does not modify or embed a replacement client implementation.

```sh
PYTHONPATH=/path/to/remerkleable /path/to/consensus-venv/bin/python docs/reviews/pr-160/cl_oracle.py /path/to/lighthouse/consensus/types/testdata docs/reviews/pr-160/EIP_8061_SOURCE.md.gz /tmp/glamsterdam-cl-audit
CARGO_TARGET_DIR=/path/to/lighthouse/target python3 docs/reviews/pr-160/run_cl_check.py /path/to/lighthouse /tmp/glamsterdam-cl-audit
```

The EIP-8061 source archive preserves the exact upstream bytes, including its trailing whitespace. The runner builds the actual type/state-processing libraries, selects exact Cargo artifact paths, links external probes, and requires both positive comparisons and deliberately wrong expectation controls. It also reproduces both builder top-up cases against the generated reference and actual CL. It refuses a dirty or different CL source tree. Hosted checks and mergeability for the final documentation head are recorded on [PR160](https://github.com/alt-research/eth2030/pull/160), independently of the local source-specific receipts above.
