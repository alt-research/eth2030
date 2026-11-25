# ssz

Simple Serialize (SSZ) encoding, decoding, and Merkleization for the Ethereum consensus layer.

## Overview

The `ssz` package implements the SSZ specification used by the Ethereum consensus layer for deterministic binary serialization and Merkle hash tree computation. SSZ provides a canonical encoding for all consensus types — beacon blocks, attestations, validators, deposits — and is the serialization format required for fork-choice, state storage, and light client proofs.

The package covers the full SSZ encoding surface: basic types (bool, uint8–uint256), fixed and variable-length containers, vectors, lists, bitvectors, and bitlists. The Merkleization functions compute SHA-256-based hash tree roots compatible with the consensus specification, including the `mix_in_length` operation for lists.

Beyond the base spec, the package implements two consensus-layer EIPs. EIP-7916 defines `ProgressiveList`, a Merkle tree structure that grows progressively with list size rather than padding to the next power of two; this enables efficient incremental appends. EIP-7495 defines `ProgressiveContainer(active_fields)`: an immutable type layout with up to 256 positions. Inactive positions are zero Merkle leaves and consume no wire bytes; active fields encode exactly as an ordinary Container.

## Table of Contents

- [Functionality](#functionality)
- [Usage](#usage)
- [Documentation References](#documentation-references)

## Functionality

### Core Interfaces

```go
type Marshaler interface {
    MarshalSSZ() ([]byte, error)
    SizeSSZ() int
}

type Unmarshaler interface {
    UnmarshalSSZ([]byte) error
}

type HashRoot interface {
    HashTreeRoot() ([32]byte, error)
}
```

### Basic Type Encoding

| Function | Output |
|----------|--------|
| `MarshalBool(v bool) []byte` | 1 byte: `0x01` or `0x00` |
| `MarshalUint8(v uint8) []byte` | 1 byte |
| `MarshalUint16(v uint16) []byte` | 2 bytes little-endian |
| `MarshalUint32(v uint32) []byte` | 4 bytes little-endian |
| `MarshalUint64(v uint64) []byte` | 8 bytes little-endian |
| `MarshalUint128(lo, hi uint64) []byte` | 16 bytes little-endian (two 64-bit limbs) |
| `MarshalUint256(limbs [4]uint64) []byte` | 32 bytes little-endian (four 64-bit limbs) |

### Composite Type Encoding

- `MarshalVector(elements [][]byte) []byte` — concatenate fixed-size element encodings.
- `MarshalFixedContainer(fields [][]byte) []byte` — alias for `MarshalVector`; use for containers with all fixed-size fields.
- `MarshalList(elements [][]byte) []byte` — same encoding as vector; length is tracked separately in the parent container.
- `MarshalVariableContainer(fixedParts, variableParts [][]byte, variableIndices []int) []byte` — encode a container with mixed fixed and variable-size fields. Fixed-size field slots occupied by variable fields get 4-byte offsets (`BytesPerLengthOffset = 4`); variable payloads are appended after.

### Bitfield Encoding

- `MarshalBitvector(bits []bool) []byte` — pack exactly `len(bits)` bits into bytes, LSB first.
- `MarshalBitlist(bits []bool) []byte` — bitlist encoding with sentinel 1-bit to mark length boundary.
- `MarshalByteVector(data []byte) []byte` — fixed-length byte vector copy.
- `MarshalByteList(data []byte) []byte` — variable-length byte list copy.

### Decoding

- `UnmarshalUint64(b []byte) uint64` — decode 8-byte little-endian.
- `UnmarshalBool(b []byte) (bool, error)` — validates `0x00` or `0x01`.
- Container and list decoding follow the offset-pointer pattern for variable fields.

### Merkleization

- `Merkleize(chunks [][32]byte, limit int) [32]byte` — standard SSZ Merkle root: pad chunks to the next power of two of `max(len(chunks), limit)`, then iteratively hash pairs up to the root.
- `MixInLength(root [32]byte, length uint64) [32]byte` — mix in the list length: `SHA256(root || uint64_LE(length) || zeros[24])`.
- `Pack(data []byte) [][32]byte` — pack a serialized byte sequence into 32-byte chunks (zero-padded last chunk).
- `HashChunk(data [32]byte) [32]byte` — SHA-256 hash of a single 32-byte chunk.

Cache: `MerkleCache` provides an LRU-backed Merkle subtree cache for repeated `HashTreeRoot` computations on large validator sets.

### EIP-7916: ProgressiveList

`ProgressiveList` is an SSZ list type whose Merkle tree grows progressively:

```
depth 1: 1-leaf subtree
depth 2: 4-leaf subtree (4× previous)
depth 3: 16-leaf subtree
...
```

```go
pl := ssz.NewProgressiveList(elementRoots)
pl.Append(chunk)
root := pl.HashTreeRoot()                     // mix_in_length(merkleize_progressive(...), len)
proof, gindex, err := pl.GenerateProof(idx)  // Merkle inclusion proof
```

Convenience functions for typed lists:
- `HashTreeRootProgressiveList(elementRoots [][32]byte) [32]byte`
- `HashTreeRootProgressiveBasicList(serialized []byte, count int) [32]byte`
- `HashTreeRootProgressiveBitlist(bits []bool) [32]byte`

### EIP-7495: ProgressiveContainer

```go
schema, err := ssz.NewProgressiveContainerSchema(
    []bool{true, false, true}, // Stable field positions 0 and 2.
    []int{2, 1},              // uint16 and uint8; 0 denotes a variable field.
)
if err != nil { return err }
wire, err := schema.MarshalSSZ([][]byte{ssz.MarshalUint16(42), ssz.MarshalUint8(255)})
// wire is 2a00ff: neither an inactive value nor active_fields is transmitted.
fields, err := schema.UnmarshalSSZ(wire)
fieldRoots := [][32]byte{ssz.HashTreeRootUint16(42), ssz.HashTreeRootUint8(255)}
root, err := schema.HashTreeRoot(fieldRoots)
proof, gindex, err := schema.GenerateProof(fieldRoots, 2)
valid := schema.VerifyProof(root, fieldRoots[1], 2, proof)
```

A schema owns its layout and field sizes. Empty layouts, widths above 256,
trailing inactive positions and declared/active count mismatches are errors.
There is no requirement for a field named `more` at position 255. Field values,
defaults and JSON use their ordinary typed Container codecs; the schema composes
already encoded active-field values and their roots, without defining another
SSZ value/type system. Variable-field offsets are uint32 and must be canonical.

`HashTreeRootProgressiveContainerChecked` accepts position-indexed root slots,
requires inactive slots to be zero and rejects undeclared mask bits. Existing
execution SSZ consumers use this checked form. The unchecked root primitive is
available only for callers that have already validated a schema.

`StableContainer` and `Profile` are deprecated compatibility root utilities.
Their historical capacity/optional-field behavior is not an EIP-7495 type
system or codec. New code should use `ProgressiveContainerSchema`.

Independent `ethereum/remerkleable` fixtures cover bytes, roots, complete
branches, growth boundaries and illegal layouts. A separately decoded captured
384-validator Gloas state verifies byte identity, all 46 field roots/proofs and
original CL current/next/finalized branches. See `testdata/eip7495/` and the
feature review for reproduction commands and the reference library's extra
256-position naming restriction.

### Union Codec

`UnionCodec` handles SSZ Union types (typed variant containers with a selector byte).

### Progressive Encoder

`ProgressiveEncoder` provides streaming SSZ encoding for large containers, writing fixed sections and accumulating offsets before appending variable sections.

## Usage

```go
// Encode a beacon block slot.
slotBytes := ssz.MarshalUint64(block.Slot)

// Build a variable container (e.g. BeaconBlock body).
fixedParts := [][]byte{
    ssz.MarshalUint64(body.RANDAOReveal),
    nil, // variable field: attestations
}
variableParts := [][]byte{attestationsSSZ}
encoded := ssz.MarshalVariableContainer(fixedParts, variableParts, []int{1})

// Compute hash tree root for a list of validators.
chunks := make([][32]byte, len(validators))
for i, v := range validators {
    chunks[i], _ = v.HashTreeRoot()
}
root := ssz.Merkleize(chunks, maxValidators)
listRoot := ssz.MixInLength(root, uint64(len(validators)))

// ProgressiveList for log index (EIP-7745).
pl := ssz.NewProgressiveListEmpty()
for _, logRoot := range logRoots {
    pl.Append(logRoot)
}
root = pl.HashTreeRoot()
```

## Documentation References

- [Roadmap](../../docs/ROADMAP.md)
- [SSZ Specification](https://github.com/ethereum/consensus-specs/blob/dev/ssz/simple-serialize.md)
- EIP-7916: SSZ ProgressiveList
- EIP-7495: SSZ ProgressiveContainer
- EIP-6404: SSZ transactions (uses this package for serialization)
- EIP-7807: SSZ blocks
