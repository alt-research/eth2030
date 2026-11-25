# Official signed Osaka regression fixtures

These five complete, unmodified test objects are copied from the official ethereum/execution-spec-tests v5.4.0 `fixtures_develop.tar.gz` asset. Packaging each selected object into its own JSON file changes only the outer collection, not its input, signed transaction, expected root/logs, metadata or post-state. The original project is MIT-licensed; see [the upstream license](https://github.com/ethereum/execution-spec-tests/blob/v5.4.0/LICENSE).

Pinned asset: SHA-256 `3e2b02d49fe903eda4fd8caca5cbf0d139c470e97e1de9a85299b1b034f97099`. The full CI campaign replays all 1,223 Osaka state subtests from the complete selected archive directory, not just these five smoke vectors. The inventory records original paths, exact case names and original collection file checksums.

## Source inventory

```json
[
  {
    "path": "envelope-0.json",
    "upstream_file": "fixtures/state_tests/osaka/eip7825_transaction_gas_limit_cap/test_maximum_gas_refund.json",
    "case": "tests/osaka/eip7825_transaction_gas_limit_cap/test_tx_gas_limit.py::test_maximum_gas_refund[fork_Osaka-state_test-exceed_gas_refund_limit_False]",
    "upstream_file_sha256": "97d34f0140d071b9f19c9fb97f9f50ad11f580d9fbc3162f4cde025cc3cee52f"
  },
  {
    "path": "envelope-1.json",
    "upstream_file": "fixtures/state_tests/osaka/eip7825_transaction_gas_limit_cap/test_transaction_gas_limit_cap.json",
    "case": "tests/osaka/eip7825_transaction_gas_limit_cap/test_tx_gas_limit.py::test_transaction_gas_limit_cap[fork_Osaka-tx_gas_limit_cap_over3-state_test]",
    "upstream_file_sha256": "cc8c8dee42e304931b47d414038fff9f742bbde35e29cb1b9f32d5af663681f7"
  },
  {
    "path": "envelope-2.json",
    "upstream_file": "fixtures/state_tests/osaka/eip7823_modexp_upper_bounds/test_modexp_boundary.json",
    "case": "tests/osaka/eip7823_modexp_upper_bounds/test_eip_mainnet.py::test_modexp_boundary[fork_Osaka-state_test-base-boundary-1024-bytes]",
    "upstream_file_sha256": "a63acabe108b6c4f9db0e2de8a64661764f499e876dd96c036dd6533564deb98"
  },
  {
    "path": "envelope-3.json",
    "upstream_file": "fixtures/state_tests/osaka/eip7594_peerdas/test_valid_max_blobs_per_tx.json",
    "case": "tests/osaka/eip7594_peerdas/test_max_blob_per_tx.py::test_valid_max_blobs_per_tx[fork_Osaka-blob_count_1-state_test]",
    "upstream_file_sha256": "69774bfa1c8daeb601fd1eda36fc59e6c8b2078a19415454096197ddf547b0d8"
  },
  {
    "path": "envelope-4.json",
    "upstream_file": "fixtures/state_tests/osaka/eip7825_transaction_gas_limit_cap/test_transaction_gas_limit_cap.json",
    "case": "tests/osaka/eip7825_transaction_gas_limit_cap/test_tx_gas_limit.py::test_transaction_gas_limit_cap[fork_Osaka-tx_gas_limit_cap_over0-state_test]",
    "upstream_file_sha256": "cc8c8dee42e304931b47d414038fff9f742bbde35e29cb1b9f32d5af663681f7"
  }
]
```
