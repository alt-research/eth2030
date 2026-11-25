package main

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/types"
)

func TestMakeCanonicalFrame(t *testing.T) {
	for _, sponsored := range []bool{false, true} {
		t.Run(fmt.Sprintf("sponsored=%v", sponsored), func(t *testing.T) {
			opts := options{key: fmt.Sprintf("%064x", 1), data: "0x1234", value: "7", execution: 50000, state: 200000}
			if sponsored {
				opts.sponsor = fmt.Sprintf("%064x", 2)
			}
			tx, err := makeFrame(opts, big.NewInt(1337), 4, big.NewInt(10), big.NewInt(1))
			if err != nil {
				t.Fatal(err)
			}
			raw, err := types.EncodePooledTransaction(tx)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := types.DecodePooledTransaction(raw)
			if err != nil {
				t.Fatal(err)
			}
			frame, ok := decoded.Inner().(*types.FrameTx8141)
			if !ok || frame.Validate() != nil || frame.Nonce != 4 || len(frame.Frames) != 2+len(frame.Signatures)-1 {
				t.Fatalf("not a valid signed canonical Frame: %T", decoded.Inner())
			}
			application := frame.Frames[len(frame.Frames)-1]
			if application.Value.Uint64() != 7 || application.Limits.State != 200000 || string(application.Data) != string([]byte{0x12, 0x34}) {
				t.Fatal("application fields were not signed")
			}
			if _, err := frame.Payment(big.NewInt(20), big.NewInt(1)); err != nil {
				t.Fatalf("automatic fee budget has no headroom: %v", err)
			}
			frame.Nonce++
			if frame.Validate() == nil {
				t.Fatal("nonce mutation reused a signature")
			}
		})
	}
}

func TestMakeFrameRejectsInvalidInputs(t *testing.T) {
	for _, change := range []func(*options){
		func(opts *options) { opts.sender = "0x0000000000000000000000000000000000000001" },
		func(opts *options) { opts.to = "0x01" },
		func(opts *options) { opts.data = "0xzz" },
		func(opts *options) { opts.value = "-1" },
		func(opts *options) { opts.execution = 1 << 24 },
	} {
		opts := options{key: fmt.Sprintf("%064x", 1), data: "0x", value: "0", execution: 10000}
		change(&opts)
		if _, err := makeFrame(opts, big.NewInt(1337), 0, big.NewInt(1), big.NewInt(1)); err == nil {
			t.Fatal("invalid input accepted")
		}
	}
}

func TestMakeFrameUsesPaymasterOwnerMetadata(t *testing.T) {
	opts := options{key: fmt.Sprintf("%064x", 1), sponsor: fmt.Sprintf("%064x", 2), paymaster: "0x0000000000000000000000000000000000007300", data: "0x", value: "0", execution: 30000, state: 97920}
	tx, err := makeFrame(opts, big.NewInt(1337), 0, big.NewInt(1), big.NewInt(1))
	if err != nil {
		t.Fatal(err)
	}
	frame := tx.Inner().(*types.FrameTx8141)
	if frame.Validate() != nil || frame.Frames[1].Target == nil || *frame.Frames[1].Target != types.HexToAddress(opts.paymaster) || string(frame.Frames[1].Data) != string([]byte{0x63, 0x4b, 1, 0xe8}) || frame.Signatures[1].Signer == nil || *frame.Signatures[1].Signer == *frame.Frames[1].Target {
		t.Fatal("coded payer and signature owner were confused")
	}
	opts.sponsor = ""
	if _, err := makeFrame(opts, big.NewInt(1337), 0, big.NewInt(1), big.NewInt(1)); err == nil {
		t.Fatal("paymaster without owner signature accepted")
	}
}
