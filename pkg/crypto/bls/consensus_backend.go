package bls

// ConsensusBLSBackend uses the maintained gnark-crypto implementation and the
// Ethereum POP ciphersuite. It never consults local secret keys or test vectors.
import (
	"fmt"
	g "github.com/consensys/gnark-crypto/ecc/bls12-381"
)

type ConsensusBLSBackend struct{}

func (*ConsensusBLSBackend) Name() string { return "gnark-consensus" }
func (b *ConsensusBLSBackend) Verify(pk, msg, sig []byte) bool {
	return b.AggregateVerify([][]byte{pk}, [][]byte{msg}, sig)
}
func (*ConsensusBLSBackend) AggregateVerify(keys, messages [][]byte, signature []byte) bool {
	if len(keys) == 0 || len(keys) != len(messages) || len(signature) != 96 {
		return false
	}
	var sig g.G2Affine
	if _, err := sig.SetBytes(signature); err != nil || sig.IsInfinity() || !sig.IsInSubGroup() {
		return false
	}
	p := make([]g.G1Affine, len(keys)+1)
	q := make([]g.G2Affine, len(keys)+1)
	for i := range keys {
		if len(keys[i]) != 48 {
			return false
		}
		if _, err := p[i].SetBytes(keys[i]); err != nil || p[i].IsInfinity() || !p[i].IsInSubGroup() {
			return false
		}
		h, err := g.HashToG2(messages[i], blsSignDST)
		if err != nil {
			return false
		}
		q[i] = h
	}
	_, _, generator, _ := g.Generators()
	p[len(keys)].Neg(&generator)
	q[len(keys)] = sig
	ok, err := g.PairingCheck(p, q)
	return err == nil && ok
}
func (b *ConsensusBLSBackend) FastAggregateVerify(keys [][]byte, msg, signature []byte) bool {
	// Callers must already have verified proof of possession, as required by POP.
	messages := make([][]byte, len(keys))
	for i := range messages {
		messages[i] = msg
	}
	return b.AggregateVerify(keys, messages, signature)
}

// A build-tagged adapter supplies this factory; unsupported requests fail closed.
var blstConsensusFactory func() BLSBackend

func SelectConsensusBLSBackend(name string) (BLSBackend, error) {
	switch name {
	case "", "pure-go":
		return &ConsensusBLSBackend{}, nil
	case "blst":
		if blstConsensusFactory != nil {
			return blstConsensusFactory(), nil
		}
		return nil, fmt.Errorf("blst backend unavailable: rebuild with -tags blst")
	default:
		return nil, fmt.Errorf("unsupported BLS backend %q", name)
	}
}
