# Independent snap/2 BAL peer

- [x] Read the BAL positional/unavailable/commitment contract, current snap/2 packet/query implementation and native client/recovery entrypoints.
- [ ] Connect the native encrypted P2P/client to the independent Geth v1.17.10 production snap/2 server and storage/query path.
- [ ] Verify typed independent nonempty BALs, unavailable holes, soft-budget truncation, subsequent recovery and wrong-header commitments over real TCP/RLPx.
- [ ] Retain raw requests/responses and source identities; keep serving/client component qualification distinct from whole-node import and canonical recovery.
- [ ] Run full affected race suites/builds and publish an independent master-target PR.
