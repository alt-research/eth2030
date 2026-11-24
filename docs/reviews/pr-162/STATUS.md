# Status

Discovered while hardening the G01 native blockchain-fixture consumer against false expected-invalid passes. Current DecodeBlockRLP accepts trailing bytes; Stream.Bytes can panic and Stream.List can accept an overflowing claimed length. Retain the failing tests and fix these independently from blockchain execution changes.

The declared length is now compared to remaining bytes before integer addition, and DecodeBlockRLP requires EOF after its single top-level block. Small negative regressions, both full affected race suites, the complete Go suite and both node builds pass. No large payload allocation is required by the overflow controls. Final PR/CI publication remains separate.
