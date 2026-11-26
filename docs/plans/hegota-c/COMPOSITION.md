# Frame composition decision and remaining canonical-runtime gate

Decision reviewed 2026-11-26 (Asia/Shanghai). This addendum supersedes the earlier
D01-open conclusion for the selected native 8141/8250 composition. It does not
select every Hegota proposal or claim official CL/devnet interoperability.

## Selected encoding

```text
0x06 || rlp([chain_id, nonce_keys, nonce_seq, sender, frames, signatures,
             [max_fee, max_priority_fee_per_gas], blob_versioned_hashes])
```

The explicit EIP-8250 instruction changes only the nonce position and inherits
all other EIP-8141 rules. Its three-fee worked example is stale after the later
base fee change; it is not a second independent fee model to implement.

The decision follows the actual merged history, in order:

1. [EIPs #11968](https://github.com/ethereum/EIPs/pull/11968), authored by the
   EIP-8250 author and merged July 29, expressly makes the nonce changes a delta
   with the base signing procedure and all other fields inherited.
2. [EIPs #12316](https://github.com/ethereum/EIPs/pull/12316), merged September 11
   with the author's approval, nests the then-current fee fields. It predates
   the base fee model replacement.
3. [EIPs #12414](https://github.com/ethereum/EIPs/pull/12414), merged October 22,
   replaces the base fees with the total max_fee budget and priority-per-gas.
4. [EELS #3892](https://github.com/ethereum/execution-specs/pull/3892), followed by
   [#3893](https://github.com/ethereum/execution-specs/pull/3893) and
   [#3897](https://github.com/ethereum/execution-specs/pull/3897), rebases and
   adapts the keyed implementation/tests. The current actual source, rather
   than the PR merge label alone, corroborates the two-field composition:
   [frame_transaction.py at 6013c4e](https://github.com/ethereum/execution-specs/blob/6013c4eaed5c9cc1cbb8428541972d6976a8cabd/src/ethereum/forks/bogota/transactions/frame_transaction.py#L386).

EIP text remains authoritative. EELS's U64 chain ID is not imported: the native
implementation retains the EIP's uint256 chain ID. Upstream fixture success is
upstream evidence, not a claim that ETH2030 ran the complete EELS fixture corpus.
Independent existing wire/hash and nonce oracle expectations are preserved.

## Activation and consumers

`eip8250Time` in genesis opts into the eight-field format at or after Bogotá.
Nil leaves the seven-field base profile selected. Historical readers understand
both, but public admission and consensus execution require exactly the schema
selected by the actual timestamp. Rollback revalidates and evicts the opposite
schema. This does not schedule a mainnet activation.

The NONCE_MANAGER must already be deployed ordinarily or allocated in genesis
with nonce 1 and runtime 0x60006000fd. Activation neither installs it nor performs
a special state/code check at the boundary. Its ordinary deployment precondition
is the chain operator's responsibility under the EIP.

Keyed replay sequences do not order or replace account-nonce transactions.
Public replacement requires identical sender, ordered keys and sequence, with
one pending Frame per sender. Each selected slot is a dependency; TXPARAM(0x0D)
adds the optional pre-state legacy nonce dependency. The shared Frame executor,
fees, receipts, sidecars, RPC simulation, P2P, builder/import and disk replay
consume the same signed envelope.

## Conditional EIP-8151

[EIPs #12653](https://github.com/ethereum/EIPs/pull/12653), merged November 16,
already adds Public Mempool rules to EIP-8151 itself. The still-open mirrored
8141 summary is not required authority. `eip8151Time` explicitly enables the
conditional rule after its EIP-8038 prerequisite; it remains off by default.

Paid ecRecover raw-code reads record permitted/rejected class dependencies,
separately from helper code hashes, including both directions across reorgs.
Canonical signing-hash recoveries have a reserved local share (default 64);
other digests have a smaller separate share (default 8). PAY recoveries are
accounted separately and once per distinct paymaster within each share. These
numbers are configurable local pool limits, not invented consensus constants.
The base 3000 plus cold 3100/warm 200 gas charges precede code access. Unpaid
checks add no read/warming. Paid reads survive rejection and outer revert;
warming follows ordinary snapshot rollback. Delegation is classified raw and
never followed by this check.

## Canonical paymaster proposal implemented; upstream standardization pending

The merged CanonicalPaymaster.sol has no frozen reproducible runtime/compiler
settings; its calldata authorization and nonstandard inline verbatim bindings
also prevent treating the local template as an identical implementation.
[EIPs #12041](https://github.com/ethereum/EIPs/pull/12041) at
4a98fcefa5c6f02a25289bf55b9d92902d02b4c6 remains open/unmerged. Its proposed
355-byte runtime changes witness and administration semantics. That exact
runtime is now implemented and independently reproduced from its annotated
assembly, selected with `--experimental-frame-paymaster-12041`. It has separate
SIGPARAM authorization, slots 0 through 3, 24-hour withdrawal/rotation, cancellation,
raw slot-1 reservation, cache invalidation and candidate-state availability.
The 459-byte local v1 remains a distinct, mutually exclusive profile. Neither
selection claims a finalized upstream standard or mainnet activation.

The current implementer discussion independently confirms this boundary:
[EIP-8141 post 182](https://ethereum-magicians.org/t/eip-8141-frame-transaction/27617/182)
reports tracing every PAY frame without a pinned canonical paymaster. The
EELS mempool fixture work is still draft and lists canonical paymaster work
as outstanding. This does not prevent implementing the pinned proposal's
determinate behavior. Native base/keyed plain/blob consumers now qualify that
explicit version through real transport, simulation, construction, independent
import, administration, reorg and disk restart. See the complete
[source reconciliation](../../reviews/pr-170/CANONICAL_PAYMASTER.md).
The single remaining gate is upstream adoption/fork pinning, rather than a
missing local runtime implementation. `public_complete: false` remains explicit.
