# PR 67 tracking cleanup status

Original review head: `8bac2058c11f55406b3ba34ee1cda5806ea645b3`. Merge base for preserved records: `38ea9259901c7873f483752ea390962e26563648`. Target master: `98716284b049933888bcd68d64c1957004d759b2`. Shared tracking policy: `42a9ceb86dea2fe4a74f01fd5e2360d6f21af0e7`. The sections below preserve this branch snapshot, including any inherited feature records; their historical validation is not a new result for this cleanup. The exact original diff, including removed text, is in [legacy-tracking.patch](legacy-tracking.patch).

Cleanup validation is pending. Publication must be a fast-forward update of the existing review branch. Draft status and dependency order are unchanged.

## Preserved feature tracking

## Glamsterdam canonical deterministic factory (2026-11-18)

Branch `feat/glamsterdam-factory` uses the EIP-7997 factory at 0x4e59b44847b379578588920cA78FbF26c0B4956C with its exact 69-byte runtime and nonce 1 for explicit genesis/test state. Removed block-processor and both builder paths that injected a different contract at a fork. Private chains must include the canonical alloc or deploy it with an ordinary transaction before activation; no fork existence check or new switch is added.

Canonical address/runtime/nonce and actual CREATE2/value/return-byte/collision tests plus a no-fork-injection regression first failed for the intended reasons and now pass. Preserved legacy private BLS 0x12 without changing the canonical 0x0b..0x11 map. Core/Engine/node/eiptest suites and full build passed. This implements the canonical factory path; live network/genesis tooling qualification remains in the broader integration gates.

