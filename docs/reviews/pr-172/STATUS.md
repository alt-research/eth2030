# Group E status

Remaining delivery is now being exercised through an independently pinned
Lighthouse two-pair network. The new acceptance job includes positive signed
gossip/finality, an actual native VALID/false omission with CL fork-choice and
restart observations, and bounded bidirectional PQ-byte transport measurements.
See [EXTERNAL_HEZE.md](EXTERNAL_HEZE.md). These results are pending actual hosted
execution; tool unit tests do not close the external integration gate.

Production source after complete-consumer audit: `70facd01d15e8068cfc9839b3f0e352784765277`, based directly on merged PR #171/master
`cc31afffaa12919bbc08dc854c03af1e1928dc3c`. No base conflicts or changes to
read-only refs or independent A/B/C/D expected oracles.

HGT-040–045 native implementation is delivered. The selected current Heze
companion authenticates and retains branch-keyed ILs, separates timely/late
views and detects equivocation. Native Bogotá builds private IL transactions,
retries nonce/funding cascades, enforces gas/complete-block RLP capacity and
retains execution/satisfaction decisions across branch selection and restart.

The current 36-criterion register names 40 mandatory native consumers, including the executable external-network fixture. Profile 1 checks
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

## End-of-transaction state audit results

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
wait for the exact CL view to be resubmitted. Focused regressions and fresh final local qualification pass. Hosted exact-head
qualification is tracked separately in the English PR description.

The snap ancestry review additionally reproduced a valid BLOCKHASH-dependent
Frame omission being excused when authenticated ancestor headers were available
without bodies. IL callbacks now follow those headers and propagate missing
header errors. A real post-pivot native backend regression passes after the fix.
The register now requires 34 consumers across 21 sources; the final control register contains 26 removals.

The PQ wire review also reproduced foreign-chain and inactive nonce-schema
envelopes incorrectly reaching an undefined omitted-witness gate. Native
import and building now share the executor's existing immutable-context checks.
Both new subtests pass, while context-valid omitted witness composition remains
explicitly unimplemented pending specification.

Final local qualification at `70eddaad4f4460d10d75d0ff6baba3ab087151d6`: 244 full-suite packages and 24,245 named
passes, 76 selected race packages and 12,091 named passes, zero failures. All
34 mandatory E consumers pass without skips. All 26 compiled semantic removals
fail their intended unchanged consumers. Builds, vet, 12/12 differential cases,
109 Python checks in both modes, A62/B79/C102/D55/E34 source-bound checks, website
path/build/export and desktop/mobile visual/DOM checks pass. The current source
register pins 21 immutable normative files. Optional skips remain inventoried;
the B fixture-only eftest consumer remains outside selected race packages.

The site source and local export are updated; the existing GitHub Pages workflow
publishes after merge. No Profile 2 consensus activation, context-valid omitted
PQ witness composition, external CL/ePBS network acceptance or network-tail
qualification is claimed. Current master remains the merged #171 baseline,
with no conflicts; independent Lean work, read-only refs and A/B/C/D expected
oracles remain unchanged. Source-bound snapshots qualify the source above;
publication/check links belong to the English PR record.

## Pre-execution validity and complete-consumer audit results

This pass starts from published head `15262949d2b4dbf0f41bf4919dd0a2d0d5550c7c`.
All three upstream source heads remain unchanged. The review will recheck all
nine tasks and explicitly investigate the boundary between definitely invalid
Frame preconditions and the unspecified omitted-witness composition. Independent
Lean work and all read-only references remain preserved.

The new raw Engine cases reproduced definite pre-frame nonce/fee/capacity failures
being treated as unknown witness validity. Actual ordinary SSTORE usage also
reproduced wrong Frame reservation verdicts and version-1 cache reuse. Complete
RLP serialization reproduced a state-dominated width underestimate. Native cache
creation/rename failures and a redundant clock lookup reproduced partial ingress
publication. All focused native/core regressions now pass after shared executor
preparation, complete dimensional accounting/projection, version-2 migration and
owned cache publication. Current upstream 8037 differs from read-only refs; its
existing current C/D snapshot is reused. The register now requires 39 consumers
across 22 normative files. Fresh final local qualification passed; exact-head hosted results are tracked
separately in the English PR description.

The first fresh full run failed the older D native omitted-witness fixture: it
used a generic transaction with correct next nonce but insufficient fees.
Only that input is corrected to the canonical account's next nonce and adequate
fee budget. Its original missing-witness error assertion and all independent
expected goldens remain unchanged. A complete fresh rerun is required.

Final local qualification at `ee798638b27a186bf168ebb830de4c7f45e5a8e7` passed: 244 completed full-suite
packages / 24,267 named passes (124 optional skips), 76 selected race
packages / 12,113 named passes (87 optional skips), zero failures. All
39 required E consumers pass without skips. All 36 semantic-removal controls
compile and fail their intended unchanged consumers. Native/wrapper builds,
vet, 12/12 differential cases, 109 Python checks in both modes, A62/B79/C102/D55/E39
source-bound checks and fresh website build/export/desktop/mobile checks pass.
The B fixture-only eftest consumer remains outside selected race packages.

All nine tasks remain tied to defined normative consumers. Profile 2 enforcing
rules, precondition-valid omitted PQ witness composition, external CL/ePBS
acceptance and network tail latency retain their explicit gates. Website source
is updated; Pages deploys after merge through the existing master workflow.
Read-only refs and independent expected oracles are unchanged. Publication and
hosted results belong to the current English PR record, with no earlier head's
CI results counted as qualification for this source.

## Independent completion pass on published head 9b4884be

The latest EIPs, consensus-specs and execution-apis heads still match all pinned
Group E sources. The 7805 discussion remains at 17 posts (last November 21),
and 8369 at four (last September 9). EIPs PR #12110 remains merged, while
optional direct-proof PR #12672 remains open at the selected revision. No new
binding rule or omitted-witness composition has been published.

| Task | Independent review outcome |
|---|---|
| HGT-040 | Source precedence, fork selection and execution/satisfaction separation remain correct. |
| HGT-041 | SSZ, BLS context, exact bytes, first-arrival timing, equivocation, committee-position requests and retention reviewed. |
| HGT-042 | Required Bogota fields, malformed transaction tolerance, capabilities, fork versions and Boolean/null responses reviewed. |
| HGT-043 | Sender-indexed and dynamic retries, trial rollback, immutable builds, cancellation, dimensional reservations and exact RLP reviewed. |
| HGT-044 | Account-only Profile 1 agrees with selected ordinary validity; non-Profile-1 full-state paths and transaction-end boundaries remain distinct. |
| HGT-045 | Actual native builds/imports, accepted views, branch/restart/finality decisions and version-2 migration reviewed. |
| HGT-046 | Defined reference predicates remain implemented; enforcing Profile 2 is still blocked by the upstream Standards Track extension. |
| HGT-047 | Local authenticated ingress and CL decision predicates reviewed; external beacon transport/CL/ePBS acceptance remains HGT-070/080. |
| HGT-048 | Real direct-8504 byte/verification evidence remains bounded; omitted witness composition and network tail latency remain unqualified. |

No additional defined-rule functional gap was found in this pass. All 39 mandatory
E consumers were freshly executed in normal and race modes: 39/39 passed in each,
zero failures and zero required skips. Fresh `go build ./...`, `go vet ./...` and
all A-E source/hash checks passed. Commands, exact log identities and source
equivalence are recorded in [INDEPENDENT_QUALIFICATION.json](INDEPENDENT_QUALIFICATION.json).

This pass changes only scoped audit records. Production code, tests, reference
pins, independent oracles, qualification tools and website files are byte-identical
to published head `9b4884be`. Its 244-package full/76-package runtime-race
qualification, 36 compiled controls and 16 successful hosted checks remain
identified as that source's results; they are not relabeled as newly executed.
The website already displays the accurate implementation/gate split and needs
no further progress change. Full public Group E activation is still not complete.
New publication-head CI outcomes are recorded separately in the English PR.

## Remaining-delivery local qualification

The current 40 required consumers passed without skips in a fresh 245-package
full run (24,268 named passes, 124 optional skips) and a ten-package selected
race run (97 named passes, zero skips). Full builds and fixture vet passed.
The 118 Hegota tooling tests pass in both normal and optimized modes. All
70 devnet-script tests pass with task-local Python 3.14.7. An initial run under
the host's Python 3.9 failed four unrelated modern-runtime checks; that failure
and the complete modern-interpreter rerun are retained. No assertion was
weakened. See [EXTERNAL_LOCAL_QUALIFICATION.json](EXTERNAL_LOCAL_QUALIFICATION.json).

The previous 36 compiled removals retain their original run identities. Every
production and consumer hash used by them was rechecked unchanged. They are
not represented as newly executed controls for the added proxy or fixture.
Actual external Heze network acceptance remains pending the hosted build/run.
