# EIP-7495 validation and scope

## Contract

Current EIP-7495 at 3b3c832577ec4205d463d990d52006e299962449 and EIP-7916 at 88da569e65d32ad6efad017f1e40d107f1de8394 are retained alongside this record. The implementation owns an immutable active-field schema, rejects all four normative illegal layouts, composes ordinary typed Container encodings/defaults/JSON, mixes the packed 256-bit metadata word, and generates stable field proofs. Field-level codecs remain the existing typed SSZ codecs. No second reflective SSZ type system is introduced.

The low-level root primitive is explicitly unchecked; concrete EL root consumers now use checked indexed slots and propagate layout errors. The old mutable StableContainer/Profile API remains a deprecated compatibility root utility, not a normative type declaration. Its historical optional/capacity behavior is not offered by the new schema API. No production caller of that compatibility constructor was found.

## Executed failures and mutation controls

The pre-fix audit retained 477 invalid gindices in 487 list proofs, with reference index 40 versus returned 48 for the second element. Ordinary binary Merkleization also passed all four old named conformance tests. PRE_FIX_AUDIT.json preserves these failures. The new tests replace those ineffective assertions with expected independent roots, complete branches and schema errors.

Seven isolated source-copy mutants must fail actual assertions: ordinary binary Merkleization, omitted type mix-in, the old appended gindex edge, accepting a trailing inactive field, borrowing caller-owned field sizes, accepting an offset gap and accepting a nonzero inactive slot. Source compilation failure is not counted as a successful control; production sources are not modified by the mutation runner. See MUTATION_CHECKS.json.

## Independent references

`pkg/ssz/testdata/eip7495/generate.py` imports unchanged ethereum/remerkleable at 2f0baeef0082d4278acaef7d822deb7009d7db7e. Its deterministic gzip corpus contains 29 dense/sparse/fixed/variable/nested/default container cases, 1016 complete field proofs, 487 list proofs and four independently rejected schemas. Roots and branches are not derived from the Go implementation.

That reference imposes an extra field-name requirement at exactly 256 positions. The EIP does not. Its vector uses the accepted name without patching the reference; names do not affect SSZ bytes/roots/indices. An independent unchanged consensus-specs d2021d27 reference also builds the normative 256-position type with ordinary field names and confirms identical bytes and roots. Go accepts 256-position schemas without any field-name requirement. This limitation is documented rather than importing an extra protocol rule or waiving the boundary.

Reproduction:

```sh
PYTHONPATH=/path/to/unmodified/remerkleable python3 pkg/ssz/testdata/eip7495/generate.py > /tmp/eip7495-vectors.json.gz
PYTHONPATH=/path/to/unmodified/remerkleable /path/to/generated-consensus-reference-venv/bin/python pkg/ssz/testdata/eip7495/capture_reference.py /path/to/lighthouse/consensus/types/testdata > /tmp/eip7495-captured.json.gz
```

The captured corpus decodes the original actual 384-validator Gloas state, checks its authenticated recorded root, and exports actual typed field bytes/roots and reference-generated branches. The new Go API reproduces its exact wire SHA256, state root, all 46 field proofs and the original current/next/finalized complete branches. No synthetic zero-state replacement is used. The unchanged Lighthouse captured-state release test independently passes; CL_CONSUMER.json records the exact source and command. No CL source change or new CL PR is necessary.

## Local and hosted gates

RESULTS.json records all 220 normal Go packages passing, affected full race suites, vet and both native/Geth-wrapper builds. Existing unrelated external real-proof tests skip when their packaged backend is absent; those skips are listed and not counted as EIP-7495 cryptographic acceptance. Every EIP-7495 reference and captured consumer case runs without an optional backend.

Existing EIP-6404/EIP-7807 bytes and roots remain unchanged under the checked root path. Their conformance fixtures do not imply activation of those separate draft features. EIP-7495 qualification does not depend on EIP-8025, a new public activation schedule, 48h/scale admission or upstream Lodestar approval. The original historical 5.5-hour attribution is not changed by this pure SSZ correctness work.

The implementation evidence is pinned to functional source 36b3e62d; subsequent site/documentation updates report the resulting 20/20 official-scope project labels. Final exact-head hosted checks and master mergeability are attached to PR159 at delivery. EL master is not merged by this task.
