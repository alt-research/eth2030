package pqc

import "github.com/eth2030/eth2030/crypto/pqc/internal/pqclean"

// ErrRound3ProviderUnavailable means this binary was built without CGO.
var ErrRound3ProviderUnavailable = pqclean.ErrUnavailable

// Falcon512Round3 uses the original padded Falcon-512 wire format:
// 897-byte public keys, 1281-byte private keys, and 666-byte signatures.
type Falcon512Round3 struct{}

func (Falcon512Round3) GenerateKey() (public, private []byte, err error) {
	return pqclean.Falcon512.GenerateKey()
}

func (Falcon512Round3) Sign(private, message []byte) ([]byte, error) {
	return pqclean.Falcon512.Sign(private, message)
}

func (Falcon512Round3) Verify(public, message, signature []byte) bool {
	return pqclean.Falcon512.Verify(public, message, signature)
}

// SPHINCSPlusSHA2128fRound3 uses the round-3.1 SHA2-128f-simple wire format.
type SPHINCSPlusSHA2128fRound3 struct{}

func (SPHINCSPlusSHA2128fRound3) GenerateKey() (public, private []byte, err error) {
	return pqclean.SPHINCS128f.GenerateKey()
}

func (SPHINCSPlusSHA2128fRound3) Sign(private, message []byte) ([]byte, error) {
	return pqclean.SPHINCS128f.Sign(private, message)
}

func (SPHINCSPlusSHA2128fRound3) Verify(public, message, signature []byte) bool {
	return pqclean.SPHINCS128f.Verify(public, message, signature)
}
