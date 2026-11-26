# Group C independent completeness re-audit

## Post-paymaster mixed nonce lifecycle audit (2026-11-26)

This audit starts from pushed source `e12ddc37c7ee08e83caa27e268d4619dc425cd11`.
The implementation correction is `b4840950`. The forty-criterion table below
still covers the complete C register; this follow-up additionally inspects the
interactions of its C21.4, C24.4 and C26.3 lifecycle requirements with ordinary
transactions already following an admitted Frame.

The current upstream EIP-8141/8250/8272/8151 file blobs are respectively
`9bc38a368881fa52e1a1a9970274e6511874d13f`,
`10bd1d84c9534c9db65f9ccbffcf8d5902b9c1ae`,
`0acc21ad8359be83c0929c7cc70e45326be8acff` and
`bfd0db906f55033442a064357d8e6768490b8fd5`: all match the read-only
references byte-for-byte. Current discussion streams have no additional post IDs
(8141: 181 available/highest 187; 8250: 5/6; 8272: 3/3; 8151: 32/33).
Their actual latest numbered posts were read, including EIP-8151's recovery-cache
and witness discussion. No new merged requirement changes the selected C paths.
PR #12041 is still open and unmerged at the same `4a98fcef` proposal; the bot still
requests one author review. The exact proposal implementation and its explicit
selection remain described in [CANONICAL_PAYMASTER.md](CANONICAL_PAYMASTER.md).

Two genuine omissions were reproduced against the exact preceding production
files, without modifying reference text or independent expected vectors:

1. Capacity eviction only considered the scalar sender tail. A Frame at nonce 0
   followed by an ordinary nonce 1 transaction was omitted, even if its expiry
   was earlier than all other Frames. All pending Frames now participate in
   expiry/priority ordering; ordinary tail and sole-anchor policy is preserved.
2. Public removal, expiry, dependency invalidation and bounded unpaid-attempt
   eviction removed a scalar Frame but left its ordinary successors pending.
   Removal now reconciles the contiguous scalar sequence against current state,
   queuing the orphaned suffix and preserving unrelated keyed domains. Normal
   inclusion with an already advanced state nonce retains executable successors.
   Freshly receiving an admissible predecessor promotes the suffix again, while
   an existing later gap stays queued. This never resurrects an evicted Frame.

The authority is EIP-8141 **Replacement and Eviction/Revalidation** (expiry order,
resource release and nonce consumed once on payment) together with EIP-8250
**Nonce state/Stateful validity/Nonce consumption**: empty `nonce_keys` uses the
legacy account nonce; nonempty keys are independent. Pending/queued bookkeeping
is local implementation policy preserving those validity rules, not a new
consensus rule or a new public restriction.

Mandatory native consumers `TestFrameCRemovalReconcilesScalarSuccessors` and
`TestFrameCEvictionIncludesNonTailScalarFrame` use real signed ordinary and Frame
transactions. Their 17 subcases cover base/empty-key/keyed domains, five removal
causes, multiple successors, readmission and a persistent later gap. The exact
old-source overlay has ten failing regression subcases and seven passing keyed
or inclusion controls; the two enclosing tests also fail as expected. The current
source passes all cases. The register/checker requires both consumers, with
manifest-removal guards. Current qualification and log hashes are recorded in
A/B/C evidence; exact pushed-head hosted results live in the English PR.

Local qualification on `b4840950340fea2d0779013c280edbe03b632133`: full/race
24,024/9,607 passing events, zero failures, all 240/57 packages complete;
mandatory A/B/C 62/79/94 with zero required skips; build/vet/both binaries,
baseline 12/12 and 91 Python checks per mode pass. Optional 124/74 skips are
inventoried independently. Nineteen native negative controls are retained.

## Prior independent audit

The later canonical-paymaster follow-up is recorded in
[CANONICAL_PAYMASTER.md](CANONICAL_PAYMASTER.md). It implements the exact
PR-12041 proposal and corrects candidate withdrawal availability. The source,
counts and qualification below describe the preceding independent audit.

Baseline: pushed PR head 6b9bbf8a4e5d5aa92432bf773ed54ceb0cce0059. This audit
checks HGT-020–029, all forty criteria, including public EIP-8250 and conditional
EIP-8151. It does not adopt Group D–J proposals or official CL/devnet release claims.

## Latest normative authority and discussion

EIPs master advanced to 514f00c369f9586e4d70a25da798ae2bd3d0ead7. EIP-8141,
8250, 8272 and 8151 are text-identical to the pinned ade9e9ca sources. The read-only
references and independent wire/state expectations are unchanged. Consensus-specs
remains d2021d2719235f0df4726a1d330502cb15aa8ea3.

The complete current discussion streams were checked: Frame transactions topic
27617 (181 available posts, highest post 187), keyed nonces topic 28437 (five
available posts, highest six), recent roots topic 28621 (three posts), and
account-code restricted ecRecover topic 27690 (32 available posts, highest 33).
Historical openings are not substituted for the current nonce-key-set or
canonical recent-root-frame definitions. The latest 8151 discussion reiterates
that caching may cover pure recovery only: the stateful code check must remain
outside it, as in the native EVM implementation. Bundler/SDK/fault-proof changes
outside this client's selected C paths are not claimed by its passing tests.

Still-open EIPs #12714 (PROOF mode), #12708 (witness-byte read restrictions),
#12644 (300,000 verification budget), #12670 (mirrored 8151 pointers), #12659
(single-use clarification) and #12642 (blob/GASPRICE wording) do not change the
merged rules. The native implementation retains the current 100,000 public
verification bound and existing signature-list encoding. Single-use nonce and
conditional signer rules already follow the merged 8250/8151 text.

Canonical-paymaster PR #12041 remains open at
4a98fcefa5c6f02a25289bf55b9d92902d02b4c6. The sole upstream runtime gate in
COMPOSITION.md remains valid. Generic PAY tracing/exposure and the opt-in local
v1 template must not be presented as finalized upstream canonical bytecode.

## Complete criterion reconciliation

The source/test register remains specs/hegota/group-c.json. Each row below
covers all four criteria of its task; referenced consumers are mandatory in the
source-bound checker rather than inferred from task labels.

| Task / criteria | Audited production paths and outcome |
|---|---|
| HGT-020 / C20.1–4 | All sixteen prefix shapes, exact code/top-level verifier permissions, gas/state caps, stop-at-payment and caught/transient trace restrictions remain qualified by native VM consumers. No unmerged opcode/witness rule was added. |
| HGT-021 / C21.1–4 | Selected nonce/code/balance/storage/root and conditional signer-class dependencies, separate helper hashes, fork changes and reverse-index cleanup remain covered. Corrected simultaneous head refresh: stale invalid reservations and stale signer paths no longer arbitrate against valid peers. |
| HGT-022 / C22.1–4 | Exact slot+1 admission and candidate slot, root age/storage/code checks, expiry heaps, complete keyed identities, MAX exhaustion and optional legacy reads retain native boundaries. Activation installs no manager/root state. |
| HGT-023 / C23.1–4 | Signed max_fee exposure, one withdrawal per payer, empty-code sponsor versus generic/canonical-local PAY, replacement and release remain intact. Added invalid-first and same-head shared-resource arbitration. |
| HGT-024 / C24.1–4 | Actual base/blob fee ranking, complete replacement identity, dimensional/byte/count bounds and bounded unpaid work remain intact. Head-time insolvency now follows expiry/priority pruning with deterministic ties. |
| HGT-025 / C25.1–4 | Exact plain versus blob wrappers, announcements/fetch metadata and bounds, full cell proof verification and structured receipts still flow through the same native handlers. Existing two-peer plain/blob consumers are rerun on the final source. |
| HGT-026 / C26.1–4 | Candidate-state/slot validation, independent reservations and private/IL separation remain covered. Corrected both builder interfaces so regular/blob partitioning cannot drop a scalar successor of a blob Frame; independent keyed domains remain independent. |
| HGT-027 / C27.1–4 | Raw submission, shared complete base/keyed formatting, noncolliding local pool keys, pending null locations, original block fee context, receipt/body persistence and restart remain covered. |
| HGT-028 / C28.1–4 | Authenticated raw simulation, exact timestamp/schema, dimensional measurements, explicit ordinary-method rejection, overrides/cancellation/trace/return limits and state isolation remain covered. No signature-mutating gas search was introduced. |
| HGT-029 / C29.1–4 | Actual public peer transport, ordinary pool construction, independent native replay/import, FCU, reorg and disk restart retain their consumers. New interactions have frozen prior-source red controls and preserved positive controls; final full/race/build/vet/evidence and hosted results are recorded separately. |

## Reproduced omissions and corrections

1. A nonce-invalid pending transaction retained its shared-payer reservation while
   a valid transaction was visited first. A simultaneous balance decrease evicted
   both. Preparation now removes head-invalid entries before collective limits.
2. Hash ordering alone decided which transactions survived shared-payer
   insolvency. Pruning now observes explicit expiry first and effective priority
   second. Root expiry stays in canonical slot units in its separate index.
3. Two valid prefixes swapping recovery-witness paths saw each other's stale
   signer shares and lost one transaction. All metadata now refers to the same
   head before EIP-8151 aggregate quotas are applied.
4. A blob Frame advancing the account nonce was partitioned after its ordinary
   successor, so the successor was skipped and never retried. Both builders now
   apply class preference among ready sender heads. Six actual build/replay
   cases cover base, empty-key and nonempty-key replay domains on both interfaces.

The old-source overlays contain the exact affected files from 6b9bbf8a. They fail
all four omission consumers, while the ordinary last-nonce, unrelated-sender
class preference and nonempty-keyed ordering controls pass. These are real native
counterexamples, not altered specification/oracle expectations. Qualification
log hashes and exact final source identity belong in evidence/hegota-{a,b,c}.

## Final local qualification

Implementation source: c4fffe3922d534acd88b919e29defffad239e5d7. Full native
build, native/wrapper binaries, vet, all 240 full-suite and 54 focused-race
packages complete successfully: 23,947/9,478 passing events and zero failures.
All 62/79/74 mandatory A/B/C consumers pass with zero required skips. The
124/74 optional skips have separate inventories. The independent historical
baseline passes 12/12 and ordinary/optimized Python each pass 89 checks.
Current-head hosted results are recorded in the English PR description/checks;
prior runs do not qualify this audit. PAYMASTER-CODE remains upstream-gated.
