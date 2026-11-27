# Group E verification and scope

Production source: `01c1be41241f32c87b1ced2814debe02b15b7a05`.
Qualified source snapshot: `70eddaad4f4460d10d75d0ff6baba3ab087151d6`.
Baseline: merged PR #171/master `cc31afffaa12919bbc08dc854c03af1e1928dc3c`.

| Check | Observed result |
|---|---|
| Full native Go run | 244 completed packages; 24,245 named pass events; zero failures; 124 optional skips |
| Selected concurrent runtime race run | 76 completed packages; 12,091 named pass events; zero failures; 87 optional skips |
| Native and upstream wrapper builds | Both passed; recorded executable hashes and actual version output |
| `go vet ./...` | Passed; sizing headers use independent copies with empty atomic caches |
| Historical differential baseline | 12 expected, executed and passed; zero failed/skipped |
| Python normal / optimized | 109 / 109 passed |
| Group A source/acceptance | Seven tracked tasks, 26 criteria, 62 mandatory native consumers passed |
| Group B / C / D regression | 79 / 102 / 55 mandatory native consumers passed |
| Group E source/acceptance | Nine tasks, 36 criteria, 34 mandatory native consumers passed |
| Compiled semantic-removal controls | All 26 intended unchanged consumers failed; no build failures counted |
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

The fresh final full and race suites were serialized and passed without changing
independent expected oracles. All 26 controls compiled and failed their intended
unchanged consumers. Earlier attempts are not re-claimed as runs of this snapshot.
An initial full run exposed an old activation fixture that manually inserted an
empty resolved view without the new internal predicate version. Its metadata now
identifies the selected predicate; all activation/reorg assertions remain intact.
Actual legacy migration is covered independently by native wire/restart tests.

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
