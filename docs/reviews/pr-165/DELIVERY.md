# Glamsterdam conformance delivery

The confirmed G01–G05 engineering repairs are implemented and published. The
qualified contract is the twenty core EIPs scheduled by EIP-7773 at EIPs
`d27448e08f60513856bbe228ea7ce04f156209dc`, with detailed consensus rules at
`d2021d2719235f0df4726a1d330502cb15aa8ea3`. Networking references and the
retained FCR extension have separate qualification evidence. This describes
project implementation and finite validation, not global Ethereum activation.

| Work | Implemented result | Qualification and delivery |
| --- | --- | --- |
| G01 | Native signed-block import, full header/state/BAL comparison, strict framing and six native consensus corrections. | EL #160/#162/#163. All 43,790 official Osaka/Amsterdam cases pass: 52,620 block checks, 45,221 accepted blocks and 7,399 expected rejections. Six executed behavioral mutations are caught. Sixteen independently verified multi-fault blocks reject at an allowed alternate check; their declared signature branch is not falsely reported as executed. |
| G02 | Current native CL admission, canonical light-client serving, real producer notifications, progressive cell bitmaps, exact FCR committee/vote accounting and complete strict reference consumption. | Lighthouse #18–23. All 12,195 selected full-node/compatibility cases pass locally and in hosted CI. No case receipt is missing, duplicated, skipped or waived. Four retained light-client client/store-role cases remain explicitly unrun outside full-node serving. |
| G03 | Independent encrypted BAL recovery and actual Rust/JS full-column fallback, reverse delivery, reconnect, malformed-message scoring and custody metadata updates. | EL #164 and Lighthouse #21. Two stock Geth 1.17.10 peers exercise native snap/2 recovery over encrypted RLPx. The compatible public Lodestar fork 4e224659 exercises actual transport/codec components; actual delivered bytes reach real native BLS/KZG consumers. |
| G04 | Retire misleading exploratory gas-helper authority and point the tracker at canonical Amsterdam execution code. | EL #161. Runtime token streams are unchanged. Current merged LIFO remains authoritative; EIPs #12581/#12582 are closed without merge. |
| G05 | Preserve the current detailed CL builder-prefix precondition, including existing-builder top-ups; keep stake and registration authority in the CL. | Both clients and unchanged Python reference agree: valid-prefix existing-builder top-up credits 1 ETH with no re-registration signature; invalid prefix credits zero and neither changes the recipient. EIPs #12572 remains an external prose clarification, not an adopted consensus change or code blocker. |

## Crypto modes and real consumers

The 12,195-case inventory consists of 2,515 dynamic cases in the real-crypto
profile, six operation cases explicitly declaring ignored BLS, 330 synthetic
signature FCR cases with real public-key aggregation, and 9,344 SSZ cases.
Serialization/mathematical fixture results are not real-signature acceptance.
The 9,344 SSZ goldens independently regenerate in about 73 seconds and match every
retained input byte under the pinned 47-package generator environment.

The clean combined Lighthouse source
`c93c64e5ae71c771133e48f0b40c5326f9e19f45` passes all eight local release stages:
the complete reference replay, 403 beacon-chain / 4 client / 69 state-processing
library tests, the actual encrypted independent peer, its real delivered-byte
BLS/KZG consumer, 24 native network integration tests, affected Clippy and a
production node build. The two external tests omitted from ordinary test runs
are explicitly executed and pass. The resulting release binary reports real
`blst` and has SHA256
`51f4f23cfce154d96f0334706ce2ac636e6d1b730d3031aa32658e9e7a658fb9`.

The full production-clock basic and fallback simulations also pass, preserving
all assertions and cryptography. The fallback uses 6 beacon nodes and 3 validator
clients, forced EL disconnection/recovery and the 95-percent attestation gate:
head 888/900 (98.67 percent), target/source 891/900 (99 percent). Earlier compressed
clock failures retain their actual delayed-message/reorg traces and do not
establish a native EL deadlock.

## Delivery gate

EL #160–165 all target `master`; EL merging remains the user's separate action.
The five functional reviews retain their own source/CI evidence. The complete
six-review pre-publication checkpoint has 720 conflict-free merge orders and one
resulting tree. Final published heads and CI are checked again after the
documentation commit; the final PR metadata records that outcome without a
self-referential commit hash.

Lighthouse #18–23 are actually merged into `eth2030`, ending at
`e021d1bb17c16579432aa402441c82ed375a93ef`. The final review head
`50c23b97270e16d430b618cab4b769fd8ec2843b` synchronizes all prerequisites
without changing any files. Its tree
`59e45eb596e7695fea22605425f1bd124a4aeaa7` equals the already qualified
predecessor and hosted checkout. All required latest source checks complete
successfully before merge; `LIGHTHOUSE_DELIVERY.json` preserves each exact
review SHA, check result and actual merge receipt.

The artifact-upload path failure is repaired and hosted evidence is retained:
job 138908083879 / run 45525625554 succeeds, including upload. Artifact 14982933817
is downloaded and digest-verified, and every one of its 12,195 case records is
reparsed independently. See `HOSTED_REFERENCE_QUALIFICATION.json` and
`LOCAL_CHECKPOINT.json`.

## Residual boundaries

- Original-04 / the historic 5.5-hour native EL import stall: actual original
  waiting chain and physical cause remain **unconfirmed**. New deterministic
  defects, a non-reproduction or controlled recovery do not close it.
- 48-hour/larger-scale acceptance and Lodestar upstream approval remain excluded
  from development progress, **not passed**.
- The independent Lodestar probe qualifies the stated compatible component
  campaign; it is not a full Lodestar CLI or complete preceding-chain replay.
- EIP-8025 and a full Ethereum STF guest remain later work outside official
  Glamsterdam inclusion.

Older dated checklists and earlier failed snapshots are retained. This current
reconciliation and live review/CI receipts determine delivery; unchecked items
in a historical checkpoint do not silently become new implementation gaps or
passed operating evidence.
