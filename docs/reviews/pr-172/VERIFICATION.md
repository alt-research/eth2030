# Group E verification and scope

Production source: `70facd01d15e8068cfc9839b3f0e352784765277`.
Qualified Go source snapshot: `ee798638b27a186bf168ebb830de4c7f45e5a8e7`.
Baseline: merged PR #171/master `cc31afffaa12919bbc08dc854c03af1e1928dc3c`.

| Check | Observed result |
|---|---|
| Full native Go run | 244 completed packages; 24,267 named pass events; zero failures; 124 optional skips |
| Selected concurrent runtime race run | 76 completed packages; 12,113 named pass events; zero failures; 87 optional skips |
| Native and upstream wrapper builds | Both passed; recorded executable hashes and actual version output |
| `go vet ./...` | Passed; sizing headers use independent copies with empty atomic caches |
| Historical differential baseline | 12 expected, executed and passed; zero failed/skipped |
| Python normal / optimized | 109 / 109 passed |
| Group A source/acceptance | Seven tracked tasks, 26 criteria, 62 mandatory native consumers passed |
| Group B / C / D regression | 79 / 102 / 55 mandatory native consumers passed |
| Group E source/acceptance | Nine tasks, 36 criteria, 39 mandatory native consumers passed |
| Compiled semantic-removal controls | All 36 intended unchanged consumers failed; no build failures counted |
| Website | EIP paths, static build/export, desktop 1440×1000 and mobile 390×844 reviewed; no horizontal overflow |

Full-suite mandatory consumers have zero skips. The B fixture-only `eftest`
encoding consumer is outside selected race packages, explicitly identified in
its report; that is not a skipped execution. Complete optional-skip inventories
and source fingerprints are retained in `evidence/hegota-a` through `hegota-e`.
Expected A/B/C/D oracles and read-only refs are unchanged.

Reproduce the checks from the repository root:

```sh
python3 tools/hegota/check.py
python3 tools/hegota/check_b.py
python3 tools/hegota/check_c.py
python3 tools/hegota/check_d.py
python3 tools/hegota/check_e.py
python3 tools/hegota/e_negative_controls.py --output /tmp/hegota-e-controls
```

The fresh full and race suites were serialized and passed. All 36 controls
compiled and failed their intended unchanged consumers. Three historical
context/static controls now remove both redundant protections: the early filter
and the shared preflight. Single-filter removal alone no longer removes the
behavior. Their exact compiled production/target-test hashes remain unchanged
by the later correction to a separate D native fixture's inadequate fee input.
That original missing-witness error assertion is preserved. The first full
run's fixture failure and the complete fresh rerun are recorded separately;
no independent expected goldens changed. Previous qualification and compiled
pre-fix reproductions are identified separately in the E evidence.
Qualification metadata is recorded after execution; production/test source is
the qualified Go snapshot above.

The completion re-audit reproduced and fixed finalized verdict loss and the
in-memory CL adapter authentication bypass. Native wire tests cover descendant
finalization, restart, stale-head VALID/false without finality rollback, and
invalid checkpoint errors. Memory/disk replacement tests enforce authentication,
atomic failure, exact bytes, timeliness and quota retention. See [REAUDIT.md](REAUDIT.md)
for the complete task review and current discussion decisions.

The subsequent transaction-boundary audit reproduced four further missing
behaviors: final-block state differed from EIP-7805's last-transaction state;
old persisted verdicts retained that wrong predicate; header-only authenticated
snap ancestry was ignored; and definitely invalid Frame contexts demanded an
undefined omitted witness. The exact shared processor prefix, authenticated
post-phase metadata, versioned cache migration, authenticated header walk and
shared immutable Frame context rules correct those paths. Four new mandatory
consumers and eleven new compiled controls exercise the repairs. See
[TRANSACTION_BOUNDARY.md](TRANSACTION_BOUNDARY.md) for all nine task results,
current sources/discussions and explicit migration behavior. Legacy compact
records return SYNCING/null until their exact CL list is resubmitted.

The latest complete-consumer pass additionally repaired pre-frame nonce/fee/gas
checks before missing-witness handling, ordinary state gas and Amsterdam gross
execution in capacity, complete RLP header projection across both dimensions,
version-2 verdict migration and atomic native ingress publication. Actual
nonce/fee/capacity, storage/Frame/restart, filesystem and byte-boundary consumers
failed before repair and pass afterward. Read errors remain errors; preparation
consumes no state, warmth, nonces or gas. The precondition-valid omitted-witness
gate remains explicit. See [RUNTIME_BOUNDARIES.md](RUNTIME_BOUNDARIES.md) for all
nine task results and the latest 22-source specification review.

The native path includes real canonical PQ IL builds, ordinary blob validity
under base EIP-7805, authenticated sidecar recovery, same-ID retry after a failed
build, invalid proof/signature tolerance, and complete appended header RLP
widths. Informational 8369 profile exclusions do not override base FOCIL.

EIP-8369 Profile 2 enforcement remains disabled under D04. Omitted PQ witness
composition, external beacon transport/fork digests and independent CL/ePBS
network qualification remain explicit upstream/HGT-070/080 gates. Local native
qualification does not claim complete public activation, network tail latency,
or live GitHub Pages deployment. The existing Pages workflow deploys after merge.

Hosted exact-head CI results are recorded in [PR #172](https://github.com/alt-research/eth2030/pull/172);
this local report does not predict them.
