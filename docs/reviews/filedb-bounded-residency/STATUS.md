# FileDB bounded residency

This independent capacity correction preserves the current file-per-key and WAL formats, exclusive ownership, committed-intent recovery and fail-closed writes. The existing implementation reads every value into a permanent index and reads the entire WAL at open. The new path will retain bounded cached values and recovery offsets rather than retaining the complete stored byte volume. State/header/finality authentication is not skipped.
