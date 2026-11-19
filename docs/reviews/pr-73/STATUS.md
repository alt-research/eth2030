# PR 73 tracking cleanup status

Original review head: `3ec63b64dd2b963a6829fcf3fccf57db3729b326`. Merge base for preserved records: `38ea9259901c7873f483752ea390962e26563648`. Target master: `98716284b049933888bcd68d64c1957004d759b2`. Shared tracking policy: `42a9ceb86dea2fe4a74f01fd5e2360d6f21af0e7`. The sections below preserve this branch snapshot, including any inherited feature records; their historical validation is not a new result for this cleanup. The exact original diff, including removed text, is in [legacy-tracking.patch](legacy-tracking.patch).

Cleanup validation is pending. Publication must be a fast-forward update of the existing review branch. Draft status and dependency order are unchanged.

## Preserved feature tracking

## Glamsterdam EIP-8024 bytecode and activation (2026-11-18)

Corrected native DUPN/SWAPN/EXCHANGE immediate permutations and reserved ranges, and activated the instructions at canonical Amsterdam. Published bytecode vectors, all valid depths/pairs, stack and jump boundaries, and native transactions/fork transitions pass. Affected suites, targeted race/vet and the local build pass. See [verification evidence](../../verification/glamsterdam-stack-opcodes-20261118.md). Independent-client and full release gates remain open.

