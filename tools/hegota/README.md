# Hegotá A verification tools

From the repository root:

```sh
python3 tools/hegota/check.py
python3 tools/hegota/codec_goldens.py
python3 tools/hegota/semantic_goldens.py
python3 -m unittest discover -s tools/hegota -p 'test_*.py'
cd pkg
env -u GOROOT go run ./cmd/eth2030-baseline
```

All Python verification uses only the standard library; it performs no network
calls and does not change locks/oracles. Source check requires the two pinned
reference submodules. `check.py --compare-lock NEW_LOCK --compare-admission
NEW_ADMISSION` prints old/new source and disposition changes plus affected tasks;
changed EIP hashes, revisions or activation decisions require an appended dated impact/admission history entry. New source locks must match admission records; old history cannot be rewritten. Repository SHA changes require dated `source_updates` records naming repo/ref, previous/new SHA, affected tasks and rationale. Immutable source URLs cannot silently change their digest. Additional/removal of non-scope source files requires `file_updates` old/new digests, date, tasks and rationale unless covered by a reviewed repository change. It never
adopts the new files. New scope requires a new explicit baseline version.

`codec_goldens.py --write` and `cd pkg && go run
../tools/hegota/bls_vector.go` are explicit, review-only regeneration commands.
The first uses an independent minimal Python RLP/Keccak oracle. The second signs
a deterministic public test message with the separately pinned blst library;
normal builds test gnark verification against the frozen bytes. The fixture's
key is exclusively a public test vector. Neither command is invoked to update
an oracle during acceptance.

`semantic_goldens.py` checks two hand-worked gas/fee oracles, a root-domain oracle,
structured fixture coverage and two isolated nonce/window rule oracles. All nine
former trace seeds now have concrete inputs/outputs. Seven base cases are executed
by `core/execution.TestHegotaSemanticGoldens` using the existing native VM/executor;
the two conditional oracles do not activate a composed network profile.

`acceptance.py` validates the criterion matrix, actual passing native test events,
current artifact/source hashes, skip counts, prerequisite assessments and active
reference identity. It permits truthful review/blocked task records, while the
completion gate rejects unresolved criteria. Neither `verified` labels nor
existing artifact filenames replace execution evidence. Negative regressions
cover missing criteria/tests, skipped/incomplete/stale runs, version drift and
incorrect conditional semantics. CI runs the source/acceptance checks and strict
baseline before the normal Go checks; Go tests execute the native fixtures.
