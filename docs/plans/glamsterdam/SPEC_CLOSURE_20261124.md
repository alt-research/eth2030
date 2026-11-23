# Current Glamsterdam specification closure - November 24, 2026

## Authority and scope

This refresh follows the current [EIP-7773 inventory](https://github.com/ethereum/EIPs/blob/81a90efc672bdd6130c813d8dae005e5b3a8ce5b/EIPS/eip-7773.md): 20 scheduled EIPs, five networking EIPs and two informational documents, plus the meta document itself. The [28-document matrix](../../reviews/glamsterdam-spec-closure-20261124/MATRIX.md) and [exact blob comparison](../../reviews/glamsterdam-spec-closure-20261124/SPEC_COMPARISON.json) retain every entry. SSZ dependencies 7495/7916 already had production CL consumers; their explicit meta placement does not create new missing-code credit.

Gas remains the current merged LIFO text, blob-identical to the selected prior revision. Reservoir alternatives 12581/12582 closed unmerged. Amsterdam Engine clarifications and approved EIP-8070 changes are incorporated; the current Gloas signed consensus contract is unchanged, while gossip quota/authority/timing checks are completed. No public activation parameter is invented or modified.

## Implemented follow-ups

| Review | Concrete before/after |
| --- | --- |
| [EL 150](https://github.com/alt-research/eth2030/pull/150) | Ordinary eth/72 announcements sent/accepted an empty mask; every message now encodes/requires B_16 while ordinary bits are ignored and eth/71 remains compatible. |
| [EL 151](https://github.com/alt-research/eth2030/pull/151) | V2/V3/V4 distinguish sync/closure/general unavailability from healthy missing positions, validate DATA_32/counts, retain source proof versions and return owned stored cell proofs. Missing cells cannot retain proofs; legacy sidecars are not upgraded by a query. |
| [Lighthouse 16](https://github.com/jiayaoqijia/lighthouse/pull/16) | Replace one-bid booleans with pending-reservation-aware counts, atomically admit after real BLS, resolve the correct parent state and process slots before authority checks, and apply the normative clock-disparity window. |

Five confirmed problem groups are covered by these three feature reviews; this is not a new denominator replacing the historical 33-task/5-follow-up ledger. No confirmed implementation task in the reviewed current Glamsterdam set is left without a delivered change or retained production consumer. Pending review/merge and qualification results remain explicit.

## Validation and remaining actions

The [validation record](../../reviews/glamsterdam-spec-closure-20261124/VALIDATION.md) separates new code tests from retained measurements: 219 Go test packages (18 no-test packages separate), five full race suites, full vet/build, actual affected HTTP/TCP/SDK/KZG/restart paths, actual Lighthouse binary typecheck and 59 Gloas bid tests with real BLS. All source identities and failed attempts are retained. The current EL reviews have passing hosted CI; final Lighthouse hosted/integration status is recorded on the PR and in the current review status rather than inferred from this initial snapshot.

Next actions are final hosted CI, the already-authorized Lighthouse merge into eth2030, and review/merge of independent master-target EL/docs PRs. No EL master merge is performed by this task. The cumulative merge simulation verifies compatible code; it does not merge branches.

48h/larger-scale admission and upstream Lodestar approval remain excluded at the user's direction and are not declared passed. Original 04 attribution stays unknown. Public-network rollout is a separate decision. The broader roadmap's Frame/FOCIL, canonical proofs, PQ, binary-state and native-rollup research do not become implicit Glamsterdam obligations merely because they have old tracker labels.
