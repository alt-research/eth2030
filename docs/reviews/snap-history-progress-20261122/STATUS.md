# Historical prefix validation blocks native imports

A deterministic mature-state replay starting at the actual failed04 empty block77623 failed the8s import budget. Its before-cleanup goroutines show InsertSyncedBlockWithResult waiting on insertMu while the startup snap-history worker traverses the entire saved prefix under that mutex. This is a real reproduced wait, not a transaction loop or a guessed lock cycle. SIGTERM cancels this worker, explaining how this mechanism can release pending imports on stop; it does not independently prove it was the sole cause of the original multi-hour incidents.

The fix must snapshot the already-finalized pivot/frontier under the mutation lock, then authenticate the immutable hash-linked prefix outside it. All existing header/body/BAL/retention/canonical checks remain required; downloaded history must never acquire execution authority. StageSnapHistoryParent and CompleteSnapHistoryBlock retain their serialized commit/check operations.
