package execution

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func TestFrame8141ApprovalScopeMatrix(t *testing.T) {
	for flags := byte(0); flags < 4; flags++ {
		for scope := byte(0); scope < 8; scope++ {
			t.Run(fmt.Sprintf("flags_%d_scope_%d", flags, scope), func(t *testing.T) {
				tx, db, cfg, header := frameExecutionFixture(t)
				db.SetCode(tx.Sender, []byte{0x60, scope, 0x5f, 0x5f, 0xaa})
				tx.Frames[0].Flags = flags
				// A successful execution-only frame is followed by an independent payer.
				sponsor := types.Address{19: 0x90}
				db.AddBalance(sponsor, big.NewInt(100000000))
				db.SetCode(sponsor, []byte{0x60, 1, 0x5f, 0x5f, 0xaa})
				if scope == 2 && flags&2 != 0 {
					tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeVerify, Target: &sponsor, Flags: 1, Limits: types.FrameGasLimits{Execution: 5000}})
				}
				before := db.GetRoot()
				_, _, err := ApplyTransaction(cfg, db, header, types.NewTransaction(tx), new(gaspool.GasPool).AddGas(header.GasLimit))
				valid := (scope == 3 && flags == 3) || (scope == 2 && flags&2 != 0)
				if (err == nil) != valid {
					t.Fatalf("approval scope result: %v", err)
				}
				if !valid && db.GetRoot() != before {
					t.Fatal("invalid scope changed state")
				}
			})
		}
	}
}

func TestFrame8141IntrospectionBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name string
		code []byte
		ok   bool
	}{
		{"future_structure", []byte{0x60, 4, 0x60, 2, 0xb3, 0x50, 0}, true},
		{"past_status", []byte{0x60, 5, 0x5f, 0xb3, 0x50, 0}, true},
		{"current_status", []byte{0x60, 5, 0x60, 1, 0xb3, 0}, false},
		{"future_execution", []byte{0x60, 10, 0x60, 2, 0xb3, 0}, false},
		{"future_state", []byte{0x60, 11, 0x60, 2, 0xb3, 0}, false},
		{"unknown_frame_param", []byte{0x60, 12, 0x5f, 0xb3, 0}, false},
		{"unknown_tx_param", []byte{0x60, 5, 0xb0, 0}, false},
		{"keyed_index_on_base", []byte{0x60, 13, 0xb0, 0}, false},
		{"unknown_sig_param", []byte{0x60, 4, 0x5f, 0xb4, 0}, false},
		{"protocol_signature_length", []byte{0x60, 3, 0x5f, 0xb4, 0}, false},
		{"protocol_signature_copy", []byte{0x5f, 0x60, 1, 0x5f, 0x5f, 0xb5, 0}, false},
		{"signature_index", []byte{0x60, 1, 0x60, 2, 0xb4, 0}, false},
		{"frame_index", []byte{0x60, 3, 0x5f, 0xb1, 0}, false},
		{"zero_copy_invalid_index", []byte{0x60, 3, 0x5f, 0x5f, 0x5f, 0xb2, 0}, false},
		{"data_zero_padding", []byte{0x60, 2, 0x60, 255, 0xb1, 0x5f, 0x55, 0}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx, db, cfg, header := frameExecutionFixture(t)
			target := types.Address{19: 0x90}
			db.SetCode(target, tc.code)
			tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeSender, Target: &target, Limits: types.FrameGasLimits{Execution: 50000, State: 97920}}, types.Frame8141{Mode: types.ModeDefault, Data: []byte{1, 2, 3}, Limits: types.FrameGasLimits{Execution: 5000}})
			receipt, _, err := ApplyTransaction(cfg, db, header, signFrameExecution(t, tx, false), new(gaspool.GasPool).AddGas(header.GasLimit))
			if err != nil {
				t.Fatal(err)
			}
			got := receipt.FrameResults8141[1]
			if (got.Status == 1) != tc.ok {
				t.Fatalf("introspection status: %+v", got)
			}
			if !tc.ok && (got.GasUsed.Execution != 50000 || got.GasUsed.State != 0 || len(got.Logs) != 0) {
				t.Fatal("exception did not consume execution gas and roll back state")
			}
		})
	}
}

func TestFrame8141DefaultCodeDoesNotAuthenticateArbitraryWitness(t *testing.T) {
	tx, db, cfg, header := frameExecutionFixture(t)
	tx.Signatures = []types.FrameSignature{{Scheme: types.FrameSignatureArbitrary, Signature: make([]byte, 65)}}
	if err := tx.Validate(); err != nil {
		t.Fatal(err)
	}
	before := db.GetRoot()
	if _, _, err := ApplyTransaction(cfg, db, header, types.NewTransaction(tx), new(gaspool.GasPool).AddGas(header.GasLimit)); err == nil {
		t.Fatal("default code treated arbitrary bytes as protocol authentication")
	}
	if db.GetRoot() != before {
		t.Fatal("failed default validation mutated state")
	}
}

func TestFrame8141InvalidProtocolSignatureBeforeExecution(t *testing.T) {
	for _, kind := range []string{"recovery_id", "high_s", "wrong_signer", "unknown_scheme", "short_signature"} {
		t.Run(kind, func(t *testing.T) {
			tx, db, cfg, header := frameExecutionFixture(t)
			db.SetCode(tx.Sender, []byte{0x60, 3, 0x5f, 0x5f, 0xaa})
			transaction := signFrameExecution(t, tx, false)
			frame := transaction.Inner().(*types.FrameTx8141)
			sig := &frame.Signatures[0]
			switch kind {
			case "recovery_id":
				sig.Signature[0] = 27
			case "high_s":
				s := new(big.Int).SetBytes(sig.Signature[33:])
				s.Sub(gethcrypto.S256().Params().N, s)
				s.FillBytes(sig.Signature[33:])
				sig.Signature[0] ^= 1
			case "wrong_signer":
				other := types.Address{19: 99}
				sig.Signer = &other
			case "unknown_scheme":
				sig.Scheme = 3
			case "short_signature":
				sig.Signature = sig.Signature[:64]
			}
			before := db.GetRoot()
			gp := new(gaspool.GasPool).AddGas(header.GasLimit)
			if _, _, err := ApplyTransaction(cfg, db, header, types.NewTransaction(frame), gp); err == nil {
				t.Fatal("malformed protocol signature accepted")
			}
			if db.GetRoot() != before || gp.Gas() != header.GasLimit || gp.StateGas() != header.GasLimit {
				t.Fatal("invalid signature executed approval or reserved gas")
			}
		})
	}
}
