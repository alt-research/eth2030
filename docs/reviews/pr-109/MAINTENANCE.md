# Native Merkle history maintenance

Authority: [EIP-1186](https://eips.ethereum.org/EIPS/eip-1186), [EIP-1898](https://eips.ethereum.org/EIPS/eip-1898), and the selected native MPT state-root encoding. This auxiliary index changes neither block hashes nor consensus selection.

New validated ordinary-MPT blocks retain their immutable account/storage nodes and code in the same atomic database batch as their validation marker. Historical reads authenticate every loaded commitment and the full original root. Orphan roots remain accessible under the existing exact-header policy. The binary overlay retains its separate checkpoint format.

To upgrade an existing full-history native database:

1. Build both executables from the reviewed source in `pkg`:

   ```sh
   go build -o eth2030 ./cmd/eth2030
   go build -o eth2030-state-history ./cmd/eth2030-state-history
   ```

2. Stop that execution process cleanly. Retain its original data directory, chain genesis and normal startup arguments. The database lock excludes concurrent node/maintenance access.
3. Run the tool with the original files:

   ```sh
   ./eth2030-state-history --datadir=/original/native/datadir --genesis=/original/genesis.json
   ```

4. Retain the JSON result (`head`, `number`, `indexed`, `existing`) and command exit status. SIGINT/SIGTERM cancels between authenticated operations; each completed block batch remains resumable. Rerun the same command after interruption. The maintenance entrypoint opens only the required storage/genesis primitives, avoiding normal startup replay before cancellation can act.
5. Start the reviewed native executable with the original node arguments. Recheck the original finalized block, exact historical header, contract return value and account/storage proof against the independent peer. Record actual recovery time and memory/disk usage separately from unit results.

This builds the retained, execution-validated canonical path and does not select a new head, import peer data or scrub every previously indexed root. Newly read indexed roots are fully authenticated; missing or corrupt published nodes fail. A database missing pre-pivot full history cannot be upgraded by inventing that history. Its explicit error and completed prefix remain available for diagnosis. Existing auxiliary markers are reused; a repeated index run is idempotent. Historical roots not yet indexed retain authenticated, cancellable replay compatibility.

Capacity depends on the retained state and history. Full snapshots are materialized through the existing native secure snapshot type; this is not a mainnet memory/throughput claim. Original-process migration and the predeclared operating campaign remain acceptance evidence to collect.
