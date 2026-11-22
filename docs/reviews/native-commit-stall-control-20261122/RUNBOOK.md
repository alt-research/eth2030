# Checked native commit pause and recovery

## Why this control exists

Original04's empty block77623 did not finish before its Engine8s deadline and was only logged accepted after native SIGTERM. Its real waiting owner remains unknown. A single response-error experiment was not equivalent: the payload was already executed, and a parallel lookup imported its envelope before the delayed gossip request errored. A beacon head that advances alone is also insufficient recovery evidence.

This fixture holds the actual checked native block batch before its atomic Write. The native insert mutex remains owned and the validation marker is unpublished; newPayload/FCU, unrelated reads, pending worker lifetimes and real CL behavior are exercised. It preserves normal execution, header/state/receipt/requests/BAL checks and default Engine8s deadlines. The known artificial holder is test instrumentation, not a claimed original root cause.

## Implementation and default behavior

`New` delegates to an internal instance-local database opener and still always opens the same FileDB. There is no new production CLI flag, environment setting, global hook or alternate production database. Internal tests can wrap the same FileDB before chain/Engine/P2P wiring; storage fencing and batch reset semantics are retained.

Only the test binary accepts `-native-commit-stall-config`. Its diagnostic test visibly skips without an explicit manifest. Two ordinary tests always run: opener errors retain their identity, and a real fully checked block reaches a controlled barrier with no validation/canonical publication while an unrelated upcheck remains readable; releasing it permits actual VALID and marker publication. The builder prevalidates its candidate, so that unit control arms after building and resets only its own candidate marker before the incoming Engine call. A manual unit barrier avoids timing-based assertions.

The batch wrapper selects a new validated epoch-boundary header at/after the armed slot. It pauses exactly once, records blocked/released/owner-aborted phase atomically, then invokes the unchanged underlying FileDB batch Write. It bounds the hold to180s and releases on owner shutdown; an aborted hold is not a successful recovery. Nonsecret status is readable inside a private output directory. A test-only goroutine profile uses the existing optional metrics listener; no listener is added to ordinary nodes.

## Explicit diagnostic input

```json
{
  "datadir": "/snapshot/execution-data",
  "genesis": "/genesis.json",
  "jwt": "/jwtsecret",
  "bootnodes": "QUALIFIED_EXISTING_REFERENCE_ENODE",
  "arm": "/control/arm.json",
  "status": "/control/commit.json",
  "hold_seconds": 150,
  "disposable": true
}
```

Compile from the candidate and run only against a new owned disposable database/container with2CPU/4096000000B:

```sh
cd pkg
GOTOOLCHAIN=go1.26.11 go test -c -o /absolute/qualification/native-commit-stall.test ./node
/absolute/qualification/native-commit-stall.test \
  -test.v -test.run='^TestNativeCommitStallDiagnostic$' -test.timeout=70m \
  -native-commit-stall-config=/absolute/fixture.json
```

Complete the startup history worker and reach real non-optimistic synchronized readiness before writing `{"minimum_slot": CURRENT_VERIFIED_HEAD_SLOT}` to the arm file. Keep a reference chain and full CL BLS active. Record actual executable/start identities, exact input source and every injected phase. Capture both native goroutines and CL threads, marking debugger interference. Release, then require CL health and actual canonical execution number/slot progress past the held block with independent Geth hash/state-root agreement. Stop only the owned control and retain cleanup/failures. Never restart the separate original-source long diagnostic for this experiment.

## Measured control and limits

The original CL57ccb466/blst source with the test fixture held real canonical block83061/slot85696/hash e4d3d1d... for150.000549s after full checks and before commit. Go stacks show commitStallBatch.Write -> Blockchain.insertBlock -> Engine newPayload while insertMu is retained. CL/Go snapshots were taken early in the deliberate hold, before the first8s timeout; this is diagnostic interference and is not a post-timeout or clean operating claim.

After release the CL became synchronized/non-optimistic/online and the actual native canonical head advanced to83113/slot85748. Its hash and state root agreed with reference Geth; both daemons stopped exit0 with noOOM. The control used the normal8s Engine deadline and original2CPU/4.096GB limits. Exact identities are retained in CONTROL_RESULT.json. This finite deliberate commit stall recovered under the observed control conditions; it does not explain the original indefinite waiting dependency or certify48-hour stability.

Two targeted real-node tests pass,10 race repetitions and all219 native package suites plus affected vet pass, and the ordinary native CLI builds. The fixture is a reusable causal recovery experiment; actual original04 root attribution remains open in the independent long diagnosis.
