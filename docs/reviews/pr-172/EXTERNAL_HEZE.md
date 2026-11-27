# Independent Heze network acceptance

This remaining-delivery pass starts from published head `4aaaa365`. It selects
the existing independent Lighthouse implementation at
`jiayaoqijia/lighthouse:bc56927fdb0c31f908286ec39013e4452864dc18` and the existing
ethereum-package pin `c4a8e15d40f0965f3728f776d72f1954440ad96b`. Neither source is
patched by the acceptance job. The native EL is built from the exact tested
GitHub source. Image IDs and all three source identities are retained.

## Normative contract

The selected EIP-7805, Heze inclusion-list/fork-choice/optimistic-sync and Engine
Bogota documents remain unchanged. EIPs master advanced to `4e55e80d` with an
EIP-8425 recovery-proof update; that does not define an EIP-8369 enforcing
extension or an EIP-8504 omitted-witness composition. The latter prerequisites
remain open. No new consensus fields, claims, committee bounds or fork dates
are assigned by this test.

The new private-devnet proxy forwards actual native Engine responses unchanged.
For a negative control it removes the exact fixture's builder IL view while
retaining the original validation list. This produces a real execution-valid
payload that omits an includable transaction. Acceptance requires both native
nodes to return `VALID/false`, both independent CLs to retain execution validity
and the unsatisfied beacon root, and later bids not to extend that execution
payload as FULL. The CL decision must also survive a real CL process restart.
An empty-parent beacon child is permitted by the selected ePBS rules.

Independent positive controls also require actual VALID/true decisions for
malformed-envelope and future-nonce ILs. A capacity control includes a separate
public-account-2 transfer, omits an otherwise valid near-block-limit candidate,
and checks the actual imported blocks: the filler is present, the candidate is
absent, and its declared gas exceeds the remaining gas. Both CLs must retain
the same execution-valid, inclusion-satisfied root. The near-limit ordinary
envelope remains valid under current EIP-8037; the native fixture consumer
executes it successfully with full capacity and rejects its reservation after
the independently executed filler.

The positive control requires actual BLS-signed gossip between two independently
run CLs, an isolated proposing EL that never received the transaction in its
pool, a successful included transaction, and common finality on both pairs.
The public-key-1 fixture is bound to the isolated chain ID 3151908. Its new
recipient must pay current EIP-8037 account-creation state gas: the former
21,000-gas fixture is replaced by a 250,000-gas signed transfer. A native unit
consumer independently executes it and checks the actual 183,600 state charge.

## Bounded PQ transport measurement

Canonical direct-8504 transaction bytes are exported from the existing real
fixture. The test fills at most 8192 transaction bytes with repeated occurrences
and lets real validator clients sign and gossip those lists. This respects the
Heze rule that IL gossip does not perform execution validity checks. The fixture
is not claimed to be a valid transaction on this private chain.

Measurements start at the source Engine callback's availability and end at the
peer CL's authenticated inclusion-list event. All processes share one Linux
host's monotonic clock. Each producer direction needs at least eight distinct
signed lists, with at least 32 matched observations in total. Missing peer lists,
different bytes/signatures, negative clock intervals or insufficient samples
fail the run. These are test-quality thresholds, not protocol constants.
The report retains p50/p95/p99/max, exact transaction and signed-SSZ sizes, raw
observations and Engine journals. Repeated bytes measure carriage, not multiple
independently spendable transactions. The dependency witness remains outside
the canonical envelope, and no omitted-witness transport is created.

This is a bounded Docker-host network measurement. It does not qualify public
network tail latency, full PQ validity/replay, Profile 2, mixed-client coverage,
or every external beacon request protocol. Existing explicit release gates
remain until those separate prerequisites are actually demonstrated.

## Execution and evidence

The hosted job is defined in `.github/workflows/hegota-e-interop.yml`. It builds
the unchanged independent CL and the actual native node, uses a qualification-
only proxy image, and runs `tools/hegota/run_external_heze.py`. It retains the
launch log, actual source/image identities, positive/negative observations,
CL snapshots, Engine responses, resource samples and owned-resource cleanup.
Failure is a failure, never a successful unsupported/skip result.

The current local machine lacks Docker/Kurtosis and the sibling CL checkout.
Hosted Linux execution is therefore required. Local unit/build checks qualify
the tools and fixture only; network success is recorded after the actual job.

The first complete launch attempt failed at the native RPC exposure guard,
before network cases could run. The old shared profile had no explicit
private-RPC compatibility choice. The new runner generates an owned profile
with the documented private-devnet flag required by ethereum-package's
admin_nodeInfo health checks; the native default guard and Engine JWT are
unchanged. The failed attempt, exact artifact identity and verified removal
of its enclave/containers/volumes are retained in
`evidence/hegota-e/external-network.json`.
