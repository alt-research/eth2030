# Shared foundation

This extracts existing value/journal paths and per-operation materialization without changing behavior. The real FCU failure fixture derives its target from the actual writer path. The persistence-performance and directory-layout branches share this parent so each feature remains independently reviewable against master without moving duplicate deletion logic into conflicting patches. The complete rawdb/chain race suites and explicit native cmd/eth2030 plus Geth-wrapper builds pass. Paths and operation semantics remain unchanged.
