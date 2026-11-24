# Status

Reproduced and corrected against master 5fedb7e5733fbef69767a711115ef9925e7bd769. EIP-7843 carries a uint64 slot in the Engine payload and header. The existing request validator and independent Geth QUANTITY decoder accept uppercase hex digits; both backends incorrectly required lowercase-only formatted text.

Both the memory backend and actual native Engine HTTP import reject 0x2A before the fix. After accepting hex digit case while retaining the lowercase 0x prefix, minimal representation and uint64 bound, all twenty Engine/Node race suites pass. Tests verify unchanged block/header hash with independent Geth decoding, canonical RPC output, SLOTNUM and persisted restart. Seven malformed encodings remain rejected. Affected vet and both node builds pass. REGRESSION.json retains the actual failing events. Final exact-head hosted CI and conflict checks follow on the independent master-target PR.

## Whole-upgrade audit

The current-spec review covers all twenty official core entries at the pinned current EIP source, with five networking references separate. See [the complete audit](AUDIT.md), [per-EIP matrix](EIP_MATRIX.json), [source manifest](SOURCE_MANIFEST.json) and [test/skip results](TEST_RESULTS.json). The active-path slot defect is corrected. G01 full native blockchain/BAL corpus, G02 complete CL reference fixture admission, G03 independent mixed-peer qualification and G04 misleading legacy gas-helper authority remain explicitly open. These are not represented as newly passed whole-upgrade conformance. No master merge or public deployment is performed.
