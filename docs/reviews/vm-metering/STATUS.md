# Native VM conformance correction in progress

Source e9223d6d corrects MUL's five-gas tier, failed-precompile gas exhaustion, EIP-1052 empty-account results, zero-length memory ranges and word-rounding overflow. Fourteen actual EVM boundary controls (Osaka and Amsterdam) match unchanged Geth, after first failing on the earlier native implementation. Affected VM/execution/backend tests pass. Earlier unit assertions encoding three-gas MUL and zero-length memory expansion were corrected to the normative behavior.

Complete modern EELS reruns, remaining historical VM categories, full/race/integration/vet/build qualification and dedicated PR publication are still in progress. Unknown and failing cases remain failures. This record does not claim completion of the full corpus or release campaign.
