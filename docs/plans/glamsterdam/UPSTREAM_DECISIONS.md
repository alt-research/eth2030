# Upstream decision register

This register uses the upstream snapshot collected on 2026-11-18 and retained in the [discussion index](../../audits/glamsterdam-upstream-2026-11-18/discussion-index.json). States below are archived observations, not a fresh GitHub status poll during plan preparation. GL-001 must refresh them before implementation and record the compatible spec/fixture/client set. A proposal, author approval or closed issue is not an adopted rule.

An engineering owner may close a decision entry only with an explicit adopted revision and rationale. This does not introduce a user confirmation step: continue authorized independent work while dependent semantics are unsettled.

## Decision records

### U-01 — execution/state gas refill and compatibility model

Status: unresolved at the snapshot. Owner role: EL + Validation. Affects GL-020–025, GL-030/032–034, GL-045, GL-060/061 and final G2/G6.

The merged EIP text and its current LIFO model are the versioned baseline. Open [EIPs #12581](https://github.com/ethereum/EIPs/pull/12581) proposes reservoir-only refill and a legacy execution-gas share. An [additional author comment](https://github.com/ethereum/EIPs/pull/12581#issuecomment-6412891161) proposes delayed settlement credits and explicitly says it was not pushed. These are three distinct artifacts; do not implement a mixture.

Close when the chosen protocol model is explicitly recorded from authoritative upstream decisions/specifications, and its compatible executable-spec/native vectors are pinned. Record initialization, charge order, success/revert/halt, pre-execution failure, refunds, floor and block-fee equations. Review the [eight application invariants](https://github.com/ethereum/glamsterdam-gas-invariants), including fixed gas budgets and off-chain behavior. ACDT November 16 did not decide the path forward; ACDE November 19 was a future discussion at the snapshot.

Work available now: canonical header/slot repairs, test corpus construction, independent current-text measurements, settlement invariants and compatibility inventory. Release acceptance remains pending until a coherent target is selected and qualified.

### U-02 — optional per-call state-gas allowance

Status: proposed, not adopted. Owner role: EL + CL. Affects GL-022/023, GL-025 and corresponding compatibility tests only if selected.

Open [#12582](https://github.com/ethereum/EIPs/pull/12582) stacks on #12581 and limits child/subtree reservoir consumption using forwarded gas. It is not a second mandatory layer merely because a PR exists. If selected, define ancestor ceilings, draw accounting and rollback, and test nested factories, multisigs, 4337 and relayed 7702 batches. If rejected, remove allowance implementation from the release scope and document the actual shared-reservoir boundary. Keep tests for intended fixed-budget behavior under the selected model.

### U-03 — ePBS payload authentication and envelope format

Status: alternatives pending. Owner role: CL + Consumer. Affects GL-051/052/055, wire fixtures and final G5/G6.

Open [consensus-specs #5770](https://github.com/ethereum/consensus-specs/pull/5770) authenticates contents by locally calculating the EL block hash in CL gossip. Open [#5820](https://github.com/ethereum/consensus-specs/pull/5820) commits a payload root in the bid and changes the envelope to unsigned form. They are alternatives. The recorded #5820 head `7af0af3c` removed an earlier parent_payload_root; do not implement stale review suggestions as current PR content.

Close with selected CL schema, signing/authentication, seen/cache keys, fork choice and networking revisions, plus an explicit Sepolia/existing-chain compatibility strategy. Qualify bad claimed hashes, modified contents, empty transaction entries, oversized payload cost and seen-cache pollution. Negative tests must isolate the intended condition, as the [#5820 review](https://github.com/ethereum/consensus-specs/pull/5820#discussion_r4602321894) demonstrates.

### U-04 — CL transitions, onboarding and optimistic sync baseline

Status: partly settled; remaining changes require a selected compatible CL baseline. Owner role: CL. Affects GL-050/052/054/055 and G5.

Already merged at the snapshot: [#5736](https://github.com/ethereum/consensus-specs/pull/5736) separates builder/validator routing; [#5808](https://github.com/ethereum/consensus-specs/pull/5808) clears first-slot payload availability; [#5822](https://github.com/ethereum/consensus-specs/pull/5822) selects bid-slot signing domain; [#5835](https://github.com/ethereum/consensus-specs/pull/5835) restores the Gloas partial-message group version. Audit actual Lighthouse adoption now rather than waiting for unrelated proposals.

Open at the snapshot: [#5836](https://github.com/ethereum/consensus-specs/pull/5836) branch-consistent fork onboarding; [#5816](https://github.com/ethereum/consensus-specs/pull/5816) gossip message domains; [#5781](https://github.com/ethereum/consensus-specs/pull/5781) optimistic sync and [#5818](https://github.com/ethereum/consensus-specs/pull/5818) its vectors. Close each dependent behavior with the selected normative revision and real CL test evidence. Do not wait to test missing envelopes, FULL/EMPTY invalidation and first-fork boundaries.

### U-05 — RPC execution modes and trace representation

Status: mixed. Owner role: Consumer + EL. Affects GL-025/044/045.

Settled: merged [execution-apis #950](https://github.com/ethereum/execution-apis/pull/950) requires actual accessed entries from eth_createAccessList; its gasUsed corresponds to that returned list. The empty placeholder is a repair that can start now.

Pending: [#977](https://github.com/ethereum/execution-apis/issues/977) describes method-specific execution-cap/reservoir semantics; [#852](https://github.com/ethereum/execution-apis/pull/852) and [#976](https://github.com/ethereum/execution-apis/issues/976) describe trace dimensions and nested-frame accounting. Close with selected API fixtures/schema and node-backend behavior. Preserve the distinction between a simulation that skips transaction validation and an estimate intended for a valid transaction. Do not silently make user-visible arbitrary choices while labeling them API conformance.

### U-06 — builder payment invariants across late children

Status: unresolved interpretation and unverified additional reports. Owner role: CL + Validation. Affects GL-053 and payment qualification in GL-061.

[consensus-specs #5754](https://github.com/ethereum/consensus-specs/issues/5754) discusses slashed-parent payment asymmetry after epoch-window eviction. Later comments allege additional double-payment/no-quorum variants. Those allegations were not independently reproduced in the ETH2030 audits. First reproduce against the selected upstream and actual local CL; record builder debit, withdrawal count, proposer credit and supply. Then choose a fix/spec update backed by the intended settlement invariant. Do not copy a one-line suggestion before checking all paths or characterize it as a demonstrated local exploit.

### U-07 — scope dependencies and test authority

Status: core scope known; completeness of production adoption remains unqualified. Owner role: Validation + CL. Affects GL-001/002/054/060/063.

The 18 core items are from EIP-7773; 7495/7916 are dependencies of 7688 even while the [meta-list update #12449](https://github.com/ethereum/EIPs/pull/12449) is open. Supporting network EIPs are separate release requirements, not evidence that all current declarations are implemented.

Merged [execution-specs #3982](https://github.com/ethereum/execution-specs/pull/3982) provides fixture chain-invariant checking. Open [#3948](https://github.com/ethereum/execution-specs/pull/3948) is a candidate deployed-idiom corpus. Select exact fixture revisions and run native execution; a corpus's all-green outcome does not establish preservation of pre-fork assumptions. Document fixture generation backend, mutation sensitivity, unsupported oracles and actual consumers.

Do not promote these to blockers solely from their titles: [RANDAO #5826 reply](https://github.com/ethereum/consensus-specs/issues/5826#issuecomment-6871879332) explains mix copying; [SSZ #5823](https://github.com/ethereum/consensus-specs/issues/5823) explicitly states no live Gloas exploit; closed [PTC #5825](https://github.com/ethereum/consensus-specs/issues/5825) does not imply its closed/unmerged [PR #5827](https://github.com/ethereum/consensus-specs/pull/5827) shipped. Use relevant regression scenarios and qualified findings.

### U-08 — release schedule, retention and public-network compatibility

Status: local release schedule to be chosen; official Hoodi/mainnet dates TBD at snapshot. Owner role: EL + CL + release maintainer. Affects GL-012/040/061–063.

The [official upgrade plan](https://github.com/ethereum/pm/blob/master/glamsterdam-mainnet-plan.md) and [EF compatibility update](https://blog.ethereum.org/2026/11/06/glamsterdam-8037-8038) take precedence over treating EIP-7773's stale Hoodi row as an activation instruction. [#12568](https://github.com/ethereum/EIPs/pull/12568) removes that row and was still open. A local test schedule is not an official activation date.

Merged [EIPs #12347](https://github.com/ethereum/EIPs/pull/12347) requires BAL availability for at least 14,299 epochs through payload bodies V2. It allows retention or regeneration; preserve exact canonical bytes and cost/availability guarantees. Select the pruning/nonfinality behavior in the compatible spec set, handle missed slots using real epoch/time semantics and record the recovery strategy. Open [#12546](https://github.com/ethereum/EIPs/pull/12546) is related 8159 wording, not a reason to defer the already merged availability requirement.

Canonical header repairs change hashes compared with the experimental native format. Before integration, document whether a fresh test chain, resync or explicit migration is required; preserve existing unrelated data and do not rewrite public history. Final release qualification needs actual node/CL startup configuration, not only a saved genesis file.

## Updating a decision

Append the refresh timestamp, selected source URL and commit/release, adoption status and rationale, rejected alternative disposition, affected GL tasks, invalidated evidence and replacement vectors. Record implementer/reviewer responsibility when assigned. A changed upstream decision reopens the affected acceptance claims without erasing previous versioned evidence. Do not silently mutate historical audit results or mark all development blocked while unrelated tasks remain actionable.

## Networking refresh on 2026 11 19

At 2026-11-18 17:09:48 UTC, the supporting eth/72 source set is pinned to devp2p commit `88f1c80800d622cc3a542a2125bbe33cc18c9f9c` (file blob `a23dc172`). [Merged Geth 35428](https://github.com/ethereum/go-ethereum/pull/35428) confirms flat request IDs; [merged devp2p 291](https://github.com/ethereum/devp2p/pull/291) resolves [issue 290](https://github.com/ethereum/devp2p/issues/290) to blob-major cell order. PR 82 adopts those settled inputs and qualifies native/SDK consumers; full independent-client and CL/resource release gates remain open. This refresh does not choose an unmerged gas or payload-authentication alternative.

EIPs 12581/12582 remain open/unmerged at the baseline heads on this refresh. Current merged LIFO text remains the explicit development and measurement baseline, with final release selection unclosed.
