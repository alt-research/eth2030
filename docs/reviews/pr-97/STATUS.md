# Ordinary transaction context isolation qualified

The actual published Uniswap V2 sequence exposed a refund leak: after pair mint, the following token transfer paid 31,505 native Osaka gas instead of the independent 34,305. The ordinary pre-Amsterdam message path also retained transient storage across transactions. Both counters must be transaction-local under EIP-2200/3529/1153, including when a live state is reused without a commit.

The common message entry now clears the previous refund and transient storage before execution. Child-frame metering and rollback are unchanged. Three consecutive independently signed transactions share the same state and block gas pool without a Commit; receipt gas, complete roots and persistent TLOAD side effects match unchanged Geth v1.17.10 on Osaka and Amsterdam. Removing either reset is detected by its own behavioral mutation. The actual Uniswap transfer sequence no longer diverges.

Full Go tests, core integration, execution/state/Node/backend race checks, full vet and build pass. A clean complete official rerun at source 2d827990a2114b521cbe4c3a8874740dab632782 passes all 14,665 Osaka and 16,908 Amsterdam cases. The larger real application sequence and its wallet gas-boundary characterization stay in a separate review.

Normative sources: EIPs 2200, 3529 and 1153 at selected EIPs revision 3db38a3725105cb67b8804ac3e147967113bc3ca. This is transaction-boundary correctness, not a change to the selected EIP-8037 LIFO model.
