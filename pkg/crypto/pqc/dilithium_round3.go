package pqc

import "github.com/cloudflare/circl/sign/dilithium/mode3"

// Dilithium3Round3 implements the NIST round-3 Dilithium3 wire format.
// Its keys and signatures are for callers explicitly using that format.
type Dilithium3Round3 struct{}

func (Dilithium3Round3) GenerateKey() (public, private []byte, err error) {
	pk, sk, err := mode3.GenerateKey(nil)
	if err != nil {
		return nil, nil, err
	}
	return pk.Bytes(), sk.Bytes(), nil
}

func (Dilithium3Round3) Sign(private, message []byte) ([]byte, error) {
	var sk mode3.PrivateKey
	if err := sk.UnmarshalBinary(private); err != nil {
		return nil, err
	}
	signature := make([]byte, mode3.SignatureSize)
	mode3.SignTo(&sk, message, signature)
	return signature, nil
}

func (Dilithium3Round3) Verify(public, message, signature []byte) bool {
	var pk mode3.PublicKey
	return len(signature) == mode3.SignatureSize &&
		pk.UnmarshalBinary(public) == nil &&
		mode3.Verify(&pk, message, signature)
}
