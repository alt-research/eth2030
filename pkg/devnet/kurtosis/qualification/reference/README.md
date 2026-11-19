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
