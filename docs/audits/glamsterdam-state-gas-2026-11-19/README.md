# Current merged gas implementation checkpoint

The clean development baseline `52e35b68` preserves 19 differing Amsterdam observations out of 22. The first implementation snapshot in `first-transition-match.json` matches all 22 Amsterdam scenarios. It explicitly records its dirty source state. `transaction-checkpoint.json` repeats the full signed corpus from clean source `c6cdf15ac7377773a5c05d8948bf8b292dd8d700` and matches all 22 Amsterdam cases. The five Osaka differences are unchanged transition-versus-outer-validator observations from the existing corpus, not newly introduced block acceptance failures.

The native implementation adds fixed current parameters, state-independent intrinsic/floor calculation, two-dimensional reservations/settlement, LIFO borrowing/refill/baseline restoration, cross-frame carry, conditional creation costs and authorization pre-execution commit/rollback. The reference remains unmodified Geth v1.17.10 and is an implementation observation rather than normative authority.

This is not whole-program acceptance. Outer block/header/pool/RPC composition, 7702/application and system-call vectors, official fixtures, performance, CL/multiclient and release gates remain open. Existing tests relying on old 21,000-gas new-account transfers or old state pricing currently fail and are retained pending source-based fixture/implementation analysis. They are not waived or silently reclassified as passing.

Read-only local EIPs are older `49beda9`; the chosen source is `3db38a3`. EIPs 8037, 8038, 2780, 7778, 7976, 7981 and 7702 were fetched at that exact revision and read in full. Current values include CPSB 1530, account/slot state sizes 120/64, the fixed 2^24 execution cap and the 2^32-1 total cap. Open alternatives 12581/12582 are not mixed into this implementation.

## Outer consumer implementation checkpoint

The expanded signed corpus adds a 64 KiB deployment and matches all 23 Amsterdam scenarios. Actual Node HTTP admission accepts its 110M total gas, the builder includes it with a second transaction, two Nodes import the block, and fresh receiver readback preserves code/head/root/receipts. State/header gas is 100,453,680 while user receipt cumulative gas is 100,516,383. A header rehashed with the wrong scalar receipt sum is rejected by actual block accounting. Source paths now use canonical total caps, per-dimension builder reservations and gross-execution/net-state aggregation. This does not close RPC mode, authorization/application, system margin or full-suite qualification.

The full-suite pass also exposed a snap/2 rebase case where an unchanged pending storage account lacks proof-root binding in the fresh adapter. That actual protocol regression is being reproduced and repaired in PR 81 independently, rather than hidden by increasing a transaction fixture's budget.

## Expanded boundary checkpoint

Thirty signed Amsterdam scenarios now match independent Geth, including zero base fee and GASPRICE, all four delegated-call families and an original 256-bit call gas request with zero low bits. The additional Osaka controls also match; its six remaining differences are the previously classified outer declared-gas-cap boundary, including the added large-code control.

The zero-base-fee scenario first reproduced wrong balances/root: the native helper incorrectly charged the fee cap when base fee was zero. EIP-1559 still requires min(cap, base plus priority), including zero. Its source was read in full at the pinned EIPs revision before correction. Delegation resolution now covers its separate target access and code context, with original/delegated addresses recorded at the specified pre/post-state gas boundaries. New direct VM tests distinguish SSTORE's stipend failure from post-read write-cost failure and delegation-target OOG from a later insufficient-balance child.

The HTTP block control additionally proves declared state reservation: a third 60M-total-gas transaction cannot fit the first block's remaining state budget, and its correctly rehashed three-transaction body is rejected by the gas-pool consumer. It succeeds in the fresh next block, whose base fee matches the independent Geth formula based on the parent's bottleneck gas.

Current execution/VM suites pass after preserving fresh-account transfer/creation obligations with sufficient budgets, correcting the canonical 15000 zero-value intrinsic expectation, and keeping the PQ pricing test's explicitly pre-floor fork schedule coherent. The mixed-receipt case now requires successful status for every transaction. The overall suite, RPC/system/authorization/application campaigns and release qualification remain unfinished.

## Independent system transition gap

The production EIP-2935 helper still uses address 0x0F792be4B0c0cb4DAE440Ef133E90C0eCD48CCCC and an 8192 window. The complete selected EIP-2935 text instead specifies 0x0000F90827F1C53a10cb7A02335B175320002935 and 8191. This is a separate full-state interoperability gap; native two-node agreement cannot qualify it. EIPs 2935 and 4788 permit direct storage optimization, but EVM invocation is preferred for observing absent/custom code. The system transition/deployment correction will receive its own feature branch/review, with canonical runtime and independent state consumers. Final system/multiclient release gates remain open.

## Signed authorization and actual consumer follow-up

Ten signed authority controls cover valid existing/new authorities, two distinct pre-execution failure points, frame revert, skipped nonce warming, max nonce, clear/reset once, existing indicators and sender authorization. Explicit expected charges supplement the independent comparison. Added GAS-dependent storage controls make LIFO refill and ancestor repayment observable in actual execution.

Historical Prague authorization tests select Prague explicitly; canonical Amsterdam tests fund the actual account/slot costs and preserve success, state, nonce, fees and requests. Test block assemblers use max(sum gross execution, sum state), while receipts retain paid cumulative gas. Tagged tests include a Prague control that fits twenty new slots, whereas the same Amsterdam block admits nine because the next declared state reservation does not fit. Funded/unfunded reservoir controls pay the same 125026 gas for a single slot.

Actual Node HTTP settlement additionally checks both blocks' advertised blockValue and fee-recipient balance, final sender debit, deferred transaction and next SDK base fee. Ten compiling guard mutations detect state pricing, LIFO refill, ancestor repayment, exceptional baseline, authorization commit/rollback, gross execution, header bottleneck, import reservation and system margin. The first rollback mutation escaped an early failure with no prior write; this failed sensitivity result is preserved and repaired by a late pre-execution failure after successful authorization writes.

Full ordinary/tagged and broad affected race/vet/build passed before these last boundary additions. Final evidence records the subsequent affected repeat and clean signed checkpoint separately. This does not qualify RPC modes or final protocol choices.

Clean expanded checkpoint: `2cde17ad`, 42/42 Amsterdam signed matches. Full 212 packages, tagged 27, broad 96 race packages and final 33 affected race packages pass, with vet/build. [Current evidence](../../verification/glamsterdam-state-gas-20261119.md) supersedes earlier in-progress test failures without erasing their logs.
