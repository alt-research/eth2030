package bls

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"testing"
)

func TestConsensusBLSIndependentBlstVector(t *testing.T) {
	data, err := os.ReadFile("../../../specs/hegota/bls-vector.json")
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]string
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatal(err)
	}
	decode := func(s string) []byte {
		b, err := hex.DecodeString(v[s])
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	pk, msg, sig := decode("pubkey"), decode("message"), decode("signature")
	b := &ConsensusBLSBackend{}

	secret := new(big.Int).SetBytes(decode("secret"))
	ownPK, ownSig := BLSPubkeyFromSecret(secret), BLSSign(secret, msg)
	if !bytes.Equal(ownPK[:], pk) || !bytes.Equal(ownSig[:], sig) {
		t.Fatal("canonical signer differs from independent blst vector")
	}
	var fixedSig [96]byte
	copy(fixedSig[:], sig)
	aggregate := AggregateSignatures([][96]byte{fixedSig, fixedSig})
	if !bytes.Equal(aggregate[:], decode("twice_signature")) || !b.FastAggregateVerify([][]byte{pk, pk}, msg, aggregate[:]) {
		t.Fatal("aggregate differs from independent blst vector")
	}
	if mixed := AggregateSignatures([][96]byte{fixedSig, {}}); mixed != ([96]byte{}) {
		t.Fatal("invalid aggregate member silently dropped")
	}
	if mixed := AggregatePublicKeys([][48]byte{ownPK, {}}); mixed != ([48]byte{}) {
		t.Fatal("invalid public key silently dropped")
	}
	if !b.Verify(pk, msg, sig) || !b.AggregateVerify([][]byte{pk}, [][]byte{msg}, sig) || !b.FastAggregateVerify([][]byte{pk}, msg, sig) {
		t.Fatal("independent blst signature rejected")
	}
	for _, tt := range []struct {
		name         string
		pk, msg, sig []byte
	}{{"wrong message", pk, []byte("wrong"), sig}, {"zero signature", pk, msg, make([]byte, 96)}, {"zero key", make([]byte, 48), msg, sig}, {"infinity key", BLSPointAtInfinityG1[:], msg, sig}, {"infinity signature", pk, msg, BLSPointAtInfinityG2[:]}, {"truncated key", pk[:47], msg, sig}, {"truncated signature", pk, msg, sig[:95]}} {
		t.Run(tt.name, func(t *testing.T) {
			if b.Verify(tt.pk, tt.msg, tt.sig) {
				t.Fatal("invalid proof accepted")
			}
		})
	}
	if b.AggregateVerify(nil, nil, sig) || b.AggregateVerify([][]byte{pk}, nil, sig) {
		t.Fatal("invalid aggregate accepted")
	}
}
func TestConsensusBackendSelection(t *testing.T) {
	b, err := SelectConsensusBLSBackend("pure-go")
	if err != nil || b.Name() != "gnark-consensus" {
		t.Fatal("consensus backend unavailable")
	}
	if _, err := SelectConsensusBLSBackend("fake"); err == nil {
		t.Fatal("unsupported backend accepted")
	}
	if blstConsensusFactory == nil {
		if _, err := SelectConsensusBLSBackend("blst"); err == nil {
			t.Fatal("uncompiled blst accepted")
		}
	}
}
