# Group B specification and implementation review

The execution scope is the 12 tasks under Group B in the development plan. The machine-readable criterion register is [`group-b.json`](../../../specs/hegota/group-b.json); it maps every task to normative clauses, production files and named native tests. `tools/hegota/check_b.py` rejects missing tasks or criteria, missing executable consumers, stale source evidence and skipped/failed required tests. Scope labels alone do not pass this gate.

## Normative baseline

Work starts from merged master `960baab3103e0d34435433be0f8314f127521052`. The EIPs reference is `ade9e9cab7e148bba4cd293f916a4a189cd7a8aa`; the consensus-specs reference remains `d2021d2719235f0df4726a1d330502cb15aa8ea3`. The upstream EIPs master was checked on 2026-11-26 and still matched the local pin. Reference contents were not edited.

The register hashes EIP-8141, 8250, 8272, 8037, 8038, 7843, 7928, 7934, 2718, 7702 and 2929. Existing A locks retain the wider prerequisite source set. The current 8141 text is byte-identical to its implementation revision `dc5d5fa688b7fe87b95a907fcc128b4fde2f4d2f`.

## D01: nonce-only delta remains isolated

[EIP-8250](https://github.com/ethereum/EIPs/blob/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8250.md#transaction-payload) instructs clients to replace the nonce field with `nonce_keys, nonce_seq` and preserve other 8141 fields. Its example still lists three fee fields. Current [EIP-8141](https://github.com/ethereum/EIPs/blob/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8141.md#payload-encoding) has a two-field total fee budget.

The source history explains the likely origin, without establishing a new upstream resolution: 8250's latest edit is [the October 19 single-use pricing change](https://github.com/ethereum/EIPs/commit/1a086d5c67cdcf3a134cc571eb4bb4b9bed6fb02), while 8141 adopted `max_fee` on October 22 in [PR #12414](https://github.com/ethereum/EIPs/pull/12414). That PR's discussion also removes the base-fee-dependent TXPARAM 0x06; no permission to revive the old three-fee interface is inferred. The [8250 discussion](https://ethereum-magicians.org/t/eip-8250-keyed-nonces-for-frame-transactions/28437) confirms that `[]` now selects the legacy nonce and `[0]` is keyed; its older opening summary is not used as the current schema.

`FrameTx8250Delta` implements the literal nonce-only replacement against the current two-fee base as an explicitly isolated research profile. It has its own eight-field codec, signing hash, intrinsic/floor pricing, validation and native execution entry point. It does not implement `TxData`; ordinary transaction decoding rejects it. There is no public fork switch, RPC/pool admission route, block importer route or database migration for it. `ExecuteFrame8250Delta` requires a caller-owned disposable research state. The shared native frame loop is reused rather than duplicated.

This closes HGT-017's state-and-vector scope under the development plan's controlled-profile allowance. It does **not** close D01 or claim a public, interoperable 8141+8250 wire format. Resolving D01 requires a selected, authoritative combined schema and matching valid/invalid vectors. Group C's keyed pool identity/revalidation work remains separate.

The profile verifies all selected keys against actual transaction pre-state, charges access intrinsically, charges first-use state gas at payment approval, writes all selected slots atomically and emits ordinary-storage BAL changes at the caller's transaction index. These protocol accesses do not warm accounts or slots. An empty key set increments the current legacy nonce; keyed consumption leaves it unchanged. TXPARAM 0x0D captures the immutable pre-state legacy nonce even when CREATE changes the current nonce.

Native tests cover sorted/duplicate/overflow keys, zero key, empty set, 16 keys, sequential/single-use/exhausted keys, overlapping/disjoint sets, late-key mismatch, state exhaustion, approval revert/retry, invalid VERIFY rollback, later body/atomic failure, sponsorship without legacy-account creation and CREATE/introspection. Four codec/hash/key-set vectors are calculated independently with Python RLP/Keccak and consumed by the native Go implementation.

## Recent roots use the ordinary contract

[EIP-8272](https://github.com/ethereum/EIPs/blob/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8272.md) uses an ordinary contract and existing VERIFY/introspection instructions. Its [discussion](https://ethereum-magicians.org/t/eip-8272-recent-roots-for-frame-transactions/28621) records the change from an outer field to a leading verifier frame. The old outer-field design is not implemented.

`RecentRootCode` is the exact normative runtime. Source IDs hash 20-byte addresses plus 32-byte salts; entry/storage hashes use separate domains and eight-byte big-endian slots/indices. Native execution supplies the actual header `slotNumber` to SLOTNUM. No timestamp/slot-duration conversion is added. The tuple classifier does not replace execution or grant pool exceptions.

Both system contracts are tested through the exact ordinary keyless deployment transactions in their EIPs, including recovered deployment address, deployed nonce and full runtime bytes. Activation performs no installation or boundary check. The fixture does not repair an absent manager or recent-root account.

Actual EVM tests cover previous/current/future slots; ages 8191 and 8192; malformed, empty, duplicate and excessive tuple encodings; wrong source/root/slot; undeployed/default-code failure; static write rejection; sixteen distinct cold keys including frame-entry access; exact gas and one-gas-short failure; same-slot final-write wins; ring overwrite; delegate storage context; skipped consensus slots and timestamp independence. A native builder/import/fork-switch test removes an orphaned write, excludes its verifier on the new branch and restores the original branch's root.

Public-pool prefix ordering, exact-code trace exceptions, next-header-slot simulation, dependency indexing and eviction are HGT-020–022 in Group C. They are not presented as delivered by ordinary-contract execution in Group B.

## Base execution and persistence

The existing seven-field native implementation is reused for HGT-002/003/010–016/019. All seven independently generated base codec vectors now have direct native decode/encode/hash/signing-hash consumers. Added matrices cover approval scopes, every parameter mapping, result visibility, invalid indices, non-Frame contexts, hidden protocol witnesses and stateless integer/signature rejection. Valid P256 signatures are executed through sender contract code; default code still requires the prescribed SECP256K1 entry. Arbitrary witness bytes never constitute protocol authentication.

Two boundary corrections are included: nonce overflow at payment approval exceptionally halts, and failed canonical frame receipts cannot carry surviving state gas or logs. Canonical stateless validation remains before sender-state reads.

The shared builder/import path is re-executed from independent state, comparing state/BAL and exact settlement. Corrupt receipt/state/BAL/gas/bloom commitments must fail import without changing canonical state. BAL item and encoded block byte bounds have explicit consumers. Existing prerequisite system-call tests remain part of the criterion register.

Public transaction and receipt encodings do not change. The type byte and existing payload schema distinguish canonical and retained historical formats; the isolated keyed codec is excluded from public readers. Existing FileDB, cold receipt readback, replay, fork switching and restart tests retain coverage. No public database migration is required; rolling back this branch leaves the previously supported public encodings readable. No claim is made that a hypothetical externally activated keyed schema can be downgraded without a migration decision.

## Qualification

The final verification record is `evidence/hegota-b/verification.json`, with exact native test identities and current artifact hashes. A evidence is requalified because its runtime source fingerprints cover the shared executor and VM. Prior A run URLs remain historical evidence, not new-head results.

The independent historical Geth differential baseline covers its declared 12 historical cases; it is not an independent implementation oracle for these Frames profiles. Base Python goldens, the four isolated keyed goldens, normative recent-root vectors and native execution tests have separate roles. Public combined Hegota release, Group C pool behavior, official devnet interoperability and independent-client Frames equivalence are not inferred from these results.

The [second audit](SECOND_AUDIT.md) records four subsequently reproduced dispatch/BAL defects, their normative corrections and the additional regression/independent-call-body qualification. Earlier completion labels are not reused as proof that those defects were absent.
