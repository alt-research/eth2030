# Bounded value residency without a format change

FileDB retains the startup-discovered key/filename metadata and a16MiB accounted value cache. Directory entries are scanned in batches of128; startup does not retain every value. Cache results and eager prefix iterators preserve independent byte copies and snapshot behavior. The exact existing data files and WAL record format remain supported.

WAL scanning retains the last committed per-key operation plus its file offset/size. Only final values are read for materialization; committed-prefix, unknown/torn suffix, directory sync, atomic batches and failed-write isolation retain their existing rules. Opening no longer holds all historical WAL value bytes. Working memory still includes key metadata, final-intent offsets, one decoded value and explicitly requested iterator output; this is not a claim of constant total memory at arbitrary cardinality. Header/trie authentication remains complete and is never skipped.

A filesystem-only key lookup experiment was rejected: loss of a previously indexed committed file must be a read error rather than an ordinary missing/zero account. The retained metadata preserves that distinction. Cold persisted read failures fence writes until reopen. Optional error carriers expose iterator/store errors without breaking existing in-memory/custom iterator interfaces. Table and snapshot adapters, migration/pruning, flat-state reads/root commits and typed RPC getters propagate those errors. A required committed code read also checks the Keccak code commitment. Copy/revert cannot erase an I/O error.

The real temporary-directory tests cover cache eviction/copying, atomic overwrite/delete/put, empty values, reopen, immutable iterator snapshots, lost committed account/slot/code, changed code hash, error forwarding and empty-root snapshot/migration publication. Two compiling behavioral mutations disable the snapshot error guard or code-hash check and are detected. The first code-hash mutator's unused-import build failure is retained separately and is not counted as a successful mutation test.

BASELINE_PROFILES.json and CANDIDATE_PROFILES.json use separately prepared byte-identical synthetic fixtures. The640MiB-class value volume and412,201,000-byte WAL profiles check every key/value and full fixture digest. Preparation is outside the300-second reopen budget and1GiB RSS ceiling. Both baseline and candidate fit the declared budgets; the baseline still demonstrates linear stored-value residency and whole-WAL transient growth. The candidate materially reduces these allocations. Cold full-dataset reads do incur additional I/O and are reported, not hidden. These are isolated KV measurements, not a production-scale Ethereum or multi-day soak result.

Key/header cardinality, full state/ancestry audit time and genuine multi-day churn remain qualification boundaries. No model/consensus rule changes, data-format migration, new environment switch or validation bypass is introduced.

Reproduce with the committed source and unique owned directories:

```sh
cd pkg
go build -o /tmp/filedb-profile ./cmd/filedb-profile
/tmp/filedb-profile --mode=prepare --dir=/fresh/owned-profile --keys=10000 --value-bytes=65536
/tmp/filedb-profile --mode=open --dir=/fresh/owned-profile
```

Preparation refuses an existing directory; open requires its fixture marker. Historical original-chain datadirs are never fixture inputs.
