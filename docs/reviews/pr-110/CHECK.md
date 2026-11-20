# Durable FileDB recovery

- [x] Retain the actual cold-observer maintenance startup and trace WAL replay rewriting already materialized committed data.
- [x] Coalesce only complete committed transaction intents, retain committed prefixes on every torn suffix and apply only final per-key operations under the exclusive database lock.
- [x] Preserve file/directory durability, missing/corrupt-file repair and write fencing; skip exact materialized values without changing consensus validation.
- [x] Execute truncation/delete/overwrite/corruption/fault/restart regressions, behavior mutations, affected race/integration/vet and native/tool builds.
- [ ] Publish the separate review and qualify actual process recovery without extending its operating deadline.
