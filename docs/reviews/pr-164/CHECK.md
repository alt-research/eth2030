# Independent snap/2 BAL peer

- [x] Read the BAL positional/unavailable/commitment contract, current snap/2 packet/query implementation and native client/recovery entrypoints.
- [x] Connect the native encrypted P2P/client to the independent Geth v1.17.10 production snap/2 server and storage/query path.
- [x] Verify typed independent nonempty BALs, unavailable holes, soft-budget truncation, subsequent recovery and wrong-header commitments over real TCP/RLPx.
- [x] Retain raw requests/responses and source identities; keep serving/client component qualification distinct from whole-node import and canonical recovery.
- [x] Run full affected race suites, final profile race, vet, both node builds and two executed behavioral mutation controls.
- [x] Publish master-target PR #164 and verify GitHub reports no conflicts.
- [x] Verify the exact functional head native-geth-snap2 CI job and all nine standard Go/fixture checks pass.
- [ ] Confirm final metadata-head checks and cumulative merge orders with the other open master PRs.
