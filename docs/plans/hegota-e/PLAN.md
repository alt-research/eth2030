# Group E: current normative implementation and acceptance

Baseline: merged PR #171 (`cc31afffaa12919bbc08dc854c03af1e1928dc3c`).
Scope: HGT-040–048, nine tasks, 36 criteria and 45 mandatory consumers.
Current EIP-7805: EIPs `c24e33014288b5e712da2629c185c91327c24c69`, merged PRs #12470/#12702;
Heze: `36888ee87b0faa6ad7804b017a0184e8e6e6a1c6`;
Engine Bogotá: `9e85b16c7fb54a838c9a612842c138c2bdf9bc8d`.
The latest EIP snapshot is outside read-only refs under `upstream/2026-11-28`.
Other selected source blobs and independent SSZ/BLS oracles remain unchanged.

The merged rebase supersedes earlier inline-CL/full-state omission assumptions.
See [the complete clause and discussion audit](../../reviews/pr-172/LATEST_7805.md).

## Required implementation

- Canonical progressive SSZ, domain `0x10000000`, 8192 transaction bytes and 16 committee members; authenticated CL committees, dependent roots and fork context.
- Current-slot gossip, first timely messages, late storage/forwarding, branch-scoped equivocation, committee-position requests, bounded retention and authenticated restore/reconfiguration.
- Exact `getInclusionListV1`, `forkchoiceUpdatedV5`, `newPayloadV6`, capabilities and VALID Boolean/non-VALID null satisfaction. Execution validity remains independent from satisfaction.
- Owned/cancellable builds, nonce/funding cascades, rollback, actual execution/sidecars/witnesses and EIP-7934 complete-block RLP validity.
- Omission checks at transaction-end state before withdrawals/post-execution calls: malformed/type-3 entries ignored; current-chain pre-execution validation; independent EIP-8037 reservations; no candidate execution or capacity consumption.
- No unmerged RLP or payload-size omission excuse. EIPs #12723 remains a separately reviewed, disabled proposal.
- Predicate-version-3 migration: versions 0–2 require reevaluation of retained views; old compact results require exact CL view resubmission.
- Native ingress commits only after persistence and one authenticated arrival clock; failed writes preserve views, quotas and pruning.

## Explicit prerequisites

EIP-8369 is Informational. Defined reference predicates and closed activation
checks are implemented, but enforcing Profile 2 still requires its Standards
Track claims/commitment, canonical state diffs, access/replay/code bounds, final
parameters/vectors and ePBS duties. No whole-transaction position stability is
inferred from one fresh keyed nonce.

The direct EIP-8504 profile has real byte/verifier/pool consumers. Omitted-IL
witness transport/composition remains unspecified. No extra field, excuse,
committee size or IL limit is invented. The high-gas size-limit composition
identified in #12723 remains an upstream release prerequisite.

Selected independent Lighthouse two-pair controls passed at their exact recorded
source: signed inclusion/common finality, same-payload VALID/false with CL fork
choice/restart, invalid/full exemptions and bounded canonical PQ-byte carriage.
Current publication-head requalification is separately required. Exhaustive
beacon requests/fork digests, mixed-client/ePBS release coverage and public tails
remain HGT-070/080 work. No complete public activation is claimed.

## Complete acceptance register

The executable clause/file/native-consumer mapping is
`specs/hegota/group-e.json`, checked independently by `tools/hegota/check_e.py`.
Its nine tasks each retain four criteria. HGT-040–042 cover source/CL/Engine;
043 covers building; 044 the merged pre-execution predicate; 045 native
integration/persistence; 046 informational reference/activation checks; 047
native and selected independent CL decisions; 048 bounded PQ resources.
Fresh pass events, source fingerprints, compiled controls and explicit skip
inventories are required. Prior A/B/C/D expected oracles remain unchanged.
Website/GitHub Pages roadmap and progress must show the qualified scope and
these exact remaining gates.

The latest #12702 clarification counts EIP-2718 canonical transaction bytes.
Exactly 8192 bytes are allowed; block RLP/SSZ framing and external witnesses are
not part of this budget. The native boundary consumer and source checker lock
this contract. See [LATEST_2718.md](../../reviews/pr-172/LATEST_2718.md).

Native Bogota publication checks the completed block against its immutable
requested IL before exposing a result. An execution-valid but unsatisfied
build fails; adding a real pool state transition can make the same payload ID
retry successfully. Candidate runtime failure is not an omission excuse.
