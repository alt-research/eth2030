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
changed EIP sources require a dated impact/admission history entry. It never
adopts the new files. New scope requires a new explicit baseline version.

`codec_goldens.py --write` and `cd pkg && go run
../tools/hegota/bls_vector.go` are explicit, review-only regeneration commands.
The first uses an independent minimal Python RLP/Keccak oracle. The second signs
a deterministic public test message with the separately pinned blst library;
normal builds test gnark verification against the frozen bytes. The fixture's
key is exclusively a public test vector. Neither command is invoked to update
an oracle during acceptance.

`semantic_goldens.py` recomputes two hand-worked gas/fee oracles and one recent-root
domain oracle; nine semantic trace seeds remain for the future native EVM
conformance suite. Passing the oracle arithmetic does not count as executing
those traces. CI executes the source/admission checks and strict native/upstream
baseline before the full Go build/test checks.
