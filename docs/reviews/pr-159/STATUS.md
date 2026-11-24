# EIP-7495 status

## Implemented and independently qualified

PR159 targets master b14dbb15; functional implementation source is 36b3e62d9f2b3f8f71db25980930ce9d37a5d6ba. Current ProgressiveContainer(active_fields) semantics use immutable validated schemas, canonical typed Container serialization, progressive roots and stable field proofs. Existing EL execution SSZ consumers use checked slot/mask construction and propagate errors. StableContainer/Profile remain explicitly deprecated compatibility root utilities, not current EIP type declarations. No CL source change is needed.

The retained audit reproduced 477 incorrect mixed-root gindices in 487 progressive-list proof cases. Correct root-prefix insertion repairs all cases. Four old named conformance tests also accepted ordinary binary Merkleization in an isolated mutant; independently expected roots and branches replace those weak assertions. PRE_FIX_AUDIT.json preserves the failures, not successful acceptance.

Unmodified independent references qualify 29 container cases, 1016 field branches, 487 list branches and four illegal schemas. An actual 384-validator Gloas state agrees byte-for-byte by its recorded wire fingerprint, state root, all 46 field branches and the original three CL branches. A second unchanged consensus reference confirms the normative 256-position layout without remerkleable's additional field-name restriction. The unchanged Lighthouse release consumer passes. All seven executed mutations fail actual assertions. See [validation and reproduction](VALIDATION.md), RESULTS.json, MUTATION_CHECKS.json and CL_CONSUMER.json.

All 220 Go packages pass. Affected full race suites, vet and both node builds pass. Hosted functional-source CI run 45400878152 passed all nine jobs. Site build, 149-entry/404-path verification and export guards pass; desktop/mobile browser checks cover fourteen views without page errors or overflow. Bottom disclosures remain closed by default. BROWSER_CHECKS.json records the checks. The progress update reports 20/20 official core implementation labels only after this qualification; EIP-8025 and the canonical Ethereum STF guest remain later proposals.

## Delivery boundary

Final documentation/site source and its exact-head CI/mergeability are reviewed on [PR159](https://github.com/alt-research/eth2030/pull/159); the hosted run above is explicitly the functional-source run, not a claim about a future commit. This record does not merge EL master or deploy the public site. Master review approval and site deployment follow the ordinary merge workflow.

48h/larger-scale acceptance and upstream Lodestar approval remain excluded from development progress. No public activation parameter is selected. Historical original-04 / 5.5-hour waiting-chain attribution remains UNKNOWN; this pure SSZ repair does not establish its root cause or resolution.
