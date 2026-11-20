# Corrected and qualified for review

An actual OS EISDIR materialization fault after the durable WAL commit originally returned an Engine error followed by SYNCING. Check durable head readability before Engine cached/boolean lookup shortcuts and direct chain forkchoice. Fenced or unreadable storage now keeps ErrStorageFailure; it never becomes peer synchronization or cached VALID.

The real isolated-process campaign passes missing payload authority, bad-slot and duplicate handling, same-height reorg, finality protection, actual materialization fault/fenced retry, SIGKILL/WAL recovery and normal indexed restart. Original failed process inputs remain retained. The corrected composition e61ea843 passes all 218 full package suites; backend/chain/node/engine normal and race tests, scoped vet and native build pass. RESULTS.json records exact scope; remaining independent CL/deployment/pressure campaigns are not relabeled completed.
