# Checked-native-commit recovery control completed

Normal New still opens its original FileDB through an internal instance-local opener. The explicit test-only fixture holds one fully checked new epoch block before atomic commit, while the real native import mutex remains owned. There are no production flags or changed validation/deadlines. Opener failure, real read/publication behavior and post-release VALID are tested;10 race repetitions, all219 native suites, affected vet and the ordinary native build pass.

The owned full CL/native control deliberately held canonical block83061/slot85696 for150.000549s, then recovered to canonical83113/slot85748 with independent Geth header/state-root agreement and both exit0/noOOM. Actual blocking Go and CL frames are retained privately; CONTROL_RESULT.json records exact identities and early-debugger interference. All signed/hash/state/BAL guards and2CPU/4096000000B/8s limits remain in place.

This result is not original04 root closure. The injected holder is known by design, and the simple finite hold recovered. The earlier response-only case had a parallel lookup rescue of an already-executed payload. Keep both controls distinct from the unchanged long-running original source diagnosis and its unresolved5.5-hour wait. See RUNBOOK.md for explicit invocation and boundaries.
