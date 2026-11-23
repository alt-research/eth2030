# Current adopted-scope audit

The review starts at merged master cfd39f3f and compares pkg tree 48afbc345e9418d562be901c13e2e54eaae500b9 to the actual cumulative runtime 50514bf3: equal. PR 142-147 merge records are live-rechecked and retained in CURRENT_STATE.json. No new runtime measurement is claimed by this review.

Search covered non-test engine/epbs/bal/node/core-chain/sync source for TODO/FIXME/unimplemented/placeholder/stub terms and followed the identified production consumers:

| Hit | Actual consumer finding | Disposition |
| --- | --- | --- |
| epbs/auction signature placeholder | NewAuctionEngine has no production constructor caller in this graph. Standard signed bids/registry/collateral use CL and EIP-8282; legacy Engine extensions are retired. | Unadopted helper scope; no bypass in the selected signed path. |
| engine/blobsbundle placeholder sidecar helpers | PrepareSidecars/GetSidecar have no non-test production caller. Actual Engine publication uses payload.BuildBlobsBundleV1/V2 and real proofs. | Do not advertise the unused helpers as production KZG consumers. |
| lighthouse_compat progressive inline IL TODO | This shape is part of later-fork compatibility; the adopted native Bogota route requires its selected contract before parsing. | Later-fork target selection, not a missing frozen Glamsterdam consumer. |
| lighthouse_compat custody TODO | Actual FCU handler calls SetCustodyColumns through EngineBackend/EthHandler to sparse relay. | Broad stale helper comment; actual serving consumer is wired. |
| Node sampler/beam stubs | beamSync has no production method consumer beyond construction; the optional independent EL sampling mode is outside this qualified graph. | Research integration remains separately scoped. |
| concurrent-healer placeholder comment | Describes re-creating an in-memory task moved out by ProcessBatch. | Normal scheduling bookkeeping, not missing validation. |

A text hit alone does not establish a live missing requirement. The selected finite evidence remains bounded; inactive handlers and skipped BLS cases are not executed passes. No claim that the entire client is bug-free is made.

Activation inspection follows IsAmsterdam/IsGlamsterdan, genesis fields and applyForkOverrides, CLI override parsing and persisted compatibility/restart tests. The mainnet preset has no Amsterdam/private Glamsterdan activation timestamp. Support for selecting and persisting a schedule already exists; no public network was selected for deployment. The current EIP7773 activation table is consulted as network-level planning, not silently adopted as a new local configuration.
