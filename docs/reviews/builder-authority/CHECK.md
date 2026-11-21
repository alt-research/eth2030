# Legacy Builder Engine retirement

- [x] Read selected Gloas builder/authority/signing/deposit rules and identify unsupported legacy RPC/wire shapes.
- [x] Record user selection to retire legacy extensions in favor of standard CL/EIP-8282.
- [x] Reproduce false capability/method acceptance with actual HTTP/JWT requests.
- [x] Remove dispatch, capability claims, synthetic stake registry, incomplete signature conversion and node auction feed.
- [x] Verify canonical request extraction, actual signed CL reference positives/mutations and captured standard Engine traffic.
- [ ] Publish master-targeted PR and confirm exact-head CI and cumulative merge compatibility.
