// Package leansphincs implements the EIP-8504 pinned leanSPHINCS parameter set.
// It is SLH-DSA with Keccak-256, WOTS+C, and hypertree heights 8, 8, 10;
// it is not interchangeable with the SHA2 SPHINCS+ implementation in pqc.
// Normative source: ethereum/EthLib c4474ed422d83b6b735ef802ee43960abe1b788f.
// Independent reference: leanEthereum/leanVM 92a97c919b2f2d6fe09853090b6b23f781ecec07.
package leansphincs

import (
	"crypto/subtle"
	"encoding/binary"

	"github.com/eth2030/eth2030/crypto"
)

const (
	PublicKeySize         = 32
	MessageSize           = 32
	SignatureSize         = 7740
	MaxVerifyPermutations = 515
)

type digest [16]byte
type address [32]byte
type hashing struct{ permutations uint64 }

func (h *hashing) hash(parts ...[]byte) []byte {
	n := 0
	for _, p := range parts {
		n += len(p)
	}
	h.permutations += uint64(n/136 + 1)
	return crypto.Keccak256(parts...)
}
func (h *hashing) truncated(parts ...[]byte) (d digest) { copy(d[:], h.hash(parts...)); return }
func (a address) word(at int, v uint32) address         { binary.BigEndian.PutUint32(a[at:at+4], v); return a }
func (a address) kind(v uint32) address                 { clear(a[20:]); return a.word(16, v) }
func treeAddress(layer int, tree uint64) (a address) {
	a = a.word(0, uint32(layer))
	binary.BigEndian.PutUint64(a[8:16], tree)
	return
}
func keyAddress(layer int, index uint64, kind uint32) (address, uint32) {
	heights := [3]uint{8, 8, 10}
	shift := uint(0)
	for i := 0; i < layer; i++ {
		shift += heights[i]
	}
	i := index >> shift
	leaf := uint32(i & ((1 << heights[layer]) - 1))
	return treeAddress(layer, i>>heights[layer]).kind(kind).word(20, leaf), leaf
}
func (h *hashing) th(seed []byte, a address, parts ...[]byte) digest {
	all := make([][]byte, 0, len(parts)+2)
	all = append(all, seed, a[:])
	all = append(all, parts...)
	return h.truncated(all...)
}
func (h *hashing) fold(seed []byte, a address, value digest, path []byte, leaf uint32, base int) digest {
	for z := 0; z < len(path)/16; z++ {
		sibling := path[z*16 : (z+1)*16]
		leafAt := leaf >> uint(z)
		node := a.word(24, uint32(z+1)).word(28, uint32(base>>(z+1))+(leafAt>>1))
		if leafAt&1 == 0 {
			value = h.th(seed, node, value[:], sibling)
		} else {
			value = h.th(seed, node, sibling, value[:])
		}
	}
	return value
}

// Verify rejects malformed lengths before hashing. Verification has no state,
// counter search, attacker-controlled iteration count or dynamic parameters.
func Verify(publicKey, message, signature []byte) bool {
	valid, _ := verify(publicKey, message, signature)
	return valid
}

func verify(publicKey, message, signature []byte) (bool, uint64) {
	if len(publicKey) != PublicKeySize || len(message) != MessageSize || len(signature) != SignatureSize {
		return false, 0
	}
	h := new(hashing)
	seed, root := publicKey[:16], publicKey[16:]
	r := signature[:16]
	var expanded [64]byte
	for c := uint32(0); c < 2; c++ {
		var counter [4]byte
		binary.BigEndian.PutUint32(counter[:], c)
		copy(expanded[c*32:], h.hash(r, seed, root, message, counter[:]))
	}
	index := (uint64(expanded[30]&3) << 24) | (uint64(expanded[31]) << 16) | (uint64(expanded[32]) << 8) | uint64(expanded[33])
	forsAddr, _ := keyAddress(0, index, 3)
	roots := make([]byte, 0, 24*16)
	at := 16
	for c := 0; c < 24; c++ {
		var leaf uint32
		for j := 0; j < 10; j++ {
			p := c*10 + j
			leaf = (leaf << 1) | uint32((expanded[p/8]>>uint(7-p%8))&1)
		}
		value := h.th(seed, forsAddr.word(24, 0).word(28, uint32(c<<10)+leaf), signature[at:at+16])
		at += 16
		value = h.fold(seed, forsAddr, value, signature[at:at+160], leaf, c<<10)
		at += 160
		roots = append(roots, value[:]...)
	}
	signed := h.th(seed, forsAddr.kind(4).word(20, binary.BigEndian.Uint32(forsAddr[20:24])), roots)
	for layer, height := range [3]int{8, 8, 10} {
		counter := signature[at : at+4]
		at += 4
		key, leaf := keyAddress(layer, index, 7)
		encoded := h.th(seed, key, signed[:], counter)
		var chunks [64]byte
		sum := 0
		for i := range chunks {
			chunks[i] = (encoded[i/4] >> uint(6-2*(i%4))) & 3
			sum += int(chunks[i])
		}
		if sum != 128 {
			return false, h.permutations
		}
		ends := make([]byte, 0, 64*16)
		for i, start := range chunks {
			var value digest
			copy(value[:], signature[at:at+16])
			at += 16
			for step := uint32(start); step < 3; step++ {
				value = h.th(seed, key.kind(0).word(20, leaf).word(24, uint32(i)).word(28, step), value[:])
			}
			ends = append(ends, value[:]...)
		}
		signed = h.th(seed, key.kind(1).word(20, leaf), ends)
		signed = h.fold(seed, key.kind(2), signed, signature[at:at+height*16], leaf, 0)
		at += height * 16
	}
	return subtle.ConstantTimeCompare(signed[:], root) == 1, h.permutations
}
