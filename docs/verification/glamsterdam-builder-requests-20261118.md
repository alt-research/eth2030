# EIP-8282 builder execution requests

Branch `feat/glamsterdam-builder-requests`, independently based on current master
`38ea9259`. This implements the settled EL request pipeline for GL-034; it does
not qualify the complete Glamsterdam program.

## Normative inputs

Read [EIP-8282](https://github.com/ethereum/EIPs/blob/3db38a3725105cb67b8804ac3e147967113bc3ca/EIPS/eip-8282.md)
in full. Canonical runtime and constructor assets are copied from the linked
[sys-asm revision](https://github.com/ethereum/sys-asm/tree/83f9801245ff56878a450b5625801101b9a225a1).
The selected [Gloas transition](https://github.com/ethereum/consensus-specs/blob/2ea9d96a420927eb1a35d7c9399c8b3691c69f6a/specs/gloas/beacon-chain.md)
defines the actual CL credential, signing-domain and exit gates. Its explicit
builder credential check is retained despite the EIP's shorter deprecation
summary. No pending authentication or gas alternative is selected here.

## Implementation and deployment boundary

Amsterdam finalization drains mandatory deposit/exit contracts into ordered
request types 3/4, after the Prague queues. Missing code, a failed system call,
malformed record width or an excessive response count invalidates the block.
The dedicated 30-million system budget does not debit transaction/block gas.
Existing snapshot finalization reverts earlier queue changes and withdrawals on
failure; builders and native importers use this same transition.

The contracts MUST already exist when Amsterdam activates. There is no fork-time
code injection, fallback empty queue or new runtime switch. Explicit test and
owned devnet allocations deploy them before genesis state roots are computed,
including nonce 1 and the exit constructor's inhibitor. Existing profile funds,
code and storage allocations are preserved. Supplied production genesis files
remain authoritative: operators must arrange deployment before activation.
The internal fresh-state block fixture now documents its matching genesis
requirement; ordinary Prague-only regressions explicitly select Prague.

The actual bytecode handles minimum principal, fee/overpayment, big-endian input
and anonymous logs, little-endian SSZ output, queue state and the 64/16 drain
caps. The native EL does not verify BLS proofs. The public request codecs expose
the exact 184/68-byte records. Constructor execution in the native VM returns the
canonical runtime and inhibitor; runtime constants independently match unmodified
Geth v1.17.10.

## Verification and real consumers

Native contract tests cover invalid lengths, missing fees, sub-minimum principal,
value-bearing fee getters, exact count-dependent integer-series fees, read-only
fee queries, pre-fork deposits surviving activation, queue excess/count/index
updates, authorization source bytes, output endianness, capped FIFO tails,
empty queues, late-queue failure rollback and malformed/oversized system output.
Requests commitments are independently checked with Geth.

The actual node-backed HTTP Engine regression signs ordinary transactions to
both contracts, compares published type-3/type-4 bytes with a frozen real-BLS
fixture, rejects an altered request on a separate native receiver without head
adoption, then imports the valid payload. Every owned Gloas profile is checked
for the exact runtime, nonce and inhibitor allocation.

The separate Lighthouse consumer review decodes these same SSZ bytes and uses
actual real-crypto state-processing functions: correct builder-domain PoP
registers immediately, validator-domain PoP is ignored, top-ups preserve the
registered withdrawal address, inactive/unauthorized/pending-withdrawal/pending-
payment exits are ignored, and the valid exit gets the exact withdrawal epoch.
Validator registry/balances/pending deposits remain unchanged. This is an
independent component consumer of the exact Engine bytes, not a live multi-node
CL/EL network result.

Reproduce the synthetic public fixture from `pkg/`:

```sh
go run -tags blst ../tools/fixtures/eip8282/generate.go
```

The regenerated JSON matches the committed fixture byte-for-byte. No production
secret is used. The generator documents its fixed public IKM.

All 210 tested Go packages pass. Final affected tests, tagged Core/EIP tests,
25 targeted race-tested packages, affected vet and `go build ./...` pass. Logs:
`/tmp/glamsterdam-builder-requests-{all-4,final-tests,tagged,race,restored}.log`.
Three controlled mutations fail the intended regressions: dropping mandatory
contract enforcement, writing request SSZ amounts as big-endian, and removing
queue rollback. All sources are restored and affected tests pass again. The
byte-order helper is cross-checked against actual contract output; the HTTP
pipeline does not call that helper and is not its mutation oracle.

The final gas model, complete BAL qualification, deployment through an actual
factory/network, live builder registration/exit across a full CL/EL devnet,
restart/cold-sync and independent full-client release gates remain open.
