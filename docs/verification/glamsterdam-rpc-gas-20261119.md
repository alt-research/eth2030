# Method specific RPC gas and historical opcode tracing

Date: 2026-11-19. Branch `feat/glamsterdam-rpc-gas`, production source `2893324130e142a9fd800648c8557de9757bb16c`, stacked on Gas draft 83 at `bd0f3357`. [Versioned implementation policy](../plans/glamsterdam/RPC_IMPLEMENTATION.md).

## Reproduced defects and actual consumers

Real Node HTTP reproduced four independent failures: call GAS omitted intrinsic costs, storage creation estimated 21000, traceCall returned an empty fabricated step list with remaining gas reported as usage, and creation calls returned empty output without executing initcode. Historical transaction tracing also replayed raw unconfigured calls and guessed intrinsic costs; block opcode tracing interpolated receipts instead of executing.

All actual call/estimate/access-list paths now share the selected ordinary transaction transition, full block/hash/slot/blob context and original-state/transaction boundaries. Explicit simulation skips sender EOA and nonce checks. Calls use all supplied gas after intrinsic cost with zero initial reservoir; estimates and access-list discovery use the capped execution dimension and overflow reservoir. Ordinary consensus entrypoints always retain the original zero-valued internal policy. Full call arguments preserve access-list costs, dynamic fees and signed authorization entries. State/code/balance/nonce overrides apply only on isolated state; complete storage replacement clears omitted slots, and state/stateDiff are mutually exclusive. A backend unable to enumerate complete storage returns an explicit error instead of silently treating replacement as a merge.

Actual HTTP controls verify uncapped GAS above 2^24, an impossible transactional computation rejected by estimation, exact slot/account creation estimates, equal zero/nonzero calldata floors, flat access-list bytes, a signed new-authority estimate, 64 KiB creation output, real opcode steps and caught inner failure. Successful outer execution is not represented as proof of inner success. Complete override and estimation state does not leak into the canonical chain.

A signed two-transaction block creates and then reuses a storage slot. Historical transaction trace returns the target's actual 17105 paid gas and four opcode steps after the correct prefix; the first transaction pays 125025. Actual traceBlockByHash and traceBlockByNumber replay the prefix once and return both real traces. A second block reads EIP-2935 and its trace returns the first block's hash, requiring current pre-block system replay. Explicit canonical system assets precede genesis roots. Head, balances and nonces remain unchanged by replay.

A separate actual fork-boundary Node verifies Osaka historical queries still charge 21000 intrinsic after the head activates Amsterdam's 15000 zero-value call pricing. The pre-existing PQ RPC control retains its cryptographic outputs and now funds intrinsic plus the calldata floor in addition to its original precompile allowance.

## Source and qualification boundaries

The current EIP-8037/8038 execution source remains pinned at EIPs `3db38a3`. The method policy follows merged Geth [35845](https://github.com/ethereum/go-ethereum/pull/35845), merge `df4c5248`, including its reviewed consecutive-uncapped-call reservation correction for the separate future simulation work. Execution-apis [977](https://github.com/ethereum/execution-apis/issues/977) remains open with no comments. This is an explicit development policy, not an adopted cross-client API decision.

Execution-apis [852](https://github.com/ethereum/execution-apis/pull/852), head `cc1852a7`, and [976](https://github.com/ethereum/execution-apis/issues/976) remain open. New two-dimensional trace fields, complete nested callTracer hooks and eth_simulateV1 are separate unfinished work; opcode replay here does not imply those surfaces are qualified. Unqualified experimental transaction envelopes return an explicit tracing error.

All 212 Go packages and 27 tagged Core/EIP packages pass, with 48 affected race packages, vet and local build. Eleven compiling mutations detect call cap, intrinsic debit, estimation transaction policy, access-list input, authorization input, complete storage replacement, opcode hooks, historical prefix, pre-block callbacks and historical fork context. Sources are restored before the full qualification run. [Exact results](glamsterdam-rpc-gas-results-20261119.json).

Local logs: `/tmp/glamsterdam-rpc-gas-{red-http,full-second,tagged-second,race-second}.log`; targeted consumer logs retain the expanded argument, historical and system controls. Independent complete-client/API fixtures, performance/resource limits, simulator/nested tracing and final release decisions remain open.

## Access discovery follow-up

The actual HTTP consumer returns the storage access list with exactly 131253 paid gas. An above-cap computation fails in access discovery even though uncapped eth_call can run it. An eleventh compiling mutation changes this path to call policy and is detected. Five affected race packages, affected vet, build and restored-source actual HTTP tests pass. Durable follow-up logs are `/projects/eth2030/glamsterdam-rpc-access-followup-race.log`, `/projects/eth2030/glamsterdam-rpc-access-policy-mutation.log` and `/projects/eth2030/glamsterdam-rpc-access-followup-final.log`. Earlier temporary logs were runner-local; their structured results remain archived.

[Feature review 85](https://github.com/alt-research/eth2030/pull/85) is submitted. Current merged LIFO is explicitly selected for development by the user, with final design review retained. Simulator, nested trace dimensions and remaining full-client/resource/release work continue separately.
