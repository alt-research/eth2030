# Group E verification and scope

Production source: `d830d0b1c82373cd3b9f6ba135f27703405fe972`.
Qualified source snapshot: `296b6fe3f2bc009898756f84dfae0c19ca7dbfe4`.
Baseline: merged PR #171/master `cc31afffaa12919bbc08dc854c03af1e1928dc3c`.

| Check | Observed result |
|---|---|
| Full native Go run | 244 completed packages; 24,234 named pass events; zero failures; 124 optional skips |
| Selected concurrent runtime race run | 76 completed packages; 12,080 named pass events; zero failures; 87 optional skips |
| Native and upstream wrapper builds | Both passed; recorded executable hashes and actual version output |
| `go vet ./...` | Passed; sizing headers use independent copies with empty atomic caches |
| Historical differential baseline | 12 expected, executed and passed; zero failed/skipped |
| Python normal / optimized | 109 / 109 passed |
| Group A source/acceptance | Seven tracked tasks, 26 criteria, 62 mandatory native consumers passed |
| Group B / C / D regression | 79 / 102 / 55 mandatory native consumers passed |
| Group E source/acceptance | Nine tasks, 36 criteria, 28 mandatory native consumers passed |
| Compiled semantic-removal controls | All 13 intended unchanged consumers failed; no build failures counted |
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

The initial simultaneous full/race/build run hit the existing one-second native
commit-barrier timeout. That unchanged test passes in both final full and race
runs. Its failed log identity remains in the E report; superseded intermediate
runs are not used for final source qualification.

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
