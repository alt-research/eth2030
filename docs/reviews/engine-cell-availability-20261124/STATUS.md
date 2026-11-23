# Status

Implemented and locally validated. The production backend returns a null literal during actual execution sync, after closure, with no cell-serving handler or when the handler reports general unavailability. Healthy pools preserve input length/order and missing-blob positions, including 128 hashes and duplicates. Missing cells cannot retain non-null proof entries. Reads join backend shutdown ownership, without creating a new enable switch or changing activation.

Red regressions through the actual backend and JSON-RPC dispatcher failed on b4b88ed0. Fresh complete engine/backend/node suites, affected engine/backend/node race regressions, real native-node HTTP sync-to-recovery null transitions, native TCP/KZG cell projection/restart and both native/Geth-wrapper builds pass. Authority: Amsterdam Engine API section engine_getBlobsV4 at 1ca08185 and EIP-8070 PR 12701. No new long-duration or original-04 resolution result is claimed. Master-target PR follows.
