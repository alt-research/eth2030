//go:build ignore

// Generate a fixed cross-library verifier vector; run explicitly from pkg/.
// Verification never regenerates its own oracle.
package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	blst "github.com/supranational/blst/bindings/go"
)

func main() {
	message := []byte("ETH2030 HGT-008 independent BLS POP vector")
	sk := blst.KeyGen(bytes.Repeat([]byte{0x42}, 32))
	pk := new(blst.P1Affine).From(sk).Compress()
	sig := new(blst.P2Affine).Sign(sk, message, []byte("BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_POP_")).Compress()
	aggregate := new(blst.P2Aggregate)
	if !aggregate.Aggregate([]*blst.P2Affine{new(blst.P2Affine).Uncompress(sig), new(blst.P2Affine).Uncompress(sig)}, true) {
		panic("aggregate vector")
	}
	b, _ := json.MarshalIndent(map[string]string{"generator": "supranational/blst pinned in pkg/go.mod; tools/hegota/bls_vector.go", "secret": hex.EncodeToString(sk.Serialize()), "pubkey": hex.EncodeToString(pk), "message": hex.EncodeToString(message), "signature": hex.EncodeToString(sig), "twice_signature": hex.EncodeToString(aggregate.ToAffine().Compress())}, "", "  ")
	fmt.Println(string(b))
}
