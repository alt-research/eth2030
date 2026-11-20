# Original process qualification inputs

Each archive contains MANIFEST.json with the normalized path, exact size and SHA-256 of every original member. The original captures remain retained in the owned qualification workspace. Compression does not transform captured protocol/state bytes. Python 3.14 provides the standard zstd decoder used by the independent reference environment and CI.

Current bundles:

| Archive | Original files | Original bytes | Compressed bytes | Scope |
| --- | --- | --- | --- | --- |
| protocol.tar.zst | 734 | 323125226 | 2312307 | Original signed lifecycle states/blocks/envelopes, actual reorg, payment proofs, partial/custody/blob/factory consumers and original failed adapters. |
| durability.tar.gz | 69 | 24286250 | 2029206 | Original process OS faults/WAL, cold sync, indexing, final-source startup and retained-history observations. |
| operating.tar.gz | 37 | 41474947 | 3627478 | Original accepted pre-network-fix operating windows and the unqualified first pressure cadence attempt. |
| provenance.tar.zst | 33 | 5122689 | 363665 | Public genesis/trusted bootstrap, exact source/fixture pins, source/license/lock inventories and byte-identical native rebuild. |

Final network-recovery, final operating/cold-source and registry continuation inputs are being added as separate bundles. Those pending cases are not represented as completed by the three archives above.

Verify an archive from the repository root:

```sh
python3 pkg/devnet/kurtosis/scripts/glamsterdam_artifacts.py verify docs/reviews/glamsterdam-final-campaign/artifacts/protocol.tar.zst
```

Unpack verified archives into a fresh directory with Python 3.14 tarfile.open(..., "r:*").extractall(destination, filter="data"). Keep the original files unchanged. Every replay requires a new output path. The selected unchanged reference setup and its actual BLS/KZG dependencies are documented in pkg/devnet/kurtosis/qualification/reference/README.md; use an owned source copy, generate all forks with the official generator and retain consensus-specs source 2ea9d96a420927eb1a35d7c9399c8b3691c69f6a.

Representative offline consumers:

```sh
# Build from pkg with the pinned independent Geth v1.17.10 SDK.
go build -o /tmp/glamsterdam-ledger ./cmd/glamsterdam-ledger
/tmp/glamsterdam-ledger /fresh/artifacts/actual-lifecycle-payments/payment-20695.json
/tmp/glamsterdam-ledger proof-stream /fresh/artifacts/factory-process-01/proof-stream.jsonl

# Use the unchanged-reference Python environment, from the repository root.
python pkg/devnet/kurtosis/scripts/reference_lifecycle_states.py --capture /fresh/artifacts/builder-lifecycle-03 --output /fresh/replay-lifecycle.json --invalid-slot 20709 --invalid-engine /fresh/artifacts/builder-lifecycle-03/invalid-root-engine-exchanges-02.jsonl
python pkg/devnet/kurtosis/scripts/reference_reorg.py --capture /fresh/artifacts/controlled-reorg-05 --genesis /fresh/artifacts/genesis-beacon.ssz --output /fresh/replay-reorg.json
python pkg/devnet/kurtosis/scripts/reference_partial_columns.py --anchor /fresh/artifacts/peer-custody-process-01/cgc-16/anchor.json --bundle /fresh/artifacts/peer-custody-process-01/cgc-16/bundle.json --genesis /fresh/artifacts/genesis-beacon.ssz --output /fresh/replay-partials.json
python pkg/devnet/kurtosis/scripts/reference_operating_window.py --capture /fresh/artifacts/steady-final-01 --trusted-bootstrap /fresh/artifacts/trusted-bootstrap.jsonl --replay --output /fresh/replay-finality.json
```

The genesis and original trusted bootstrap belong to the provenance bundle. The captured finality replay performs no Docker discovery or server access. Authentication-only spans report that no operating budget was evaluated; they do not convert controlled network-fault intervals into steady-window passes.

Operating Engine selections contain exact requests started within their declared windows, plus sealing traffic and lower sequence tail records. Replaying those selections reproduces every original count/status/latency/gap metric exactly. Original full multi-hour Engine captures remain retained separately. Failed HTTP/RPC requests and failed case inputs are preserved. The original 129/400 missing-slot window remains a failed observation in the earlier PR-101 artifacts.

The bundle excludes node databases, process credentials and executable images. Actual OS-fault WAL bytes are the isolated chain database records inventoried in wal-artifact-inventory.json. Source pins, original license/lock inventories, reproducible build commands, actual executable/module hashes and deployment limits are recorded separately.
