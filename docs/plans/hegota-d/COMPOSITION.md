# Group D composition and compatibility decision

Reviewed 2026-11-27 (Asia/Shanghai), from merged PR #170 at
`81e63f586642a1406d56db9fa7cbf00941feb7bf`. The complete acceptance register
is in [PLAN.md](PLAN.md); clause/test/source identities are in
[the machine-readable register](../../../specs/hegota/group-d.json).

## Normative precedence and discussion review

The selected direct-verification draft is [EIP-8504/#12672](https://github.com/ethereum/EIPs/pull/12672)
at `96da3c17047ad31608c40bfe75049db78dc4417d`. The November discussion now
states that wrappers, submissions, block witnesses, header/Engine/CL fields and
gas are specified. Message assignments and provisional constants remain open.
The author's latest PR comment treats #12714 as an alternative requiring an ACD
choice. Its signature-list format is not blended into this implementation.
[Discussion](https://ethereum-magicians.org/t/eip-8504-proof-frames-for-leansphincs-signatures/30448).

EIP-8288 master blob `611f58a8668bbffb798c3b3b073cfaa275265f34` clarifies that
aggregation checks extend ordinary block validity. Its aggregate verification
key and dependency hash function remain TBD. Direct witness verification is the
selected profile; recursive STARK aggregation is unavailable and never simulated
by treating a successful no-code frame as authentication.
[Discussion](https://ethereum-magicians.org/t/eip-8288-frame-type-for-pq-sig-and-stark-aggregation/28723).

The SETCODEFROM discussion changed direction over time: early replies proposed
banning initcode adoption, while September feedback identified immediate adoption
as the way to separate loaded initcode from subsequent introspection/reentry.
The current EIP text governs: adoption is immediate, execution continues, and
successful creation ignores returned bytes when code was adopted. Same-transaction
sources are rejected. Historical advice about merely retaining deleted-source
code does not override that current restriction.
[Discussion](https://ethereum-magicians.org/t/eip-8298-setcodefrom-code-reuse-instruction/28779).

The ASSERT discussion includes an older POST_TX/TXDIFF prototype with different
opcodes and invalidation behavior. Current EIP-7906 governs: ASSERT mode 3,
TXTRACE 0xb6, EVENTDATACOPY 0xb7, static trailing suffix, body rollback with
prefix payment/nonce retained. Events alone do not prove storage outcomes;
assertion reference values must be signed or derived from prestate.
[Discussion](https://ethereum-magicians.org/t/eip-7906-transaction-assertions-via-state-diff-opcode/23130).

EIP-8151 is reused from Group C, including raw-code classification, exact paid
access boundaries and signer-class caching. New code-adoption/assertion journals
forward its special reader, preserving class dependencies rather than replacing
them with unnecessary helper-code dependencies.

## Explicit activation and native transport

`EIP8504Time`, `EIP8298Time` and `EIP7906Time` are nil by default, validated and
persisted with ordinary compatibility checks. 8298 requires Amsterdam's current
8037/8038 rules; 8504/7906 require the canonical Frame fork. Keyed nonce activation
continues to use the qualified C profile. Existing no-argument validation APIs
remain closed to optional modes, while consensus/pool/simulation use explicit
extensions at the actual block timestamp. No transaction is mutated to install
head-specific rules.

The pinned direct scheme is 0x10, PK/message 32 bytes, witness 7,740 bytes,
515 Keccak-f permutations, 3,000 execution gas per declared triple and 495,360
witness gas per distinct transaction dependency. The 256-triple/frame and
16-distinct-dependency/public-transaction limits are pinned provisional values.
Duplicates still pay verification gas on every declaration. Witness gas enters
both intrinsic cost and the calldata floor, including keyed signing hashes.

No official eth or Engine version is assigned by EIP-8504. The native `hgtp/1`
capability has an independent namespace and four local messages: chain/profile
status, wrapper announcements, requests and responses. Its pinned reference
message bodies come from ethrex `897a4fc9ecd869a8b86e99167fef46404ab1c50b`,
`proof_wrappers.rs` blob `ee651f5c906b9e1a20704435b0b6800b3aa02ac2`.
It is advertised only with an explicit 8504 schedule. Ordinary eth transaction
messages reject dependencies. Wrappers are served only from admitted lookup
entries, with one per transaction and atomic byte accounting/removal.

Local policies are 64 MiB of dependency wrappers (also charged against the
ordinary total pool budget), 2 MiB per wrapper/requested response budget,
4 MiB maximum response, 4,096 announcements/queued entries, 256 hashes/request,
16 outstanding requests, 16,384 remembered transaction requests per connection,
15-second request expiry and a 4,096-undelivered cutoff. A bounded queue drains
announcements beyond the first request; limits do not silently truncate a valid
batch. These are local policies, not assigned EIP constants.

The CL codec appends an opaque progressive witness list to Gloas's 19-field
payload, even when empty, following ethlambda
`7aef8bd75316078010193684715359b26b5c2be6`, blob
`4fa26a62b25f7165a8638874da7242b594339861`. Independent remerkleable
`2f0baeef0082d4278acaef7d822deb7009d7db7e` vectors bind bytes and active-field
roots. EIP-7807's separate execution-block identity is not the Gloas payload;
its unspecified combination is rejected rather than dropping witnesses or
inventing an aggregate/header schema. No external Lighthouse modification or
official multi-client devnet qualification is claimed.

The canonical-bytes-only inclusion-list API cannot authenticate an omitted
proof transaction: the block witness list must equal the included dependency
union, so adding an omitted dependency's witness would itself invalidate the
block. Omitted dependency candidates therefore produce an explicit unavailable
error before trial execution. Included candidates work normally. Local mempool
availability never determines a consensus IL verdict. A witness-bearing omitted
IL composition needs an upstream specification before this combination opens.

## Migration and application compatibility

Independent EELS `1d07f742873458b2a77d11af100f57a8d14180b8` signing and
verification algorithms generate canonical base/keyed signing-hash witnesses.
The real PQ wallet checks proof mode, exact declaration size, scheme, zero
message placeholder and its stored public key before approving. The actual HTTP
simulation/admission/Engine/restart consumer uses that wallet. Changing the
replay identity fails witness verification. Explicit-message sharing remains a
separate block-union vector; it is not treated as automatic transaction binding.

Native migration executes 7702 delegated code in the authority account, adopts
regular wallet code, emits a hash-only BAL change and retains code availability
after source replacement/restart. The old key can neither originate ordinary
transactions nor redelegate; Frame-based wallet control remains available.
Already pending/queued retired origins are evicted on head revalidation. Proxy
helpers retain code-hash dependencies and a local eight-pending-transaction share,
with replacement/removal/reorg tests. PUSH and EIP-8024 immediate bytes are
excluded from mutable-helper scans.

Native recovery/authorization controls show that permit-style, two-of-two
multisig and fault-proof authorization kernels continue recovering a migrated
address without 8151, and reject it with 8151. These are actual EVM executions,
not mocked recovery results; they do not certify complete deployed applications.
The 8151 discussion identifies asynchronous L1/L2 fault-proof disagreement and
immutable permit applications as compatibility hazards. Retiring an address on
L1 does not retire it on an independently stateful L2. Rollup/application owners
must qualify historical precompile acceleration and authorization migration
before public activation. Disabling the restriction to get a passing result is
not an acceptable mitigation.
[Discussion](https://ethereum-magicians.org/t/eip-8151-account-code-restricted-ecrecover/27690).

ASSERT failure preserves the exact PR-12041 paymaster's prefix authorization,
ETH payment and sender nonce; all body token transfers/logs/state gas roll back.
The payer still pays consumed execution gas. False-returning token code is not
a protocol revert. Independently authorized retries and prior atomic failures
are exercised. There is no automatic token compensation in either EIP, and no
compensation mechanism is invented. Current C canonical-paymaster adoption gates
remain unchanged.

## Release gates

Native profile implementation does not schedule a public fork. Upstream scheme
parameters, official transport/Engine assignments, CL field/limit finalization,
8288 aggregation details, 7807/omitted-IL composition and ecosystem compatibility remain
explicit release gates. The general Hegota release profile stays inactive.
Groups E–J and official cross-client/devnet/release qualification are separate.
