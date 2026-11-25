# Case-level baseline, gas and nonce acceptance audit

Starting revision: `5fb8c99843d296a2b41edd48ebfdb0283037ed01`; review date 2026-11-26. All seven original A deliverables were rechecked. The runtime head-number, canonical activation, source/profile, safety, prerequisite and 72-proposal admission work remains intact. This review closes the following additional acceptance gaps.

| Task | Finding and closure |
| --- | --- |
| HGT-001 / 004 | Fixed source/profile/reference identities and native/helper/wrapper routes remain qualified. Normative EIP/CL sources and repository pins are unchanged. |
| HGT-005 | Nonzero block-height and independent canonical timestamp/reorg tests remain mandatory. No production fork rule was weakened. |
| HGT-006 | A persisted 12/12 summary could previously omit all case results. The checker now loads the pinned manifest, verifies fixture hashes and exact case identities, rejects duplicate/missing/unknown cases and false/error results, checks native/reference roots and logs against each official post oracle, and requires positive matching gas. Unknown/canceled run counters and boolean completion counts also reject. |
| HGT-007 | Gas now has its own acceptance criterion rather than only an indirect arithmetic command. Five frozen gas/fee goldens cover standard and floor budgets, capped refund, refund-floor saturation and blob fees with clipped upfront priority budget. Existing native budget/payment/settlement methods consume all five. The isolated keyed-nonce oracle now preserves the current legacy account nonce when keys are used; new cases cover an earlier legacy nonce change and the valid 16-key boundary. |
| HGT-008 / 009 | Fail-closed safety boundaries and all 72 proposal/discussion/admission histories remain required and unchanged. Neither skipped/failed results nor a relabeled executor satisfy acceptance. |

The explicit gas criterion increases the matrix from 25 to **26 criteria**, covering the same seven A tasks. There are seven native VM trace cases, five native fee-method cases, two core conditional rule oracles and 25 conditional boundary vectors. The new fee consumers use frozen execution-usage oracle inputs; they do not claim extra VM programs, blob-proof validation or full fork conformance. Conditional extension activation and D01–D09 remain unchanged.

Current source-bound results are recorded in `evidence/hegota-a/verification.json`; exact-head hosted outcomes are recorded on the English PR #168. Earlier qualification remains pinned to its original revision and is not counted as a fresh run.

Fresh local validation: full Go suite 23,537 named pass events, zero failures, 124 separately inventoried skips across all 240 packages; focused race 3,510 pass events, zero failures, six skips across 31 packages. All 65 Python regressions pass in standard and optimized modes. Build/vet and both host binaries pass. The live differential runner executes all 12 cases with zero failures/skips, and the persisted case report passes independent manifest/oracle validation. All seven A tasks and 26 criteria are bound to 62 required executed Go records. Native negative overlays reject corrupted refund, blob-burn and nonce expectations.
