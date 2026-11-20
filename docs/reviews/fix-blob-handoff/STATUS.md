# Engine cells remain available across promotion

The audit reproduces a null result when a full-pool snapshot precedes admission and a later staging enumeration follows removal. Snapshot authenticated requested staging cells/proofs atomically first, retaining private byte copies, then read the destination full pool. The existing complete/partial row union stays intact. No lock is held across KZG computation or across the two pool implementations, and missing cells/proofs remain null.

The audit schedule is now a normal regression using a real native pool, signed transaction and KZG reconstruction/verification. The earlier five c64ac150 scenarios still pass; an additional caller-mutation assertion verifies staging snapshots own both cells and proofs. Existing real encrypted-peer/node Engine/restart assertions and deadlines are unchanged.

On Go 1.26.11, all 218 full package suites, affected eth/sparse/node/backend race suites, ten repetitions of the original native peer/restart consumer, full vet and native build pass. Source is based directly on master 7068cd97. Normative reference: selected EIP-8070 Engine getBlobs cell/proof availability; this changes no wire layout or consensus rule. Publication and new-head CI are verified separately. No EL master merge is performed.
