# Current merged gas implementation checkpoint

The clean development baseline `52e35b68` preserves 19 differing Amsterdam observations out of 22. The first implementation snapshot in `first-transition-match.json` matches all 22 Amsterdam scenarios. It explicitly records its dirty source state; the following committed rerun will identify the exact source revision. The five Osaka differences are unchanged transition-versus-outer-validator observations from the existing corpus, not newly introduced block acceptance failures.

The native implementation adds fixed current parameters, state-independent intrinsic/floor calculation, two-dimensional reservations/settlement, LIFO borrowing/refill/baseline restoration, cross-frame carry, conditional creation costs and authorization pre-execution commit/rollback. The reference remains unmodified Geth v1.17.10 and is an implementation observation rather than normative authority.

This is not whole-program acceptance. Outer block/header/pool/RPC composition, 7702/application and system-call vectors, official fixtures, performance, CL/multiclient and release gates remain open. Existing tests relying on old 21,000-gas new-account transfers or old state pricing currently fail and are retained pending source-based fixture/implementation analysis. They are not waived or silently reclassified as passing.

Read-only local EIPs are older `49beda9`; the chosen source is `3db38a3`. EIPs 8037, 8038, 2780, 7778, 7976, 7981 and 7702 were fetched at that exact revision and read in full. Current values include CPSB 1530, account/slot state sizes 120/64, the fixed 2^24 execution cap and the 2^32-1 total cap. Open alternatives 12581/12582 are not mixed into this implementation.
