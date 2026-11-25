# Current runtime routes and activation boundaries

HGT-004 / HGT-005. Baseline: integrated master `89b3b1f2`; current source fingerprints and executed tests are in [runtime evidence](../../../evidence/hegota-a/runtime.json), [verification](../../../evidence/hegota-a/verification.json) and the [criterion matrix](../../../specs/hegota/acceptance.json). The pre-merge route table is preserved [at its historical revision](https://github.com/alt-research/eth2030/blob/6a17479/docs/plans/hegota-a/runtime-and-activation.md).

| Entrypoint | Actual route | Evidence and limits |
| --- | --- | --- |
| `pkg/cmd/eth2030` | `node.New` -> native `core/chain.Blockchain` / `core/block.BlockBuilder` -> `core/execution.StateProcessor` -> native VM/state | `TestNativeRuntimePathAndVersionedBogotaAPI` records the injected `*backend.EngineBackend`. Canonical Frames use `executeFrame8141`. |
| Native Engine | Dispatcher -> `node/backend.EngineBackend` -> block processing / deferred Bogota validation / FCU | `TestBogotaDeferredWireAcrossRestart`, `TestCanonicalFrame8141ThroughEngineAndRestart`. NewPayload V6 and FCU V5 exist; malformed calls reject parameters and unknown V99 returns method-not-found. |
| Canonical selection | Native FCU -> validated canonical head/state -> pool `Reset(state, header)` and fee refresh | `TestNativeForkchoiceRefreshesFrameActivationAndReorg` uses actual selection/rewind. Its explicit CL-view fixture is not an independent CL interoperability result. P2P announcements cannot select an optimistic PoS head. |
| Canonical pool | `TxPool.Add*` -> canonical validation -> `ValidateFrame8141Prefix` on duplicated state -> shared `ConfigureBlockEVM` | `TestCanonicalFramePrefixUsesHeadBlockPrerequisites`: London at height 10, heads 9/10/11, rewind to 9, recovery to 10. Preserve head number, time, fee and gas limit; live state stays unchanged. |
| RPC | `rpc.Server` -> `node/backend.RPCBackend` -> real chain/pool or shared native simulation | Reuse master's shared simulation/override/cancellation route. Processor tests do not become HTTP/CL evidence. |
| Helper Engine | `engine.NewEngineBackend` -> its own builder/state-copy/processor orchestration | Separate backend; missing IL evidence returns unavailable/error. Never substitute its result for the native node. |
| Geth wrapper | `pkg/cmd/eth2030-geth` -> upstream `node.New`, `eth.New`, `catalyst.Register` | Distinct binary, executor and precompile context from native and direct reference. |
| Historical reference | Direct `go-ethereum@v1.17.10/tests.StateTest.RunNoVerify` | No ETH2030 wrapper or added precompiles. Twelve pinned cases compare native/reference and independent root/log/gas oracles. |
| Docker | Builds both binaries; `ENTRYPOINT ["eth2030"]`; `geth` symlink points to native | Dockerfile and host binaries verified; image execution/digest and CL/devnet qualification remain separate. |
| Kurtosis | Scripts -> network configuration -> ethereum-package/images | Existing experimental presets are not an officially qualified combined Hegota network. |

| Configuration / boundary | Consumers and admission | Evidence |
| --- | --- | --- |
| Block forks plus merge/time | `ChainConfig.Rules(number, merge, time)` -> shared VM table/precompiles | Use actual head number; timestamp alone cannot select post-merge VM rules. |
| Prague / Osaka / Amsterdam / Bogota | Backend predicates stay monotonic; dispatcher validates method versions/schemas | FCU V3 is reused after Prague/Osaka; Bogota still requires Amsterdam rules. Do not restore exclusive predicate intervals. |
| `glamsterdamTime` / `glamsterdanTime` | Normalize to stored legacy field; unequal aliases reject | `TestHegotaGenesisAliases`: legacy, canonical, equal, conflicting. |
| `bogotaTime` / `hegotaTime` / `hogotaTime` | Normalize to `HogotaTime`; no invented mainnet date | Alias tests and symmetric retroactive schedule compatibility checks. |
| Canonical `FrameTx8141` | Direct typed dispatch and Bogota public-prefix validation | Independent of `ExperimentalLegacyFrames`. Dedicated canonical processor and pool tests cover timestamps 99/100/101, pre-mutation rejection and reorg eviction; native semantics and Engine/restart tests retain explicit scope. |
| Frozen legacy `FrameTx` | `ExperimentalLegacyFrames && merge && London(number) && HogotaTime<=time` | Pre-mutation rejection, exact activation, VM isolation and reorg eviction. Canonical Frames survive legacy-profile resets. |
| Legacy AA / LocalTx / STARK | AA default off, LocalTx opt-in, unavailable STARK proof provider fails node creation | `TestAATxDisabled`, `TestLocalTx_DefaultConfigDisallows`, `TestStarkFrames_EnabledRequiresProvider`. |
| Conditional nonce/root/PQ | Explicit profiles and D01–D09 decisions | Isolated research oracles do not activate the combined release. |

Head changes refresh fees before revalidation. Historical genesis/database names stay readable; incompatible prototype behavior is not silently reclassified as canonical compatibility. The pool uses one stored fork context and existing production setters/reset rather than a second test-only API.
