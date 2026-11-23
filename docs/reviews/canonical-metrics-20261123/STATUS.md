# Canonical progress metric repair

The fresh native observer advanced RPC canonical heads while `ETH2030_chain_height` remained zero. Staged imports correctly avoid publishing a head, but successful Engine forkchoice and checkpoint adoption omitted the height gauge. This observation does not establish an execution stall or the original04 waiter.

The gauge now follows successful canonical publication in Engine forkchoice, checkpoint install/restore, startup, legacy rewind and reorg. The durable Engine callback updates it alongside the selected head; rejected heads and failed durable materialization retain the previous gauge. Staging does not advertise unselected blocks. Counter semantics are unchanged.

Normative basis: pinned Engine Paris forkchoice rules4-6 (`refs/execution-apis/src/engine/paris.md`), particularly validated heads and atomic publication; no validation or forkchoice decision changes.

Before the patch both new regressions failed: published head0 versus3 and restored head0 versus6. Afterwards the complete core/chain race suite passes, and the final added legacy rewind plus real FileDB failed-materialization gauge controls pass under race. Full `go test ./...` also passes. Native `GOTOOLCHAIN=go1.26.11 go build ./cmd/eth2030-geth` passes. Tests inspect staged/committed state, failed durable publication and downward rewinds rather than merely calling Gauge.Set.

The running observer uses the earlier native binary. It cannot count as a runtime measurement of this patch. Metrics use the existing process-global registry; independent test chains within one process continue sharing it. No new production registry architecture is introduced.
