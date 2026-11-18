# Current merged gas implementation checkpoint

The clean development baseline `52e35b68` preserves 19 differing Amsterdam observations out of 22. The first implementation snapshot in `first-transition-match.json` matches all 22 Amsterdam scenarios. It explicitly records its dirty source state. `transaction-checkpoint.json` repeats the full signed corpus from clean source `c6cdf15ac7377773a5c05d8948bf8b292dd8d700` and matches all 22 Amsterdam cases. The five Osaka differences are unchanged transition-versus-outer-validator observations from the existing corpus, not newly introduced block acceptance failures.

The native implementation adds fixed current parameters, state-independent intrinsic/floor calculation, two-dimensional reservations/settlement, LIFO borrowing/refill/baseline restoration, cross-frame carry, conditional creation costs and authorization pre-execution commit/rollback. The reference remains unmodified Geth v1.17.10 and is an implementation observation rather than normative authority.

This is not whole-program acceptance. Outer block/header/pool/RPC composition, 7702/application and system-call vectors, official fixtures, performance, CL/multiclient and release gates remain open. Existing tests relying on old 21,000-gas new-account transfers or old state pricing currently fail and are retained pending source-based fixture/implementation analysis. They are not waived or silently reclassified as passing.

Read-only local EIPs are older `49beda9`; the chosen source is `3db38a3`. EIPs 8037, 8038, 2780, 7778, 7976, 7981 and 7702 were fetched at that exact revision and read in full. Current values include CPSB 1530, account/slot state sizes 120/64, the fixed 2^24 execution cap and the 2^32-1 total cap. Open alternatives 12581/12582 are not mixed into this implementation.

## Outer consumer implementation checkpoint

The expanded signed corpus adds a 64 KiB deployment and matches all 23 Amsterdam scenarios. Actual Node HTTP admission accepts its 110M total gas, the builder includes it with a second transaction, two Nodes import the block, and fresh receiver readback preserves code/head/root/receipts. State/header gas is 100,453,680 while user receipt cumulative gas is 100,516,383. A header rehashed with the wrong scalar receipt sum is rejected by actual block accounting. Source paths now use canonical total caps, per-dimension builder reservations and gross-execution/net-state aggregation. This does not close RPC mode, authorization/application, system margin or full-suite qualification.

The full-suite pass also exposed a snap/2 rebase case where an unchanged pending storage account lacks proof-root binding in the fresh adapter. That actual protocol regression is being reproduced and repaired in PR 81 independently, rather than hidden by increasing a transaction fixture's budget.
