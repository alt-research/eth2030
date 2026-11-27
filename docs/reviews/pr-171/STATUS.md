# Group D status — PR #171

HGT-030–037 is implemented and locally qualified for the explicitly selected
native profiles. The latest-EIP re-audit covers all 8 tasks, 32 criteria and
55 mandatory native consumers. Production implementation commit:
`d7fa00f09bd0d4dc17d14fc8fb76c9d6695f6151`.

The branch starts directly from merged PR #170 at master
`81e63f586642a1406d56db9fa7cbf00941feb7bf`; the final remote check still
matches that baseline. No merged Group C commits were replayed. Root trackers,
read-only refs and independent A/B/C expected vectors remain unchanged. Existing
8151 and canonical-paymaster implementations are reused.

The [plan](../../plans/hegota-d/PLAN.md),
[composition report](../../plans/hegota-d/COMPOSITION.md),
[re-audit findings](../../plans/hegota-d/REAUDIT.md) and
[verification record](VERIFICATION.md) map every criterion to normative sources,
production consumers and positive/negative evidence. The fixed mandatory
register and source-bound evidence are in `specs/hegota/group-d.json` and
`evidence/hegota-d/verification.json`.

The re-audit fixed four reproduced boundary failures: ASSERT trace cache
amplification/stale mutation indexes; CL witness/header commitment loss;
unsupported synthetic proof, hash-only guest and STF input acceptance; and
queued/sequential/batch sync delegation before witness validation. Additional
native tests cover real blob/KZG composition, creation/adoption/destruction,
actual LOG3/event copying and paid body rollback. Nine isolated semantic-removal
controls compile and fail the intended unchanged native tests.

Current-source validation passed: full build/vet/gofmt; 243 full-suite packages,
24,176 named passes; 71 affected race packages, 11,535 named passes; 102 Python
regressions in normal and optimized mode; 12/12 unchanged independent historical
baseline cases; A/B/C/D mandatory consumers 62/79/102/55, zero mandatory skips.
Optional skips are separately inventoried. The original 45-consumer/c449112d
qualification and earlier hosted results are immutable predecessor provenance,
not substituted for these fresh runs.

The final EIPs master check is `68db81b0a75edb0e16d8f181241f994875eba52d`;
its four newly added drafts change no pinned Group D normative file.
EIP-8504/#12672 remains at the pinned proposal head. Prospective EIP-8476
composition with 8151/8298 requires upstream agreement and is not activated.

These profiles do not schedule a public fork. Provisional 8504 constants and
official wire/Engine/CL assignments, incomplete 8288 aggregation, 7807 and
omitted-proof IL composition, external ecosystem qualification and canonical
paymaster upstream adoption remain explicit release gates. Native hgtp/1 and
the pinned Gloas codec are identified experiments. No automatic token
compensation or unsupported validity proof is invented.

Published as [PR #171](https://github.com/alt-research/eth2030/pull/171).
The English PR description and exact-head check suite record final hosted
results after this offline evidence freeze. This record claims the local runs
above; it does not infer hosted success from an earlier commit.
