# Group D status — PR #171

HGT-030–037 is implemented and locally qualified for the explicitly selected
native profiles. All 8 tasks, 32 criteria and 45 mandatory consumers pass,
including the affected race run. Source implementation commit:
`c449112d2e455eef15546ff46418431cd170daaa`.

The branch starts directly from merged PR #170 at master
`81e63f586642a1406d56db9fa7cbf00941feb7bf`; fetched master still matches that
baseline. No merged Group C commits were replayed. Root trackers, read-only
refs and independent A/B/C expected vectors remain unchanged. The existing
8151 implementation and canonical paymaster are reused.

The [normative plan](../../plans/hegota-d/PLAN.md),
[composition/admission report](../../plans/hegota-d/COMPOSITION.md) and
[verification record](VERIFICATION.md) map the complete work to sources,
production consumers and positive/negative evidence. Machine-readable coverage
and source-bound evidence live in `specs/hegota/group-d.json` and
`evidence/hegota-d/verification.json`.

Local validation: full build/vet/gofmt; 243 full-suite packages, 24,166 named
passes; 68 affected race packages, 10,801 named passes; 102 Python regressions
in normal and optimized mode; 12/12 independent historical baseline cases;
A/B/C/D mandatory consumers 62/79/102/45, zero mandatory skips. Optional skips
are inventoried separately, and all three isolated semantic-removal controls
fail as intended. See the verification record for exact commands and hashes.

These profiles do not schedule a public fork. 8504 provisional constants and
official wire/Engine/CL assignments, incomplete 8288 aggregation, 7807 and
omitted-proof IL composition, ecosystem qualification and the existing canonical
paymaster upstream-adoption gate remain explicit. Native hgtp/1 and the pinned
Gloas CL codec are identified experiments, not official multi-client devnet
qualification. No automatic token compensation is invented.

Published as [PR #171](https://github.com/alt-research/eth2030/pull/171).
Hosted exact-head checks are tracked by the PR's live check suite; offline source evidence does not assert a hosted result.
