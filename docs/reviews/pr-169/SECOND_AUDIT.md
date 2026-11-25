# Second Group B audit, 2026-11-26

The previous 36-criterion qualification missed four implementation defects. Passing test identities did not establish complete execution coverage. This audit revisits all 12 tasks and adds concrete missing boundary cases rather than treating the existing completion labels as evidence.

## Normative findings and corrections

The upstream EIPs master was rechecked and remains `ade9e9cab7e148bba4cd293f916a4a189cd7a8aa`. Reference files remain unchanged. The Group B register now explicitly hashes EIP-7702 and EIP-2929 in addition to its previous sources.

| Finding | Normative requirement | Correction and executable evidence |
| --- | --- | --- |
| Direct precompile frame entry could omit its target from BAL | EIP-8141 Behavior charges entry before dispatch; EIP-7928 Scope and Inclusion includes called precompiles, including calls that fail after entry access | Record the resolved target after entry gas passes, before dispatch. `TestFrame8141DirectPrecompileBALGasBoundary` distinguishes 99, 100 and 118 gas for identity: no entry access, accessed-but-OOG and success. |
| Delegation costs were missing from CALLCODE/DELEGATECALL/STATICCALL and deducted inside CALL's child | EIP-7702 Delegation indicator/Gas Costs applies to all four instructions; EIP-8038 Account access pricing sets the selected cold cost; EIP-2929 charges access before the 63/64 split | Reuse the existing delegation resolver in the parent dynamic-gas path, before EIP-150 forwarding. Retain the separate top-level Frame dispatch charge. `TestFrame8141DelegatedCallGasAndForwarding` checks all four instructions, warm/cold delegate access and finite/256-bit gas requests with exact gas and returned child GAS. |
| SSTORE's implicit read disappeared when later execution/state gas checks failed | EIP-7928 Gas Validation Before State Access/SSTORE retains actual reads once the stipend and access sentries pass | Record the implicit storage read during gas calculation. `TestFrame8141SstoreBALReadSurvivesPostStateFailure` checks the 5305/5306 gas sentry edge and a later state-gas failure at 15105 gas; no final write is published. |
| SELFDESTRUCT could record an unaccessed beneficiary, or omit an accessed zero-balance beneficiary | EIP-7928 requires base/access gas before beneficiary access; EIP-8038 preserves cold beneficiary access even when no ETH moves | Validate pre-state access cost before reading/recording the beneficiary, then record it even when the balance is zero. `TestFrame8141SelfdestructBALPreStateBoundary` checks funded/unfunded callers at 11002/11003 gas. |

`TestFrame8141DelegationBALBeforeFailedChildChecks` separately checks that original/delegated account accesses occur only after their own sentries pass and precede a failed child balance check. `TestFrame8141DelegationWarmnessSurvivesChildRevert` verifies that accesses charged by the parent remain warm after a child reverts; whole-Frame/batch rollback tests continue to require their enclosing accesses to revert.

The delegation call-body tests execute the same cold-access programs in the unchanged upstream Geth v1.17.10 Amsterdam VM, comparing gas and output. The Frame-only entry cost is removed from the comparison. This is an independent call-body oracle, not a claim that Geth implements the selected Frame envelope.

## Negative control

The current regression tests are executed with the four original production files from `95d4be4e4a09ba3938cae9d3064def24e2ad231b` substituted using a Go overlay. All six new parent tests fail (29 failed test/subtest events). The same tests pass with the fixes. No oracle or production reference file is edited to manufacture a passing result. The qualification record retains the original-source identity and negative-run digest.

## Complete task review

| Task | Review outcome |
| --- | --- |
| HGT-002 | Canonical field order, RLP widths/constraints, transaction/signature hashes and witness-length commitments retain native independent-golden consumers. |
| HGT-003 | Scope/identity/default-account approval, escrow, nonce and nested approval rollback remain covered. The common call dispatch fixes preserve ADDRESS and caller context. |
| HGT-010 | Transaction/frame/signature parameters, future structure versus completed result visibility, invalid indices, data bounds and non-Frame rejection retain concrete consumers. |
| HGT-011 | Fixed delegation charging and forwarding, including independent call-body comparison. Existing independent execution/state/blob fee, floor/refund, escrow and gas-pool tests remain required. |
| HGT-012 | Fixed parent delegation warming across child revert; retained call/Frame/batch/transaction state-gas ownership, logs, approval, transient and outer warmness rollback tests. |
| HGT-013 | Public canonical execution still uses the shared native processor and builder/importer. No additional executor or authority path was introduced. Shared prerequisite VM code is requalified. |
| HGT-014 | Canonical payer/results/status/gas/log RLP and index derivation are unchanged; failed/skipped result constraints and import commitment rejection remain required. |
| HGT-015 | Fixed direct precompile, SSTORE, delegation and SELFDESTRUCT access evidence at pre/post-state boundaries. Existing exact published BAL, system update, item and block-byte bounds remain required. |
| HGT-016 | Canonical and historical public encodings, FileDB restart and replay remain unchanged. No storage-schema migration is introduced. |
| HGT-017 | Rechecked the complete keyed state/consumption/BAL/CREATE/rollback implementation and vectors. The plan's controlled-profile allowance still applies; the conflicting fee example does not authorize public activation or close D01. |
| HGT-018 | Rechecked the verbatim ordinary recent-root runtime, deployment, source/slot/window/overwrite and canonical fork-switch consumers. Public pool grammar/dependency exceptions remain Group C. |
| HGT-019 | Protocol signature schemes, default accounts, low-s/length/message constraints and arbitrary-witness separation remain required before execution. |

The machine register now requires the new boundary consumers; deleting them or the delegated-code normative source fails the coverage checker. No existing criterion was removed or relaxed.

## Compatibility and qualification

These are consensus corrections for the selected Frame profile. Valid historical encodings remain readable. A research chain containing previously miscomputed Frame gas or BAL commitments can fail replay under the corrected rules; such blocks must not have their roots silently rewritten. Rebuild that research history from a valid checkpoint if it exercised the defective paths. No public mainnet activation or database rewrite is performed by this change.

Fresh build, vet, full/focused-race suites, the historical differential baseline and Python checks are recorded in the updated Group A/B evidence. Earlier local and hosted runs are retained as predecessor evidence; they do not qualify the new source. Final hosted status is reported against the final PR head.
