package snap

import (
	"bytes"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	gethsnap "github.com/ethereum/go-ethereum/eth/protocols/snap"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
)

func TestAccountWireRejectsMalformedRawBodiesWithoutPartialPublication(t *testing.T) {
	_, _, original := independentAccountPacket(t, 3)
	valid := bytes.Clone(original.Accounts[0].Body)
	for _, fault := range []string{"wrapped-body", "missing-field", "extra-field", "short-root", "short-code", "leading-zero-nonce", "leading-zero-balance", "overflow-balance", "trailing-packet", "oversized-packet"} {
		t.Run(fault, func(t *testing.T) {
			packet := *original
			entry := *original.Accounts[0]
			packet.Accounts = []*gethsnap.AccountData{&entry}
			var encoded []byte
			var err error
			switch fault {
			case "wrapped-body":
				encoded, err = gethrlp.EncodeToBytes([]byte(valid))
				entry.Body = encoded
			case "missing-field":
				entry.Body = []byte{0xc0}
			default:
				var fields []gethrlp.RawValue
				if err := gethrlp.DecodeBytes(valid, &fields); err != nil {
					t.Fatal(err)
				}
				switch fault {
				case "extra-field":
					fields = append(fields, gethrlp.RawValue{0x80})
				case "short-root":
					fields[2] = gethrlp.RawValue{1}
				case "short-code":
					fields[3] = gethrlp.RawValue{1}
				case "leading-zero-nonce":
					fields[0] = gethrlp.RawValue{0}
				case "leading-zero-balance":
					fields[1] = gethrlp.RawValue{0x82, 0, 1}
				case "overflow-balance":
					raw := make([]byte, 33)
					raw[0] = 1
					fields[1], err = gethrlp.EncodeToBytes(raw)
				}
				entry.Body, err = gethrlp.EncodeToBytes(fields)
			}
			if err != nil {
				t.Fatal(err)
			}
			encoded, err = gethrlp.EncodeToBytes(&packet)
			if err != nil {
				t.Fatal(err)
			}
			if fault == "trailing-packet" {
				encoded = append(encoded, 0x80)
			}
			if fault == "oversized-packet" {
				encoded = make([]byte, HardResponseLimit+1)
			}
			native := AccountRangePacket{ID: 77, Accounts: []AccountData{{Hash: types.Hash{1}, Body: valid}}}
			if err := native.DecodeRLP(encoded); err == nil {
				t.Fatalf("accepted malformed %s", fault)
			}
			if native.ID != 77 || len(native.Accounts) != 1 || native.Accounts[0].Hash != (types.Hash{1}) {
				t.Fatal("failed packet partially changed the consumer")
			}
		})
	}
}

func TestAccountWireKeepsIndependentPageSizeAndCanonicalBytes(t *testing.T) {
	_, _, packet := independentAccountPacket(t, MaxAccountRangeCount+1)
	// Native serving has its own batching policy; it must not reject a valid
	// independent peer page merely because that peer batches more accounts.

	encoded, err := gethrlp.EncodeToBytes(packet)
	if err != nil {
		t.Fatal(err)
	}
	var native AccountRangePacket
	if err := native.DecodeRLP(encoded); err != nil {
		t.Fatal(err)
	}
	if len(native.Accounts) != MaxAccountRangeCount+1 {
		t.Fatal("valid wire page was truncated")
	}
	roundtrip, err := native.EncodeRLP()
	if err != nil || !bytes.Equal(encoded, roundtrip) {
		t.Fatalf("canonical independent raw packet changed: %v", err)
	}
	native.Proof = [][]byte{make([]byte, HardResponseLimit)}
	if _, err := native.EncodeRLP(); err == nil {
		t.Fatal("oversized outgoing response accepted")
	}
}
