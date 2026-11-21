# Final post-merge source qualification archive

`postmerge.tar.zst` contains 245 explicitly selected original public source/test/protocol/runtime captures (144,532,457 uncompressed bytes). Its MANIFEST.json records the exact size and SHA-256 of every member. ARCHIVE.json records the complete verification and archive digest. All original failures and earlier source phases keep their own identity. The earlier nine final-campaign bundles remain unchanged.

Private datadir backups, partial directory copies, executable binaries and credentials are excluded. Actual runtime credential values were checked absent from every selected member without recording their values. Full multi-hour Engine streams remain in the owned qualification workspace; the archive includes all window rows and contiguous sequence/sealing tails. The unchanged parser reproduces every original count/status/latency/gap field exactly.

Verify from the repository root with Python 3.14:

```sh
python3 pkg/devnet/kurtosis/scripts/glamsterdam_artifacts.py verify docs/reviews/glamsterdam-postmerge-20261121/artifacts/postmerge.tar.zst
```

Extract with `tarfile.open(..., "r:*").extractall(fresh_directory, filter="data")`. Use the pinned unchanged reference environment described in `pkg/devnet/kurtosis/qualification/reference/README.md`. Source scripts, SSZ decoder, genesis and original trusted bootstrap are included. Run finality consumers with `--replay` and a new output path; do not discover Docker or access live endpoints.

```sh
python /fresh/source/scripts/reference_operating_window.py --capture /fresh/steady-reviewed-02 --trusted-bootstrap /fresh/reference/trusted-bootstrap.jsonl --replay --output /fresh/replayed-steady.json
python /fresh/source/scripts/reference_operating_window.py --capture /fresh/pressure-reviewed-02 --trusted-bootstrap /fresh/reference/trusted-bootstrap.jsonl --replay --output /fresh/replayed-pressure.json
python /fresh/final-blob/reference_external_columns.py --anchor /fresh/final-blob/anchor.json --bundle /fresh/final-blob/bundle.json --genesis /fresh/reference/genesis-beacon.ssz --output /fresh/replayed-blob.json
# Build the independent Geth SDK consumer from the published qualified source.
go build -C pkg -o /tmp/glamsterdam-ledger ./cmd/glamsterdam-ledger
/tmp/glamsterdam-ledger proof-stream /fresh/pressure-reviewed-02/observations.jsonl
/tmp/glamsterdam-ledger proof-stream /fresh/final-readback/historical-proofs.jsonl
```

The external-builder blob verifier obtains its public key from complete parent SSZ checked against the signed parent header. Separate captured finality binds both that parent and the blob block. It changes no BLS/KZG/reference acceptance rule. The original beacon-root state API404 remains captured; the same successful transaction is verified through its exact committed state-root lookup. No transaction reroll or source/configuration change hides that harness failure.

Fresh archive reproduction verifies all245 members, four captured finality spans, the544-proof external-builder blob, and six SDK proof streams (282 account proofs). All11 checks pass; Python sockets are blocked. PUBLIC_REPRODUCTION.json records exact inputs/results. An initial runpy import setup failure is retained separately in PUBLIC_REPLAY_PREPARATION.json; it ran no cryptographic consumer, and the retry uses a new extraction of the same archive.
