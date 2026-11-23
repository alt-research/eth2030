# Public remaining-code evidence

The frozen archive has95 explicitly selected original files and a complete per-file size/SHA256 manifest. It contains final-source test logs, source/Go module pins and licenses, standard signed CL inputs and captured finality, large-database restart request/proof bytes, original failures and a dated in-progress observer snapshot. It excludes JWT/validator keys, private database/tars, native binaries and installed environments. Earlier archives remain unchanged. ARCHIVE.json is the compressed-byte identity; ../PUBLIC_REPRODUCTION.json records fresh extraction and independent replays.

The full original1,234,207-file database is private and excluded. Public proof/header/CL replay verifies captured cryptographic claims; it does not reproduce all database files, startup timings, resource measurements or a full48-hour operating duration. Snapshot scripts retain their exact host/input dependencies for review. Synthetic profiles can be generated with cmd/filedb-profile under its ownership/marker restrictions; preparation and measured open remain separate.

## Verify and independently replay

From a checkout of the reviewed source, use Python3.14 for .zst support and the unchanged consensus-reference environment pinned in pkg/devnet/kurtosis/qualification/reference/requirements.txt. The Python spec source must be2ea9d96a420927eb1a35d7c9399c8b3691c69f6a with BLS active; do not replace reference validation with native helpers.

```sh
python3 pkg/devnet/kurtosis/scripts/glamsterdam_artifacts.py verify docs/reviews/remaining-code-closure/artifacts/remaining-code.tar.zst
mkdir /tmp/remaining-code-capture
python3 -m tarfile -e docs/reviews/remaining-code-closure/artifacts/remaining-code.tar.zst /tmp/remaining-code-capture
cd /tmp/remaining-code-capture/source/pkg
GOTOOLCHAIN=go1.26.11 go build -trimpath -o /tmp/remaining-code-ledger ./cmd/glamsterdam-ledger
/tmp/remaining-code-ledger proof-stream /tmp/remaining-code-capture/snapshot-runtime-02/proofs.jsonl
/tmp/remaining-code-ledger proof-stream /tmp/remaining-code-capture/soak-progress-at-handoff/proofs-0.jsonl
```

Set REFERENCE_PYTHON to the pinned reference environment interpreter. Run these from /tmp/remaining-code-capture with fresh output filenames:

```sh
"$REFERENCE_PYTHON" source/docs/reviews/builder-authority/verify_standard_builder.py --capture standard-builder-capture --output /tmp/remaining-builder.json
"$REFERENCE_PYTHON" source/pkg/devnet/kurtosis/scripts/reference_operating_window.py --capture standard-builder-capture/final-blob/finality --trusted-bootstrap standard-builder-capture/reference/trusted-bootstrap.jsonl --replay --output /tmp/remaining-builder-finality.json
"$REFERENCE_PYTHON" source/pkg/devnet/kurtosis/scripts/reference_operating_window.py --capture runtime-prefix-finality --trusted-bootstrap standard-builder-capture/reference/trusted-bootstrap.jsonl --replay --output /tmp/remaining-runtime-prefix.json
```

The standard bid has13 rejected mutations and exact signed SSZ roundtrip. Builder parent/blob finality has512 participants and seven links. The new runtime prefix64059–64079 has512 participants,38 links and no missing slots; it authenticates21 observed slots, not48hours. Both finality replays reject signature/branch/genesis-domain mutations. The SDK verifies four private-snapshot proofs and two actual observer proofs from public bytes.

## Long-running controller handoff

The source-reviewed ../observe_soak.py uses an explicitly supplied owned-container record and independent SDK ledger; its labels/identities/limits prevent it from stopping the original fleet. The companion ../observe_soak_finality.py collects contiguous signed windows about hourly, preserving the original4096-link verifier bound, and independently replays each before extending covered_through. Both the operating result and complete finality coverage must pass. A process still running, old running snapshot or successful prefix does not complete either gate.

Current owned run: /projects/eth2030/qualification/remaining-code-20261122/live-soak-03. Read run/result.json and finality/result.json together. The controller stops its own EL/CL pair on failure/completion. It does not auto-retry failures or alter original peers, scoring, validation, resources or fork rules. The initial peer-depleted run remains failed in live-soak-02; only new own-CL boot seeds changed for03. Do not overwrite or silently relabel previous attempts.
