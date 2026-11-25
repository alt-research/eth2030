package bls

// BLS aggregate signature operations for Ethereum consensus layer.
//
// Implements BLS12-381 aggregate public key aggregation, signature
// aggregation, and aggregate/fast-aggregate verification as required
// by the beacon chain spec (Altair+).
//
// Public keys are 48-byte compressed G1 points. Signatures are 96-byte
// compressed G2 points. The signature path uses RFC 9380 hash-to-curve.
//
// Signature scheme: signatures live in G2, public keys in G1.
// e(pk, H(m)) == e(G1, sig) for single verification.
// For aggregate: e(G1, aggSig) == product(e(pk_i, H(m_i))).

import (
	"crypto/sha256"
	"math/big"

	bls381 "github.com/consensys/gnark-crypto/ecc/bls12-381"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
)

// BLS aggregate signature sizes.
const (
	BLSPubkeySize    = 48
	BLSSignatureSize = 96
)

// SerializeG1 compresses a G1 point to 48 bytes.
func SerializeG1(p *BlsG1Point) [BLSPubkeySize]byte {
	point := p.gnarkAffine()
	return point.Bytes()
}

// DeserializeG1 decompresses a G1 point and verifies subgroup membership.
func DeserializeG1(data [BLSPubkeySize]byte) *BlsG1Point {
	var point bls381.G1Affine
	if _, err := point.SetBytes(data[:]); err != nil {
		return nil
	}
	return blsG1FromGnark(point)
}

// SerializeG2 compresses a G2 point to 96 bytes.
func SerializeG2(p *BlsG2Point) [BLSSignatureSize]byte {
	point := p.gnarkAffine()
	return point.Bytes()
}

// DeserializeG2 decompresses a G2 point and verifies subgroup membership.
func DeserializeG2(data [BLSSignatureSize]byte) *BlsG2Point {
	var point bls381.G2Affine
	if _, err := point.SetBytes(data[:]); err != nil {
		return nil
	}
	return blsG2FromGnark(point)
}

// AggregatePublicKeys aggregates multiple BLS12-381 public keys by
// adding the corresponding G1 points.
func AggregatePublicKeys(pubkeys [][48]byte) [48]byte {
	var aggregate bls381.G1Affine
	for _, encoded := range pubkeys {
		var p bls381.G1Affine
		if _, err := p.SetBytes(encoded[:]); err != nil || p.IsInfinity() || !p.IsInSubGroup() {
			return [48]byte{}
		}
		aggregate.Add(&aggregate, &p)
	}
	return aggregate.Bytes()
}

// AggregateSignatures aggregates multiple BLS signatures by adding
// the corresponding G2 points.
func AggregateSignatures(signatures [][96]byte) [96]byte {
	var aggregate bls381.G2Affine
	for _, encoded := range signatures {
		var p bls381.G2Affine
		if _, err := p.SetBytes(encoded[:]); err != nil || p.IsInfinity() || !p.IsInSubGroup() {
			return [96]byte{}
		}
		aggregate.Add(&aggregate, &p)
	}
	return aggregate.Bytes()
}

// HashToG2 hashes a message using the BLS12-381 G2 SSWU ciphersuite.
func HashToG2(msg []byte, dst []byte) *BlsG2Point {
	if len(dst) > 255 {
		h := sha256.New()
		h.Write([]byte("H2C-OVERSIZE-DST-"))
		h.Write(dst)
		dst = h.Sum(nil)
	}
	point, err := bls381.HashToG2(msg, dst)
	if err != nil {
		return BlsG2Infinity()
	}
	return blsG2FromAffine(
		&blsFp2{c0: point.X.A0.BigInt(new(big.Int)), c1: point.X.A1.BigInt(new(big.Int))},
		&blsFp2{c0: point.Y.A0.BigInt(new(big.Int)), c1: point.Y.A1.BigInt(new(big.Int))},
	)
}

// blsSignDST is the domain separation tag for BLS signatures.
var blsSignDST = []byte("BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_POP_")

// BLSSign signs a message with a secret key (scalar), returning a G2 signature.
// secret is the private key scalar.
func BLSSign(secret *big.Int, msg []byte) [BLSSignatureSize]byte {
	if secret == nil || secret.Sign() <= 0 || secret.Cmp(fr.Modulus()) >= 0 {
		return [BLSSignatureSize]byte{}
	}
	point, err := bls381.HashToG2(msg, blsSignDST)
	if err != nil {
		return [BLSSignatureSize]byte{}
	}
	var signature bls381.G2Affine
	signature.ScalarMultiplication(&point, secret)
	return signature.Bytes()
}

// BLSVerify verifies a single BLS signature.
// Checks: e(pk, H(m)) == e(G1, sig)
// Equivalent to: e(pk, H(m)) * e(-G1, sig) == 1
func BLSVerify(pubkey [BLSPubkeySize]byte, msg []byte, sig [BLSSignatureSize]byte) bool {
	return (&ConsensusBLSBackend{}).Verify(pubkey[:], msg, sig[:])
}

// VerifyAggregate verifies an aggregate signature where each signer
// signed a different message.
// Checks: e(G1, aggSig) == product(e(pk_i, H(m_i)))
// Equivalent to: product(e(pk_i, H(m_i))) * e(-G1, aggSig) == 1
func VerifyAggregate(pubkeys [][48]byte, msgs [][]byte, sig [96]byte) bool {
	keys := make([][]byte, len(pubkeys))
	for i := range pubkeys {
		keys[i] = pubkeys[i][:]
	}
	return (&ConsensusBLSBackend{}).AggregateVerify(keys, msgs, sig[:])
}

// FastAggregateVerify verifies an aggregate signature where all signers
// signed the same message. This is the common case for sync committee
// and attestation signatures.
// Checks: e(aggPK, H(m)) == e(G1, aggSig)
func FastAggregateVerify(pubkeys [][48]byte, msg []byte, sig [96]byte) bool {
	keys := make([][]byte, len(pubkeys))
	for i := range pubkeys {
		keys[i] = pubkeys[i][:]
	}
	return (&ConsensusBLSBackend{}).FastAggregateVerify(keys, msg, sig[:])
}

// BLSPubkeyFromSecret derives the public key from a secret scalar.
// pk = secret * G1
// Uses the canonical subgroup generator; no secret-key registry is retained.
func BLSPubkeyFromSecret(secret *big.Int) [BLSPubkeySize]byte {
	if secret == nil || secret.Sign() <= 0 || secret.Cmp(fr.Modulus()) >= 0 {
		return [BLSPubkeySize]byte{}
	}
	_, _, generator, _ := bls381.Generators()
	var pubkey bls381.G1Affine
	pubkey.ScalarMultiplication(&generator, secret)
	return pubkey.Bytes()
}
