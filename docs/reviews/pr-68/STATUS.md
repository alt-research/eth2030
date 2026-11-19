# PR 68 tracking cleanup status

Original review head: `323bf8fda9bf6caf356a2ecc42427f7a850a98e8`. Merge base for preserved records: `38ea9259901c7873f483752ea390962e26563648`. Target master: `98716284b049933888bcd68d64c1957004d759b2`. Shared tracking policy: `42a9ceb86dea2fe4a74f01fd5e2360d6f21af0e7`. The sections below preserve this branch snapshot, including any inherited feature records; their historical validation is not a new result for this cleanup. The exact original diff, including removed text, is in [legacy-tracking.patch](legacy-tracking.patch).

Cleanup validation is pending. Publication must be a fast-forward update of the existing review branch. Draft status and dependency order are unchanged.

## Preserved feature tracking

## RPC access-list discovery (2026-11-18)

Branch `feat/glamsterdam-access-list` replaces the eth_createAccessList empty placeholder with production backend discovery. Independent state copies execute native transactions until the access list converges; returned gas is measured with the returned list, without selecting an empty list merely because it is cheaper. Reverted child touches are retained, lists/keys are deterministic, and malformed addresses/keys/quantities return invalid-params errors. Unsupported backends no longer report fictitious discovery.

The actual HTTP/node-backend regression first failed on the placeholder and now passes. Additional tests cover reverted inner calls, contract-creation accesses, historical versus current state, no canonical nonce/storage mutation and malformed input. RPC/node/eiptest suites, targeted race, full build and vet passed. This delivers GL-044 access discovery for current CallArgs; final gas-model/API semantic and broader authorization/blob argument qualification remain GL-020/025/045 work. [Evidence](../../verification/glamsterdam-access-list-20261118.md).

