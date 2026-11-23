# Engine blob query validation

Authorities: Amsterdam/Osaka Engine API at 1ca0818569467cff85b7f41372f22e728d5daa14; EIP-8070 at 81a90efc672bdd6130c813d8dae005e5b3a8ce5b; merged EIP alignment PR 12701. The Osaka contract explicitly requires null during sync/general unavailability for V2/V3 and prohibits returning older blob-proof versions as PeerDAS data. Retrieval takes DATA_32 hashes, so well-formed unknown prefixes are missing data, not transaction-validity errors.

Original behavioral failures are retained in RED.txt. They include V4 positional arrays during sync/closure, null-cell/proof pairing, V2 null input reaching the backend, cached V2/V3 data served during sync, and implicit legacy-proof conversion. Test-only compilation fixes are not counted as behavioral negative controls.

Fresh local complete engine/backend/txpool/node suites and their race runs pass. Native TCP/KZG promotion and receiver restart also exercise real HTTP V2/V3 against the independent Geth decoder, with all 128 proof bytes compared to the authenticated source. V4 cells 0/127 are independently verified. Actual native FCU targets drive healthy/sync/recovered query states. These are controlled API regressions, not a new historical catchup or original-04 stall reproduction.

Both node entrypoints build and affected vet passes. All runtime gas/EVM/consensus-fork logic remains unchanged. The cumulative full-package and merge/CI records are maintained in the current specification-closure review.
