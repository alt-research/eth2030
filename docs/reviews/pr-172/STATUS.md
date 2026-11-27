# Group E status

Production source after transaction-boundary audit: `6ea395d8b9a9f3d9dc66376cda0f87c998941573`, based directly on merged PR #171/master
`cc31afffaa12919bbc08dc854c03af1e1928dc3c`. No base conflicts or changes to
read-only refs or independent A/B/C/D expected oracles.

HGT-040–045 native implementation is delivered. The selected current Heze
companion authenticates and retains branch-keyed ILs, separates timely/late
views and detects equivocation. Native Bogotá builds private IL transactions,
retries nonce/funding cascades, enforces gas/complete-block RLP capacity and
retains execution/satisfaction decisions across branch selection and restart.

The 36-criterion register names 34 mandatory native consumers. Profile 1 checks
actual signatures and full-width account state without EVM execution. Payload
execution validity remains independent from IL satisfaction. Replacements cancel
unfinished builds; context cancellation rolls back trials and aborts publication.

HGT-046 is explicitly gated by D04: EIP-8369 remains Informational and does not
define enough to activate Profile 2. Defined reference predicates and thirteen
activation prerequisites are implemented, but no production partial-stateless
replay or consensus claims encoding is asserted. HGT-047 qualifies local native
and CL decision adapters; external beacon transport/CL/ePBS network acceptance
remains HGT-070/080 work. HGT-048 measures real canonical direct-8504 bytes and
verification/local pipeline cost; omitted witness transport and network tail
latency remain unqualified. These gates prevent a claim that all Group E public
activation functionality is complete.

The website homepage, roadmap and EIP-7805 progress now display this precise
implementation/gate split. GitHub Pages deploys the repository site after merge
through the existing master workflow; this branch does not claim a live deploy.

Verification and final English PR/check links will be recorded after the exact
source runs complete. The first concurrent full/race attempt hit the existing
one-second native commit-barrier timeout. A separate full run passed without
changing that test. Header-copy vet findings were fixed before final reruns.

Final path audit also added authenticated pooled-witness retrieval for canonical PQ IL builds; a native RPC/Engine regression detects its removal.

Restart restores only the still-required retention window, rejecting future cached lists and requiring authenticated state for retained lists; long shutdowns no longer require obsolete beacon state.

Base EIP-7805 blob/other-envelope append validity remains enforced independently of the nonbinding 8369 profile boundary. Valid IL blobs require available authenticated sidecars; failed IDs retry after availability recovery. Static-invalid proof transactions remain tolerated instead of becoming missing-witness errors.

Append RLP sizing includes scalar/3D calldata usage and blob-gas header widths, including prior committed appends; exact serialization independently checks these boundaries.

Previous pre-re-audit local qualification passed: full suite 244 packages / 24,234 named pass
events (124 optional skips), selected race suite 76 packages / 12,080 named pass
events (87 optional skips), native/wrapper builds, vet, 12/12 differential
baselines, 109 Python checks in normal and optimized modes, and all A-E source
checkers. All mandatory full-suite consumers pass without skips. The B fixture-only
eftest encoding consumer is outside selected race packages, explicitly recorded.
All 13 compiled semantic removals fail their intended unchanged native consumers.
Site build, static-export checks and final desktop/mobile visual checks passed.
Hosted exact-head CI is tracked in the English PR record. These frozen local
artifacts do not pre-claim its outcome.

Published review: [PR #172](https://github.com/alt-research/eth2030/pull/172), targeting `alt-research/eth2030:master`. Exact-head hosted check outcomes are maintained in that English PR record, separately from the frozen local evidence.

## Completion re-audit results

A fresh independent worktree starts from published PR #172 head
`4663af699c4c9bbc48f59976dc26e00d805a78d6`. Latest EIPs, consensus-specs and
execution-apis heads still match the pinned Group E revisions. The review rechecked all nine tasks and their actual runtime consumers, rather
than treating previous check counts as proof of completeness. Unrelated Lean work in the
original worktree is preserved.

Re-audit found and reproduced two missing boundaries. Finalization deleted
completed Bogotá verdicts, causing a known stale VALID head to become SYNCING;
completed verdicts now survive as compact records while obsolete IL bytes and
unexecuted branches are released. Replacing an in-memory CL adapter skipped
reauthentication; live snapshots now pass the same ancestry/key/signature checks
as disk restores before an atomic replacement. Regression tests failed before
these fixes and pass afterward. Fresh final qualification passed: 244 full-suite packages / 24,237 named passes,
76 race packages / 12,083 named passes, 109 Python checks in each mode, 12/12
differential cases and all A-E source-bound checkers. All 30 required E consumers
pass without skips. All 15 compiled semantic removals fail their intended
unchanged consumers. Fresh website build/export and desktop/mobile visual/DOM
checks passed without horizontal overflow. Historical verdicts use an atomic
separate compact namespace to avoid scanning the full history during cleanup.
Exact-head hosted CI is tracked separately in the English PR description.

## End-of-transaction state audit in progress

A new review starts from published head `63f85e737d8dd84c8913d6f6ef996179e5e373d5`.
Upstream EIPs, Heze and Bogotá revisions remain unchanged. This pass focuses on
the precise EIP-7805 state after the last transaction, including the separation
from withdrawals and post-execution system calls. The independent Lean worktree
and read-only references remain untouched.

The raw Engine withdrawal regression reproduced a VALID/false error for an
unfunded transaction that only becomes funded after the transaction phase.
The corrected evaluator reconstructs precisely that phase when authenticated
BAL metadata shows relevant later changes or metadata is unavailable. Native
Frame tests cover both post-system invalidation and post-system validation.
Legacy live cache records are corrected across restart; legacy compact records
wait for the exact CL view to be resubmitted. Focused regressions pass; final
full/race/control and hosted qualification is pending.

The snap ancestry review additionally reproduced a valid BLOCKHASH-dependent
Frame omission being excused when authenticated ancestor headers were available
without bodies. IL callbacks now follow those headers and propagate missing
header errors. A real post-pivot native backend regression passes after the fix.
The register now requires 34 consumers across 21 sources and 24 removal controls.

The PQ wire review also reproduced foreign-chain and inactive nonce-schema
envelopes incorrectly reaching an undefined omitted-witness gate. Native
import and building now share the executor's existing immutable-context checks.
Both new subtests pass, while context-valid omitted witness composition remains
explicitly unimplemented pending specification.
