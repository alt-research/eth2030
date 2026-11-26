# Canonical paymaster implementation reconciliation

Reviewed 2026-11-26 (Asia/Shanghai). This record replaces the inference that an
unmerged runtime proposal prevents implementing and qualifying its exact bytes.
Upstream standardization remains distinct from a completed selected-profile
implementation. Reference submodules remain read-only.

## Authority and complete discussion history

The current [EIP-8141](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-8141.md)
has blob identity `9bc38a368881fa52e1a1a9970274e6511874d13f`, identical to our
pinned normative source. Its canonical-paymaster section still describes one
secp256k1 signer and ecRecover. The merged Solidity asset has a 12-hour delay,
an arbitrary withdrawal recipient, no rotation, a calldata signature and
non-compilable inline verbatim bindings. It supplies no reproducible runtime
recognition anchor compatible with the current signature-witness rules.

[EIPs #12012](https://github.com/ethereum/EIPs/pull/12012) was authored by
EIP-8141 author svlachakis, then closed **without merge**. Its author's final
[comment](https://github.com/ethereum/EIPs/pull/12012#issuecomment-5136771510)
explicitly moves the work to [#12041](https://github.com/ethereum/EIPs/pull/12041).
The latter remains open, unmerged and conflicting with master at exact head
`4a98fcefa5c6f02a25289bf55b9d92902d02b4c6`. All sixteen commits were reviewed:
the seven inherited design commits establish code-hash recognition, protocol
signature authorization, scheme agility, caller/signature administration,
instance initialization and rotation invalidation; the subsequent commits add
the assembled runtime and annotation, revise gas prose, remove the superseded
Solidity asset, and merge the October 7 base. There is one empty approval from
AnkushinDaniil, no inline review threads, and the bot still requests an author
review. No newer author decision was found.

The entire current [Frame discussion](https://ethereum-magicians.org/t/frame-transaction/27617)
was fetched (181 available posts, highest 187). Posts 179, 182, 184 and 186
discuss ecRecover's EIP-8151 dependencies; post 182 says Nethermind traces every
PAY because it does not pin a canonical runtime. Those posts concern an
ecRecover-based implementation. They do not add an ecRecover call to #12041's
SIGPARAM-based code. The current EIP-8151 public-pool rules remain applicable
to actual paid recovery reads, including those made by other PAY contracts.

## Executable decision

Implement the exact #12041 runtime as an explicitly selected proposal profile:

```text
--experimental-frame-paymaster-12041
runtime: 355 bytes
keccak256: da42f0d11838c4c0c3129b8b8e93e9718127ad6b315e517e1088125707c4d45c
source: ethereum/EIPs@4a98fcefa5c6f02a25289bf55b9d92902d02b4c6
asset: assets/eip-8141/canonical-paymaster.md
```

The vendored CC0 annotated assembly is the reproducible source. A native test
assembles its labels/instructions independently, then compares all bytes,
length and the upstream hash. Recognition requires exact raw runtime bytes;
mutations, appended bytes, delegation indicators and the local 459-byte v1
receive no recognition under this profile. Both versions cannot be selected
together. Selection is local pool policy, off by default. No mainnet/fork
schedule is invented, and no code or storage is installed at activation.

The local initialization program writes slot 0 and returns the exact runtime.
Its bytes are not an upstream constructor identity. Zero-signer instances may
be deployed but authorize nothing, following the proposal.

## Clause-to-implementation reconciliation

| Requirement | Implemented behavior and native consumers |
|---|---|
| Payment authority | Empty PAY calldata; signature index 1 must have a protocol-verified non-ARBITRARY scheme, resolved signer equal to slot 0 and empty msg. SECP256K1 and P256 execute natively. Missing, wrong, arbitrary, explicit-digest and corrupted authorizations reject without live-state effects. |
| Canonical hash | Authorization binds chain, sender, nonce, fee budget, PAY target and all frames. The merged EIP's rejection of an explicit zero digest makes the SIGPARAM zero sentinel unambiguous; no opcode semantics are changed. |
| Versioned storage | Slots 0/1/2/3 are signer/withdrawal amount/unlock time/new signer. All four slots, raw runtime and payer balance are dependencies. The 12-hour v1 layout is kept separate. |
| Treasury | Value with empty calldata deposits. Administrative calls with value revert. Only payment approval and matured withdrawal remove ETH. Withdrawals go to the current signer; callers cannot choose another recipient. |
| Administrative authority | Direct caller equal to signer, or a protocol-verified index-1 canonical signature. Real ordinary typed transactions exercise the caller route and non-Frame rejection; Frame transactions exercise SECP256K1 and P256 signature routes. |
| Single pending action | Nonzero withdrawal or nonzero rotation; shared 86,400-second clock. A second initiation rejects. Authorized cancellation clears all pending slots. |
| Finalization | Anyone can finalize exactly at maturity, not one second earlier; absent actions reject. Withdrawal clears pending state before CALL. Insufficient funds and failed receivers restore funds/action state. Native reentrancy cannot pay twice. Rotation changes slot 0 and clears pending slots without moving ETH. |
| Pool availability | Sum of signed max_fee plus raw slot-1 withdrawal, subtracted once per instance. Maturity alone never releases the withdrawal. Replacement transfers exposure atomically; removal, inclusion, reset and reorg release or rebuild indexes. |
| Candidate state | Build validation repeats the selected version's withdrawal-adjusted availability against candidate state. It does not reserve unrelated, unselected pool transactions. A native counterexample reproduces the earlier missing check. |
| Public validation work | Existing shared execution/state validation caps apply. This profile additionally selects #12041's optional 15,000 PAY execution limit. Runtime recognition is cached per immutable VERIFY frame/profile and follows frame snapshots: disabled selection reads no code, selected recognition reads raw code at most once. Higher-budget consensus-valid transactions remain valid privately; only local public admission is limited. |
| EIP-8151 | The exact proposed runtime performs no ecRecover. Its protocol signature creates neither a recovered-code-class dependency nor an artificial signer-share charge. Actual ecRecover contracts retain the previously implemented class/quota rules. |
| Native runtime paths | Base/keyed and plain/blob cases submit over HTTP, propagate to a second node, simulate authenticated execution, build from the actual public pool, independently import/select, invoke withdrawal administration, reorg orphan-only withdrawal state and receipts, reselect and restart from disk. |
| Sending tool | `send-frame --paymaster <address> --paymaster-12041 --sponsor-key <key>` signs the empty-data interface and checks the selected runtime. Its private-key interface remains a secp256k1 convenience; native transaction admission also supports P256. |

## Remaining release boundary

There is no remaining local bytecode implementation gap for the pinned proposal.
The upstream proposal has **not** become the merged canonical implementation:
the merged text and proposed authorization/layout differ, and no finalized
fork schedule pins this hash. `public_complete: false` and the draft PR remain
honest release markers until that upstream decision is made. The proposal
profile can already be deployed, executed and qualified explicitly.

## Local qualification

Implementation revision: `4ac957e9` (later edits qualify documentation/evidence).
All 240 full-suite and 57 focused-race packages completed successfully:
24,005/9,588 passing test/subtest events and zero failures. Native/wrapper
builds, full vet, the independent 12/12 historical baseline and 91 Python
regressions in each normal/optimized mode pass. The A/B/C required consumers
are 62/79/92 with zero required skips; 124/74 optional full/race skips are
separately inventoried.

Four explicit feature mutations fail their native consumers while preserving
positive controls: disable exact proposal recognition, read the wrong
withdrawal slot, omit the pending-rotation dependency, and omit candidate
withdrawal availability. These are current-source mutations, not claims that
the unmodified predecessor compiles against new APIs. Their log and overlay
hashes are retained in C evidence alongside the preceding audit controls.
An additional exact `b750e897` predecessor overlay reproduces 100 raw-code
copies for 100 opcodes with recognition disabled; the current bounded-work
consumer and its frame/profile/rollback cases pass. Its positive verifier
control passes against both versions. This fifth control is a real prior-source
counterexample, separate from the four explicit feature mutations.
Exact pushed-head hosted results belong in the English PR description/checks.
