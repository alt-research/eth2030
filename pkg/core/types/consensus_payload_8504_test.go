package types

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"testing"
)

func TestDependency8504IndependentGloasProgressivePayload(t *testing.T) {
	raw, err := os.ReadFile("testdata/dependency_ssz/vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors struct {
		Cases []struct {
			Name, SSZ, Root string
			Fields          int
		}
	}
	if err := json.Unmarshal(raw, &vectors); err != nil {
		t.Fatal(err)
	}
	for _, v := range vectors.Cases {
		h := &Header{ParentHash: BytesToHash(bytes.Repeat([]byte{1}, 32)), Coinbase: BytesToAddress(bytes.Repeat([]byte{2}, 20)), Root: BytesToHash(bytes.Repeat([]byte{3}, 32)), ReceiptHash: BytesToHash(bytes.Repeat([]byte{4}, 32)), MixDigest: BytesToHash(bytes.Repeat([]byte{5}, 32)), Number: big.NewInt(6), GasLimit: 7, GasUsed: 8, Time: 9, Extra: []byte{10, 11}, BaseFee: new(big.Int)}
		h.Bloom[3] = 0x5a
		blob, excess, slot := uint64(17), uint64(18), uint64(19)
		h.BlobGasUsed, h.ExcessBlobGas, h.SlotNumber = &blob, &excess, &slot
		p := &ConsensusPayload8504{Header: h, BlockHash: BytesToHash(bytes.Repeat([]byte{12}, 32)), Transactions: [][]byte{{2, 0xab}, {6}}, Withdrawals: []*Withdrawal{{Index: 13, ValidatorIndex: 14, Address: BytesToAddress(bytes.Repeat([]byte{15}, 20)), Amount: 16}}, BlockAccessList: []byte{0xc1, 0x80}}
		if v.Name == "empty" {
			p.DependencyWitnesses = [][]byte{}
		}
		if v.Name == "two" {
			p.DependencyWitnesses = [][]byte{bytes.Repeat([]byte{0xaa}, 7740), bytes.Repeat([]byte{0xbb}, 7740)}
		}
		got, err := p.MarshalSSZ()
		if err != nil {
			t.Fatal(err)
		}
		expected, _ := hex.DecodeString(v.SSZ)
		if !bytes.Equal(got, expected) {
			t.Fatal(v.Name, "independent SSZ mismatch")
		}
		root, err := p.HashTreeRoot()
		if err != nil || hex.EncodeToString(root[:]) != v.Root {
			t.Fatal(v.Name, "independent progressive root mismatch", root, err)
		}
		var decoded ConsensusPayload8504
		if err := decoded.UnmarshalSSZ(expected); err != nil {
			t.Fatal(err)
		}
		if (decoded.DependencyWitnesses != nil) != (v.Fields == 20) {
			t.Fatal("active empty field lost")
		}
		round, err := decoded.MarshalSSZ()
		if err != nil || !bytes.Equal(round, expected) {
			t.Fatal("SSZ roundtrip", err)
		}
		if roundRoot, err := decoded.HashTreeRoot(); err != nil || roundRoot != root {
			t.Fatal("root changed on decode")
		}
		bad := bytes.Clone(expected)
		clear(bad[436:440])
		if err := decoded.UnmarshalSSZ(bad); err == nil {
			t.Fatal("noncanonical first offset accepted")
		}
	}
}
