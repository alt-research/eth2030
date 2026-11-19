# Native signed state fixture qualification

The strict native executor passes all 31,573 selected official EELS tests@v21.0.2 cases: Osaka 14,665/14,665 and Amsterdam 16,908/16,908. Every result compares the complete native post-state root and logs to the independent official oracle; expected invalid envelopes verify unchanged prestate. See qualification.json for exact archive, tested clean native revision and report checksums. All 1,223 historical EEST v5.4.0-develop Osaka cases also pass. Five complete original historical signed fixtures are vendored for default smoke tests, with sources and licenses retained.

The adapter consumes exact signed transaction bytes, complete typed envelopes, coherent fork/BPO configuration and official slotNumber. Invalid metadata, selectors, missing roots/logs, unknown exception classifications, malformed files, empty selected corpora, unexpected rejection and panics fail qualification. Known prohibited creation-envelope errors are classified precisely and still compare unchanged roots/logs. Newer official state exceptions preserve phase and intrinsic/floor/execution-cap distinctions instead of accepting any error.

The independent SDK corpus now includes all five standard Amsterdam envelope types and three invalid blob envelopes; 50/50 Amsterdam scenarios match unchanged Geth v1.17.10. Six deliberately direct Osaka over-cap diagnostics remain visible in that synthetic execution comparison; actual raw admission and correctly bound Engine consumer rejection are proven by the separate signed-validation review. No SDK opcode/precompile substitutions or native state repairs are used.

CI downloads the immutable 939,189,561-byte release archive, verifies its SHA-256 before extraction, builds the native executor and requires the exact complete per-fork counts, unique case identities and matching roots. Both full result reports and failure logs are uploaded even if execution fails. The existing Geth-backed EF job remains separately named. Full Go tests, core integration, adapter/CLI/comparator race checks, full vet and build pass locally; an earlier overlapping node test run hit the existing deterministic metrics port and was excluded before an isolated full repeat passed.

Reproduction from the repository's pkg directory, after extracting the pinned archive:

```sh
go run ./cmd/native-state-tests -fork Osaka /path/to/fixtures/state_tests/for_osaka > osaka.json
go run ./cmd/native-state-tests -fork Amsterdam /path/to/fixtures/state_tests/for_amsterdam > amsterdam.json
```

This is signed state-transition qualification, not independent multi-client beacon-network, builder-payment, recovery or resource-SLO qualification. Official transaction-test and blockchain-test formats are not represented as additional executed cases.
