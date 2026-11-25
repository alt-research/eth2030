# Long-soak audit captures

The173-member archive retains an independently verified historical prefix, the later original operating failure, public raw RPC/CL traces, source/license pins and real negative controls. Complete per-file byte sizes and SHA256 identities are in MANIFEST.json. The frozen source/module inputs are selected from auditor2d2bb79d, native2c0df12f and CLd547ba984; future admission selects its separately reviewed runtime tuple. Private databases, P2P/validator keys, JWT contents and native binaries are excluded. The Docker runtime file is explicitly a selected-field projection, not the full private inspection.

The old run remains failed after781 samples /23545.536 observed seconds. Its CL API times out at10s and exits137 after unsuccessful30s graceful stop, withoutOOM. The historical763-sample prefix was independently checked against actual CL/EL/Geth bytes;154 account MPT proofs and all seven BLS windows freshly replay. The added overlapping finality segment closes the entire actual failed-prefix beacon span64059–71906 and includes both earlier finalized anchors. It does not turn the run into a48-hour pass.

The full7872-block execution-prefix scan agrees on native/Geth RPC commitments and parent links; it observes5 transactions and979200 gas. This scan does not rehash every RLP header independently and does not establish a transaction/storage-load capacity result. The original stopped-observer scan setup failure remains separate.

## Fresh public replay

Use Python3.14 for zstd extraction. Verify the compressed/per-file manifest using the existing glamsterdam_artifacts.py tool, then extract into a fresh directory. Build only the captured independent Geth SDK wrapper:

```sh
python3 pkg/devnet/kurtosis/scripts/glamsterdam_artifacts.py verify docs/reviews/soak-evidence-closure/artifacts/soak-evidence.tar.zst
mkdir /tmp/soak-capture
python3 -m tarfile -e docs/reviews/soak-evidence-closure/artifacts/soak-evidence.tar.zst /tmp/soak-capture
cd /tmp/soak-capture/source/pkg
GOTOOLCHAIN=go1.26.11 go build -trimpath -buildvcs=false -o /tmp/soak-ledger ./cmd/glamsterdam-ledger
```

Use the unchanged independent consensus environment pinned by the captured requirements/source archive. Its generated Gloas/Fulu modules must match the documented byte fingerprints and BLS must be active. The included immutable source archive is2ea9d96a420927eb1a35d7c9399c8b3691c69f6a, SHA2561fc9482c214443d708b4ae4b225dce47246331e530361de56e0277400517bc94. Isolated regeneration matches the actual consumer; do not regenerate into the live reference environment.

```sh
python3 /tmp/soak-capture/replay_frozen_prefix.py --capture /tmp/soak-capture --reference-python "$REFERENCE_PYTHON" --ledger /tmp/soak-ledger --output /tmp/soak-replay
```

The fresh replay checks actual sample/wire grouping, all154 account proofs, seven unchanged-reference BLS segments, the final failed-span extension and a real incorrect-balance proof. Each BLS replay retains signature/branch/genesis-domain negative controls. The replay is historical cryptographic verification; current runtime freshness, real48-hour operation and clean daemon termination require the live admission gate and cannot be inferred from this archive. Observed participation422/459/509 is evaluated under the unchanged normative sync-committee rules.
