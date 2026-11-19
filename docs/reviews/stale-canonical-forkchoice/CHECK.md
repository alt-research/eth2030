# Recovering CL stale canonical forkchoice

- [x] Read EIP-3675 and inherited Engine forkchoice rules and retain the actual recovering Lodestar failure.
- [x] Reproduce a restarted CL nominating older validated canonical heads/checkpoints against persisted newer EL finality through actual HTTP.
- [x] Implement the permitted stale-canonical skip without regressing durable head/finality or accepting foreign/unknown checkpoints; preserve real reorg/build semantics.
- [ ] Run affected/full/integration/race/vet/build checks, independent compatibility controls and publish a separate correction.
- [ ] Recover the original independent live client using the qualified native artifact, then resume the full campaign.
