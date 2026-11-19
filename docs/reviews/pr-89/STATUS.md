# Canonical Osaka ModExp activation

The native EVM selected Prague ModExp even after Osaka. EIP-7883's minimum/complexity/large-exponent prices and EIP-7823's 1024-byte bounds were therefore absent from production execution. For the official 1024-byte boundary vector, the precompile charged 5,461 rather than 32,768 gas. This changes the signed transaction's storage accounting observation, sender balance and state root.

This correction threads Osaka through canonical chain/VM rules, selects the bounded native ModExp for Osaka and later forks, and retains earlier pricing. The implementation checks full 256-bit length fields before allocations, uses saturating gas results for oversized headers, and leaves shared pre-fork maps intact. No SDK execution or native hooks are injected into the oracle.

Independent unchanged Geth v1.17.10 comparisons cover zero/short inputs, 32/33/64/1024-byte boundaries, each over-limit field and oversized uint256 headers. The actual canonical EVM configuration is checked immediately before/at/after activation. Full 212-package tests, Core integration tests, affected race tests, full vet and build pass. The added activation regression also passes on the final source.

The broader official Osaka state corpus still exposed a separate EIP-161 touched-empty-account finalization defect; that lifecycle fix and fixture runner are separate reviews. This review does not claim that the complete official corpus or Glamsterdam release campaign is already green.

Normative inputs: [EIP-7823](https://github.com/ethereum/EIPs/blob/3db38a3725105cb67b8804ac3e147967113bc3ca/EIPS/eip-7823.md), [EIP-7883](https://github.com/ethereum/EIPs/blob/3db38a3725105cb67b8804ac3e147967113bc3ca/EIPS/eip-7883.md). Discovery corpus: EEST v5.4.0 develop asset SHA-256 `3e2b02d49fe903eda4fd8caca5cbf0d139c470e97e1de9a85299b1b034f97099`.
