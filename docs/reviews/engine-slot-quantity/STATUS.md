# Status

Reproduced and corrected against master 5fedb7e5733fbef69767a711115ef9925e7bd769. EIP-7843 carries a uint64 slot in the Engine payload and header. The existing request validator and independent Geth QUANTITY decoder accept uppercase hex digits; both backends incorrectly required lowercase-only formatted text.

Both the memory backend and actual native Engine HTTP import reject 0x2A before the fix. After accepting hex digit case while retaining the lowercase 0x prefix, minimal representation and uint64 bound, all twenty Engine/Node race suites pass. Tests verify unchanged block/header hash with independent Geth decoding, canonical RPC output, SLOTNUM and persisted restart. Seven malformed encodings remain rejected. Affected vet and both node builds pass. REGRESSION.json retains the actual failing events. Final exact-head hosted CI and conflict checks follow on the independent master-target PR.
