# Validation and review boundaries

## EL

The public cumulative branch qualification/glamsterdam-spec-closure-20261124 at 264adba468069377fb50b487bd2dec77c089b28d combines the two independent EL feature reviews. Its pkg tree is 0206fabc89395694d02b0bf63d069d8c825b5192. Full default tests pass in 219 test-bearing packages, with 18 no-test packages separately counted and 23369 test pass events (including parent/subtest events, not a count of distinct official vectors). Complete eth, engine, node/backend, txpool and node race runs pass, as do full vet and both native/Geth-wrapper entrypoint builds. EL_VALIDATION.json retains exact source, command log and package outcomes; EL_VALIDATION_INITIAL.json preserves the earlier cumulative code.

New negatives reproduce canonical ordinary-mask rejection by independent Geth, native empty-mask acceptance, unavailable-versus-missing query results, missing-cell/proof pairing, query shape/count, serving cached data during sync and silent legacy-proof conversion. Actual native HTTP FCU targets exercise healthy/sync/recovered query responses. Actual native TCP peers promote a reconstructed authenticated blob, serve SDK-compatible V2/V3 and KZG-checked V4 cells, and reacquire data after receiver restart. These are controlled affected-path tests, not an original-04 reproduction or a new historical catchup campaign.

The entire Go core and SSZ trees equal the earlier qualified source 50514bf3: f054c1411101e67c51a45724f81d44b13c527d22 and af16b77c492342cf576e98e3bd63975474a54c65. Earlier official native/RPC/application/long-window results keep their original binaries. Current EL hosted CI also runs both selected complete native fork corpora, wrapper fixtures, application workflows and integration/race/crypto variants. The one initial 151 vet job failed during dependency download with proxy HTTP/2 INTERNAL_ERROR, before a vet diagnostic; its isolated retry passed without a source change. CI_ATTEMPTS.json retains that distinction.

## CL

Source 87169d6ab7f0336f4448d6d54a23fed848426022 passes cargo check -p lighthouse and the complete 59-test Gloas bid suite under FORK_NAME=gloas, real blst/supranational and no fake_crypto. New actual signed regressions fail before each fix. The state/clock cases use the production verifier and parent cache, while pure cache tests independently exercise atomic limits and pruning. No synthetic cache helper is claimed as a live chain run.

An early exact filter ran zero tests and is not acceptance. A counter-only run passed 57 tests; adding authoritative slot advance exposed an invalid old epoch-0/finalized-1 fixture (57/58). The fixture was corrected with caches, an epoch-2 anchor and a matching latest block header; slot processing was retained. A subsequent full 58-test run and the final clock-inclusive 59-test run passed. Exact commands and sources are in CL_VALIDATION.json; behavioral/fixture failures remain in the companion review.

An extra ad hoc Clippy invocation omitted the repository Makefile's enum_variant_names allowlist and failed on an existing execution-layer enum. Repeating with the actual project lint policy passed; the baseline enum was not changed. Final hosted test-suite/local-testnet and merge status remain attached to their exact PR head; this initial document snapshot does not infer completion while they are still running.

## Integration

INITIAL_MERGE_AUDIT.json checks current EL PRs 149/150/151 as independent master targets: all 12 subset extensions are clean, cover all six merge orders, converge to the same full tree and produce the tested pkg tree. Final review submission adds its own current-head audit. Simulated trees are not actual master merges.

48h/larger-scale and Lodestar approval are user-deferred and do not lower current development progress. Original 04 exact waiter/device attribution is unknown because contemporaneous capture is absent; unrelated corrections or no recurrence cannot reconstruct it. Public rollout requires a chosen network/client tuple and schedule using existing configuration support. No public preset changes or research proof/Frame/PQ/native-rollup adoption occur here.

The final timer integration extends the signed clock test to the actual per-slot cleanup consumer. The original cleanup deleted previous-slot preferences and reset counted evidence; the new shared cutoff retains it for the allowed/in-flight window. A previously consumed credit survives the timer, and an additional signed excess bid remains ignored. Source 3d6ea012 receives separate final-head suite/CI receipts; the earlier 59-test 87169d6 run remains source-specific.
