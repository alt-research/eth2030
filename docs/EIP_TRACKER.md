# EIP Progress Tracker — Maintenance Guide

The site at `site/` renders a per-EIP progress tracker at **`/eips`** (overview) and
**`/eips/<id>`** (one page per EIP / roadmap feature). This document explains the data
model and the update workflow.

## Data source of truth

`site/src/data/eips.ts` is the **single source of truth** for EIP status labels,
paths and scoped gaps. The homepage's phase cards and `/roadmap` derive counts
from it through `site/src/lib/roadmap.ts`. Complete entries are omitted from the
homepage's pending-work lists; percentages count labels rather than whole-fork
conformance.

`site/src/lib/data.ts` supplies the shared dated development snapshot rendered
by `DevelopmentProgress` on both pages. Its November 23, 2026 snapshot pins merged EL
`d743925e` and CL `eth2030` `fa16f70a`, the selected 33-task delivery, completed
bounded 6h45m05s run,14 BLS windows and actual stopped-head binding. The dated
[reviewed state](reviews/progress-20261123/CURRENT_STATE.json) and
[next steps](plans/glamsterdam/NEXT_STEPS_20261123.md) retain actual integrated CI
failures and original-incident/48h/upstream/public gates. Older400/60-slot,
recovery and544-proof records retain their original source pins. Merge identity,
measured runtime identity, hosted CI and deployed website identity are distinct.

Optional `verification` entries are repository-relative records rendered as pinned
qualification links. Companion-CL-only EIPs can have no local `pkg` implementation
and link their actual CL qualification instead; do not label them missing EL
implementations. Seven additional accepted EIPs do not invent positions in the
historical EF drawing. ePBS remains broadly Substantial while its selected signed
Gloas route is delivered; legacy local Builder extensions were retired in PR 127.
The fast-confirmation and EVM-ASM entries retain Functional framework labels.
These broader mapped-entry labels do not share the selected 33-task denominator.

Update snapshot, EIP evidence, active plans and export assertions together.
The matrix verifier skips the Spec column when identifying Implementation/Tests;
empty companion-CL implementation cells are an intentional regression input.

### Fork placement baseline

Fork placement (Glamsterdam → Hegotá → I* → J* → K* → L* → longer term) follows the
**L1 Strawmap revision of Nov 2, 2026** (strawmap.org). FOCIL and Frame transactions
are Hegotá headliners; native rollups are placed at L*. The active
`docs/ROADMAP.md` now uses this baseline. A development-progress refresh does not
claim a new EF schedule review; re-check placement when adopting a new revision.

The strawmap visual is a Google Drawings embed; to extract it:

```bash
curl -sL 'https://docs.google.com/drawings/d/1GkcGfQv9kxgrYQMyu0BYGcZya0gIDG_wQ7GsFJEsdzY/export/svg?id=1GkcGfQv9kxgrYQMyu0BYGcZya0gIDG_wQ7GsFJEsdzY' -o /tmp/strawmap.svg
# text is baked into paths — render and read it visually:
google-chrome --headless --screenshot=/tmp/strawmap.png --window-size=2979,2272 file:///tmp/strawmap.svg
```

### Entry schema (`EipEntry`)

| Field | Meaning |
|-------|---------|
| `id` | URL slug: `eip-<n>` or `feature-<slug>` |
| `number` | EIP number; `null` for roadmap features (strawmap boxes without an EIP) |
| `layer` | `CL` / `DL` / `EL` / `cross` |
| `fork` | `pre` / `glamsterdam` / `hegota` / `i` / `j` / `k` / `l` / `longer` / `infra` |
| `status` | `complete` / `substantial` / `functional` / `referenced` / `partial` / `planned` |
| `completeness` | 0–100, only where a gap analysis quantified it (11 key EIPs) |
| `priority` | `P0` / `P1` / `P2` from the gap analysis |
| `specUrl` / `specStatus` | Spec link and upstream status (from `refs/EIPs/EIPS/eip-*.md` front matter) |
| `strawmap` | Box label on the strawmap + `SFI`/`CFI`/`EIP` tag + headliner/offchain/crossLayer flags |
| `implementation` / `tests` | Paths **relative to `pkg/`** (verified to exist by the verify script) |
| `verification` | Source-specific repository evidence, linked at the merged revision |
| `implemented` / `gaps` | Deep-analysis content (the 11 gap-analyzed EIPs) |

Status semantics: `complete` = the recorded implementation scope in `EIP_SPEC_IMPL.md`,
with current-spec and independent-client limits retained in `gaps`;
`substantial` = major parts in place (ePBS, FOCIL, execution proofs, binary trie);
`functional` = framework-level roadmap feature with working code; `referenced` =
spec consulted / partial hooks only; `partial` = known incomplete; `planned` =
strawmap box with no eth2030 implementation yet.

## Update workflow (after code changes)

```bash
cd site
node --experimental-strip-types scripts/verify-eips.ts          # cross-check data vs docs + CLAUDE.md + refs/EIPs
node --experimental-strip-types scripts/verify-eips.ts --files  # additionally verify every pkg/ path exists
npm run build                        # type-check + regenerate static pages
node scripts/check-export.mjs         # verify rendered progress against evidence
```

Use Node 22 or later for the TypeScript verifier.

1. Implement/modify EIP code in `pkg/` as usual.
2. Run `verify-eips.ts --files` — fix any drift it reports by updating
   `site/src/data/eips.ts` (paths, status, completeness, gaps).
3. Build, run `check-export.mjs`, and inspect the homepage, `/roadmap` and affected
   `/eips/<id>` pages at desktop/mobile widths. The export check compares the
   shared milestone with the retained live record and checks phase-count parity.
4. Commit everything in **this repo only** (site + docs + code together is fine);
   action-oriented commit message, no AI co-author line.

The verify script checks, in order of severity:

- **Problems (exit 1)**: EIP present in `docs/EIP_SPEC_IMPL.md` matrix but missing
  from the data file; missing implementation entries; status conflicts with the
  `Complete`/`Substantial` lists in `CLAUDE.md`; non-existent `pkg/` paths (with `--files`).
- **Notes (informational)**: upstream spec-status mismatches vs `refs/EIPs`, data
  entries not yet in the doc matrix, spec files not found in refs.

## Cross-repo notes

CL-side EIPs (7251, 7549, 7732, 7805, PQ attestations) are also implemented in the
Lighthouse fork — their eth2030 pages only track the EL-repo/`pkg/` side plus the
shared types. Update `lighthouse/CLAUDE.md` and `docs/LIGHTHOUSE_DEVELOPMENT_ROADMAP.md`
separately in that repo.

## Related documents

- `docs/EIP_SPEC_IMPL.md` — per-EIP traceability matrix (doc-side source)
- `docs/GAP_ANALYSIS.md` — 65 roadmap-item audit
- `docs/EIP_IMPLEMENTATION_GAP_ANALYSIS.md` (workspace root `docs/`) — completeness % + P0/P1/P2 for 11 key EIPs
- `docs/ROADMAP.md` — current development evidence and next acceptance gates
- `docs/PROGRESS.md` — merged implementation and verification boundaries
- `docs/next-stage-work-plan.md` — active implementation and acceptance sequence
