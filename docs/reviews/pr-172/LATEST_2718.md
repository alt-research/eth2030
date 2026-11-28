# Current Group E completeness audit after EIP-2718 clarification

Starting review head: `7d06fc8a6b7d50895cb6d148651f73cbcb47fae5`.
Baseline master: `cc31afffaa12919bbc08dc854c03af1e1928dc3c`.
The starting head's 17 checks, all 11 actual proof cases and independent CL
network artifacts were inspected in its English PR. Those remain historical
source-specific results, not pre-claimed results for later commits.

## Latest source and discussion

EIPs master is `c24e33014288b5e712da2629c185c91327c24c69`. Its only selected
change since `fbc43ec3` is merged #12702, commit
`d2be3735b2f894e1e35f54de408159b5c7e14a94`, reviewed PR head
`1d90773fd107e392e4e56140a790f8d8b8669d4a`. The full current EIP and author
review were read. EIP-7805 now explicitly requires EIP-2718 and counts the IL's
8 KiB over those transaction bytes. Typed transactions use `type || payload`;
legacy transactions use their RLP list. A typed block-item RLP wrapper, JSON hex,
SSZ offsets and external pooled witnesses are outside this transaction budget.
The current EIP-2718 reference is byte-identical to the read-only local copy.
The register now locks 23 normative files and 44 mandatory native consumers.

Heze `36888ee87b0faa6ad7804b017a0184e8e6e6a1c6` and Engine Bogotá
`9e85b16c7fb54a838c9a612842c138c2bdf9bc8d` remain unchanged. The merged
pre-execution-only predicate, independent reservations, ignored type 3 and
version-3 cache migration remain applicable.

The updated open #12723 head is
`a65df9312dce24cbf00d6da46778edaa1e5e21a0`. Its complete revised proposal,
diff and new review replies were read, including complete-payload BAL/request
accounting, the size margin, post-system changes and the distinction from
EIP-8279. It is still unmerged. Its proposed size exemption is not activated.
The prior near-8-MiB regression remains authoritative for the merged rules.
Informational EIP-8369 and optional draft-8504 composition gates also remain.

## Newly covered boundaries and source lock correction

The existing native selector already returns canonical EIP-2718 bytes. Its
`EncodeRLP` method name does not add the block's outer typed-transaction wrapper.
This review does not manufacture a production behavior change where none was
reproduced. It adds the missing end-to-end boundary consumer:

- Independently sign legacy and typed envelopes with Geth. Actual native pool
  admission and `engine_getInclusionListV1` must return exactly 8192 bytes.
- Independently encode each as a block RLP item: the typed item is larger while
  the legacy item is identical. Neither typed framing nor JSON expansion can
  shrink the valid IL budget.
- Admit a valid signed 8193-byte envelope to the pool but exclude it from GetIL;
  the IL limit is not a global mempool transaction limit.
- Admit a real authenticated pooled dependency witness and return only its
  canonical transaction. External witness bytes must not appear in the IL.
- Pass selected canonical bytes through actual Heze byte validation and SSZ
  round-trip; its framing remains outside the transaction sum.

The source-lock review found that the selector and canonical/pooled encoding
implementation files were not mandatory E fingerprint inputs. They now are.
An independent tooling mutation removes each path from all mappings and checks
that validation rejects the incomplete source lock. Three fresh compiled native
controls respectively overcount typed block framing, emit a pooled wrapper and
admit one byte over the IL limit. They join the existing 39 controls; no expected
SSZ/BLS or A/B/C/D oracle is regenerated.

## All nine tasks reviewed

| Task | Runtime and normative boundary |
| --- | --- |
| HGT-040 | Complete current source/diff/review precedence, explicit timings and execution/satisfaction split; unsupported activation remains gated. |
| HGT-041 | Progressive SSZ, canonical transaction-byte sum, authenticated slot/dependent root/committee/domain/signature, late/equivocation views, requests, retention and atomic restore/reconfiguration. |
| HGT-042 | Exact Bogotá fields/methods/capabilities, strict DATA, malformed transaction tolerance, nullable non-VALID status and native GetIL byte contract. |
| HGT-043 | Actual candidate execution, pool/IL funding and nonce closure, authenticated wrappers, rollback, independent gas and block RLP validity, cancellation/ownership/retry. |
| HGT-044 | Independent pre-execution checks at transaction-end state, ignored malformed/type-3 entries, shared Frame preconditions, actual independent pools, no candidate EVM or unmerged size excuse. |
| HGT-045 | Actual native build/import and retained exact/compact branch decisions, finalization/restart, version-3 migration and explicit alternate-backend rejection. |
| HGT-046 | Defined informational reference shapes/budgets/access/code/index checks and closed enforcement prerequisites; no invented Profile 2 activation. |
| HGT-047 | Native authenticated adapters and the selected independent CL signed-gossip/finality/omission/restart control; exhaustive mixed-client/ePBS release coverage remains HGT-070/080. |
| HGT-048 | Actual canonical PQ bytes, external witness separation, uncached verification/local SSZ/BLS pipeline and bounded signed network carriage; no public-tail or omitted-witness composition claim. |

The independent expected oracles and read-only references remain unchanged.
Fresh local qualification and exact publication-head hosted results are required
and recorded separately; earlier green checks do not qualify a later head.
