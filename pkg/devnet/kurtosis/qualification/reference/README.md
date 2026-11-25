# Independent consumers of actual Glamsterdam process artifacts

Use an isolated Python 3.14 environment. Install the unchanged consensus-specs source at `2ea9d96a420927eb1a35d7c9399c8b3691c69f6a` in editable mode, its declared dependencies, and `requirements.txt`. In that source run the official `python -m pysetup.generate_specs --all-forks`. No generated specification or cryptographic verifier is patched.

Unpack the public gzip inputs from `docs/reviews/pr-101/results` into a fresh artifact directory, verify their uncompressed SHA-256 values against `manifest.json`, and retain the original inputs. The blob verifier requires the successful independently authenticated light-client report and its original header captures. It runs with actual BLS enabled and uses the selected trusted KZG setup.

```sh
python reference-actual-blob-consumer.py --artifact-dir /path/to/fresh/artifacts
python reference-progressive-state-proofs.py --artifact-dir /path/to/fresh/artifacts
```

Output files are exclusive or require a fresh output directory; never overwrite a failed run to make it pass. Original harness adapter failures are retained outside the passing result. Progressive-state input manifests identify the three identical original SSZ states; copy the unpacked `actual-gloas-384-state.ssz.gz` bytes under those three manifest filenames when reproducing the frozen state test.

The blob consumer verifies the entire signed beacon block and execution envelope, exact slot/parent/execution identities, all 128 KZG cell proofs per node, and the versioned hash from the strict RLP-decoded transaction whose Keccak hash matches the actual application trace. Cell/index/proof and envelope message/domain mutations must fail. The validator public key is authenticated against the fixed genesis validator root. The selected blob block is linked to the independently verified supermajority-finalized header through 88 exact parent hashes.

The progressive-state consumer independently hashes the actual common 384-validator SSZ state and builds current/next committee and finalized-root branches at generalized indices 2945/2946/735. Mutated siblings must fail. The SSZ REST helper only normalizes decimal JSON integers and typed containers; consensus and cryptographic checks remain in the unchanged reference.

One real blob and full-column readback do not establish partial row ordering, peer loss/reconnect or complete operating capacity. These remain explicit campaign gates. The network is the frozen private chain 1337; these scripts do not generate or submit public-network traffic.

## Finalized builder-payment replay

The later public bundle includes the exact signed blocks and 384-validator SSZ states for slots 9600/9601, complete three-endpoint execution headers/account proofs, the original trusted light-client capture and the actual period-one extension. After verifying and unpacking every gzip against `manifest.json`, run these consumers in a fresh artifact directory:

```sh
# Run from pkg to use the pinned independent Geth SDK dependency.
go build -o /tmp/glamsterdam-ledger ./cmd/glamsterdam-ledger
/tmp/glamsterdam-ledger /path/to/artifacts/attempt4-builder-ledger-fresh-20261119T234834Z.json > /path/to/artifacts/replay-independent-el-ledger.json

# Use the same unmodified consensus-specs environment described above.
python reference-period1-finality.py --artifact-dir /path/to/artifacts
python reference-builder-ledger.py --artifact-dir /path/to/artifacts
```

Both Python replays read only frozen artifacts and fail if their output already exists. The period-one replay starts at trusted slot 96, uses the original advertised committee update, verifies the real 512-participant period-one signature and follows 462 independently rehashed parent headers to payment block 9601. The next committee is unknown after this rotation, which is an explicit store state rather than a failed signature. Frozen replay validation uses signature slot plus one; no current server or patched reference is required.

The payment replay executes the complete actual signed 9600-to-9601 state transition with BLS enabled and requires the exact original state root. It then binds that block to authenticated finality and the fresh independent execution proof report. Three independent endpoints credit precisely 101,000,000,000,000,000 wei against the 101,000,000-Gwei builder debit. Duplicate/missing pending payments, duplicate debit and a changed signed bid fail the unchanged state transition. The Go consumer independently rehashes complete headers and withdrawal tries, verifies account MPT proofs and rejects altered headers, balance claims, proof nodes, withdrawal contents and missing endpoints.

This establishes the matched actual payment transfer and its canonical provenance. It does not label validator epoch issuance or every lifetime payment as separately audited. Original old-state 404, failed restart and failed fresh-snap controls remain distinct public reports. The separate two-row artifacts and Lighthouse captured-state/real-TCP consumers provide additional R09 evidence; full-process cell acquisition and operating capacity remain campaign outcomes to record.
