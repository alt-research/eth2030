# Signed gas corpus baseline

Native source: `a0636d1faafa8085137181e1d4eb30dbe707b7e2` (clean). Reference: `github.com/ethereum/go-ethereum v1.17.10`. EIPs: `3db38a3725105cb67b8804ac3e147967113bc3ca`.

This records baseline differences; no mismatch is waived or counted as release parity. Full inputs and observations are in [native-baseline.json](native-baseline.json).

| Fork | Case | Native valid/status | Reference valid/status | Native user gas | Reference user gas | Differing fields |
| --- | --- | --- | --- | --- | --- | --- |
| amsterdam_merged_text_candidate | above_execution_cap | True/1 | True/1 | 21000 | 15000 | user_gas_used, block_gas_used, block_execution_gas, post_state_root, accounts |
| amsterdam_merged_text_candidate | above_total_cap | False/0 | False/0 | 0 | 0 | none |
| amsterdam_merged_text_candidate | access_list | True/1 | True/1 | 25300 | 23228 | user_gas_used, block_gas_used, block_execution_gas, post_state_root, accounts |
| amsterdam_merged_text_candidate | allocate_slot | True/1 | True/1 | 43106 | 125026 | user_gas_used, block_gas_used, block_execution_gas, block_state_gas, post_state_root, accounts |
| amsterdam_merged_text_candidate | allocate_then_clear | True/1 | True/1 | 34570 | 21770 | user_gas_used, block_gas_used, block_execution_gas, post_state_root, accounts |
| amsterdam_merged_text_candidate | allocate_then_exception | True/0 | True/0 | 500000 | 500000 | none |
| amsterdam_merged_text_candidate | allocate_then_revert | True/0 | True/0 | 43112 | 27112 | user_gas_used, block_gas_used, block_execution_gas, post_state_root, accounts |
| amsterdam_merged_text_candidate | calldata_floor | True/1 | True/1 | 26120 | 23192 | user_gas_used, block_gas_used, block_execution_gas, post_state_root, accounts |
| amsterdam_merged_text_candidate | clear_original_slot | True/1 | True/1 | 21206 | 21685 | user_gas_used, block_gas_used, block_execution_gas, post_state_root, accounts |
| amsterdam_merged_text_candidate | create_empty | True/1 | True/1 | 53006 | 207606 | user_gas_used, block_gas_used, block_execution_gas, block_state_gas, post_state_root, accounts |
| amsterdam_merged_text_candidate | create_revert | True/0 | True/0 | 53064 | 24320 | user_gas_used, block_gas_used, block_execution_gas, post_state_root, accounts |
| amsterdam_merged_text_candidate | create_selfdestruct | True/1 | True/1 | 80210 | 322730 | user_gas_used, block_gas_used, block_execution_gas, block_state_gas, post_state_root, accounts, logs |
| amsterdam_merged_text_candidate | existing_value | True/1 | True/1 | 21000 | 21000 | logs |
| amsterdam_merged_text_candidate | existing_zero_value | True/1 | True/1 | 21000 | 15000 | user_gas_used, block_gas_used, block_execution_gas, post_state_root, accounts |
| amsterdam_merged_text_candidate | funded_reservoir_allocate | True/1 | True/1 | 43106 | 125026 | user_gas_used, block_gas_used, block_execution_gas, block_state_gas, post_state_root, accounts |
| amsterdam_merged_text_candidate | funded_reservoir_exception | True/0 | True/0 | 16977216 | 16777216 | user_gas_used, block_gas_used, block_execution_gas, post_state_root, accounts |
| amsterdam_merged_text_candidate | funded_reservoir_nested_refill | True/1 | True/1 | 36665 | 24185 | user_gas_used, block_gas_used, block_execution_gas, post_state_root, accounts |
| amsterdam_merged_text_candidate | nested_refill_revert | True/1 | True/1 | 45837 | 128157 | user_gas_used, block_gas_used, block_execution_gas, block_state_gas, post_state_root, accounts |
| amsterdam_merged_text_candidate | nested_refill_success | True/1 | True/1 | 36665 | 24185 | user_gas_used, block_gas_used, block_execution_gas, post_state_root, accounts |
| amsterdam_merged_text_candidate | new_account_runtime_oog | True/1 | True/0 | 21000 | 21000 | receipt_status, post_state_root, accounts |
| amsterdam_merged_text_candidate | new_account_value | True/1 | True/1 | 21000 | 204600 | user_gas_used, block_gas_used, block_state_gas, post_state_root, accounts, logs |
| amsterdam_merged_text_candidate | partial_reservoir_allocate_clear | True/1 | True/1 | 34570 | 21770 | user_gas_used, block_gas_used, block_execution_gas, post_state_root, accounts |
| osaka_control | above_execution_cap | True/1 | False/0 | 21000 | 0 | validity, receipt_status, user_gas_used, block_gas_used, block_execution_gas, post_state_root, accounts |
| osaka_control | above_total_cap | False/0 | False/0 | 0 | 0 | none |
| osaka_control | access_list | True/1 | True/1 | 25300 | 25300 | none |
| osaka_control | allocate_slot | True/1 | True/1 | 43106 | 43106 | none |
| osaka_control | allocate_then_clear | True/1 | True/1 | 34570 | 34570 | none |
| osaka_control | allocate_then_exception | True/0 | True/0 | 500000 | 500000 | none |
| osaka_control | allocate_then_revert | True/0 | True/0 | 43112 | 43112 | none |
| osaka_control | calldata_floor | True/1 | True/1 | 26120 | 26120 | none |
| osaka_control | clear_original_slot | True/1 | True/1 | 21206 | 21206 | none |
| osaka_control | create_empty | True/1 | True/1 | 53006 | 53006 | none |
| osaka_control | create_revert | True/0 | True/0 | 53064 | 53064 | none |
| osaka_control | create_selfdestruct | True/1 | True/1 | 80210 | 80210 | post_state_root, accounts |
| osaka_control | existing_value | True/1 | True/1 | 21000 | 21000 | none |
| osaka_control | existing_zero_value | True/1 | True/1 | 21000 | 21000 | none |
| osaka_control | funded_reservoir_allocate | True/1 | False/0 | 43106 | 0 | validity, receipt_status, user_gas_used, block_gas_used, block_execution_gas, post_state_root, accounts |
| osaka_control | funded_reservoir_exception | True/0 | False/0 | 16977216 | 0 | validity, user_gas_used, block_gas_used, block_execution_gas, post_state_root, accounts |
| osaka_control | funded_reservoir_nested_refill | True/1 | False/0 | 36665 | 0 | validity, receipt_status, user_gas_used, block_gas_used, block_execution_gas, post_state_root, accounts |
| osaka_control | nested_refill_revert | True/1 | True/1 | 45837 | 45837 | none |
| osaka_control | nested_refill_success | True/1 | True/1 | 36665 | 36665 | none |
| osaka_control | new_account_runtime_oog | True/1 | True/1 | 21000 | 21000 | none |
| osaka_control | new_account_value | True/1 | True/1 | 21000 | 21000 | none |
| osaka_control | partial_reservoir_allocate_clear | True/1 | False/0 | 34570 | 0 | validity, receipt_status, user_gas_used, block_gas_used, block_execution_gas, post_state_root, accounts |

The Osaka execution-cap and newly created SELFDESTRUCT control differences are retained as baseline defects. The latter is independently repaired by PR 72 and remeasured in the aggregate corpus. Amsterdam discrepancies include actual included runtime-OOG status, state creation pricing, floor/refund/resource counters and fee-dependent roots. A root difference caused by a gas/fee mismatch is not counted as an independently proven storage defect.

Comparison counts (total, with differing observable fields): `{"amsterdam_merged_text_candidate": [22, 20], "osaka_control": [22, 6]}`.
