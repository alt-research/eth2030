package types

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"math"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/rlp"
)

func nonceProfile8250() *FrameTx8250Delta {
	return &FrameTx8250Delta{ChainID: big.NewInt(1), NonceKeys: []*big.Int{big.NewInt(0), big.NewInt(42)}, Sender: Address{19: 1},
		Frames:     []Frame8141{{Mode: ModeVerify, Flags: 3, Limits: FrameGasLimits{Execution: 50000, State: 195840}}},
		Signatures: []FrameSignature{{Scheme: FrameSignatureArbitrary, Signature: []byte{1, 2, 3}}},
		Fees:       FrameFeeBudget{MaxFee: big.NewInt(10000000), MaxPriorityFeePerGas: big.NewInt(1)}}
}

func TestFrame8250DeltaCodecAndSigningCommitments(t *testing.T) {
	tx := nonceProfile8250()
	if err := tx.Validate(); err != nil {
		t.Fatal(err)
	}
	wire, err := tx.Encode()
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeFrameTx8250Delta(wire[1:])
	if err != nil {
		t.Fatal(err)
	}
	again, _ := decoded.Encode()
	if !bytes.Equal(wire, again) {
		t.Fatal("noncanonical round trip")
	}
	if _, err := DecodeTxRLP(wire); err == nil {
		t.Fatal("experimental composition entered public decoder")
	}
	hash, _ := tx.SigningHash()
	tx.Signatures[0].Signature[0] ^= 255
	equal, _ := tx.SigningHash()
	if hash != equal {
		t.Fatal("empty-msg witness bytes committed")
	}
	tx.Signatures[0].Signature = append(tx.Signatures[0].Signature, 0)
	different, _ := tx.SigningHash()
	if hash == different {
		t.Fatal("witness length not committed")
	}
	tx.NonceKeys[1] = big.NewInt(43)
	keysChanged, _ := tx.SigningHash()
	if different == keysChanged {
		t.Fatal("nonce keys not signed")
	}
	tx.NonceSeq = math.MaxUint64
	seqChanged, _ := tx.SigningHash()
	if seqChanged == keysChanged {
		t.Fatal("single-use sequence not signed")
	}
}

func TestFrame8250DeltaRejectsMalformedNonceSets(t *testing.T) {
	for _, keys := range [][]*big.Int{{nil}, {big.NewInt(-1)}, {new(big.Int).Lsh(big.NewInt(1), 256)}, {big.NewInt(2), big.NewInt(1)}, {big.NewInt(0), big.NewInt(0)}} {
		if ValidateNonceKeys8250(keys, 0) == nil {
			t.Fatalf("accepted invalid keys: %v", keys)
		}
	}
	if ValidateNonceKeys8250(nil, math.MaxUint64) == nil {
		t.Fatal("empty single-use set accepted")
	}
	keys := make([]*big.Int, 17)
	for i := range keys {
		keys[i] = big.NewInt(int64(i))
	}
	if ValidateNonceKeys8250(keys, 0) == nil {
		t.Fatal("17 keys accepted")
	}
	// Canonical RLP integers exclude 0x00 and length-prefixed single bytes.
	tx := nonceProfile8250()
	fields := []any{tx.ChainID, rlp.RawValue{0xc1, 0x00}, tx.NonceSeq, tx.Sender, tx.Frames, tx.Signatures, tx.Fees, tx.BlobVersionedHashes}
	for _, bad := range []rlp.RawValue{{0xc1, 0x00}, {0xc2, 0x81, 0x01}, {0x80}, {0xc3, 0x82, 0x00, 0x01}} {
		fields[1] = bad
		wire, err := rlp.EncodeToBytes(fields)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := DecodeFrameTx8250Delta(wire); err == nil {
			t.Fatalf("accepted malformed key RLP %x", bad)
		}
	}
	fields[1] = []uint64{0}
	fields[2] = rlp.RawValue{0x89, 1, 0, 0, 0, 0, 0, 0, 0, 0}
	wire, _ := rlp.EncodeToBytes(fields)
	if _, err := DecodeFrameTx8250Delta(wire); err == nil {
		t.Fatal("overflowing sequence accepted")
	}
}

func TestFrame8250DeltaDataAndAccessPricing(t *testing.T) {
	tx := nonceProfile8250()
	base, err := tx.ExecutionFields().GasBudget()
	if err != nil {
		t.Fatal(err)
	}
	budget, err := tx.GasBudget()
	if err != nil {
		t.Fatal(err)
	}
	// rlp([0,42]) = c2 80 2a; rlp(0) = 80: all four bytes nonzero.
	if budget.Intrinsic != base.Intrinsic+4*16+2*12100 || budget.CalldataFloor != base.CalldataFloor+4*64+2*12100 {
		t.Fatalf("nonce data/access pricing: %+v base %+v", budget, base)
	}
	tx.NonceSeq = math.MaxUint64
	single, err := tx.GasBudget()
	if err != nil {
		t.Fatal(err)
	}
	if single.Intrinsic != budget.Intrinsic-16 || single.CalldataFloor != budget.CalldataFloor-64 {
		t.Fatal("single-use sequence exemption differs")
	}
	tx.NonceKeys = nil
	tx.NonceSeq = 0
	empty, err := tx.GasBudget()
	if err != nil {
		t.Fatal(err)
	}
	if empty.Intrinsic != base.Intrinsic+32 || empty.CalldataFloor != base.CalldataFloor+128 {
		t.Fatal("empty list and sequence not priced")
	}
}

func TestFrame8250DeltaIndependentWireGoldens(t *testing.T) {
	raw, err := os.ReadFile("../../../specs/hegota/b-codec-goldens.json")
	if err != nil {
		t.Fatal(err)
	}
	var corpus struct {
		Vectors []struct {
			ID          string `json:"id"`
			Wire        string `json:"wire"`
			Hash        string `json:"hash"`
			SigningHash string `json:"signing_hash"`
			KeySetHash  string `json:"key_set_hash"`
		} `json:"vectors"`
	}
	if err = json.Unmarshal(raw, &corpus); err != nil {
		t.Fatal(err)
	}
	if len(corpus.Vectors) != 4 {
		t.Fatal("missing independent vectors")
	}
	for _, v := range corpus.Vectors {
		t.Run(v.ID, func(t *testing.T) {
			wire, err := hex.DecodeString(v.Wire[2:])
			if err != nil {
				t.Fatal(err)
			}
			tx, err := DecodeFrameTx8250Delta(wire[1:])
			if err != nil {
				t.Fatal(err)
			}
			if err = tx.Validate(); err != nil {
				t.Fatal(err)
			}
			encoded, err := tx.Encode()
			if err != nil || !bytes.Equal(encoded, wire) {
				t.Fatal("independent wire differs", err)
			}
			hash, _ := tx.Hash()
			signing, _ := tx.SigningHash()
			if hash != HexToHash(v.Hash) || signing != HexToHash(v.SigningHash) || NonceKeysHash8250(tx.NonceKeys) != HexToHash(v.KeySetHash) {
				t.Fatal("independent digest differs")
			}
		})
	}
}

func TestFrame8141RejectsNoncanonicalIntegerRLP(t *testing.T) {
	tx := nonceProfile8250().ExecutionFields()
	fields := []any{tx.ChainID, uint64(0), tx.Sender, tx.Frames, tx.Signatures, tx.Fees, tx.BlobVersionedHashes}
	for _, bad := range []rlp.RawValue{{0x00}, {0x81, 0x01}, {0x82, 0x00, 0x01}, {0x89, 1, 0, 0, 0, 0, 0, 0, 0, 0}} {
		fields[1] = bad
		wire, err := rlp.EncodeToBytes(fields)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := DecodeFrameTx8141(wire); err == nil {
			t.Fatalf("accepted noncanonical/overflow nonce %x", bad)
		}
	}
}
