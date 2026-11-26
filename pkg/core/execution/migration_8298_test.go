package execution

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/bal"
	"github.com/eth2030/eth2030/core/eips"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func TestMigration8298NativeDelegationAdoptionRetirementAndFrameControl(t *testing.T) {
	frame, db, cfg, h := frameExecutionFixture(t)
	zero := uint64(0)
	cfg.AmsterdamTime, cfg.EIP8298Time, cfg.GlamsterdanTime = &zero, &zero, &zero
	key, _ := gethcrypto.HexToECDSA("0000000000000000000000000000000000000000000000000000000000000001")
	sender := frame.Sender
	initializer, template := types.Address{18: 0x91}, types.Address{18: 0x92}
	code := append([]byte{byte(vm.PUSH20)}, template[:]...)
	code = append(code, byte(vm.SETCODEFROM), byte(vm.POP), byte(vm.STOP))
	wallet := []byte{0x60, 3, 0x5f, 0x5f, byte(vm.APPROVE)}
	db.SetCode(initializer, code)
	db.SetCode(template, wallet)
	db.SetCode(sender, append([]byte{0xef, 1, 0}, initializer[:]...))
	signed := func(nonce uint64) *types.Transaction {
		to := gethcommon.Address(sender)
		g := gethtypes.NewTx(&gethtypes.DynamicFeeTx{ChainID: cfg.ChainID, Nonce: nonce, To: &to, Gas: 100000, GasFeeCap: big.NewInt(10), GasTipCap: big.NewInt(2), Value: new(big.Int)})
		g, err := gethtypes.SignTx(g, gethtypes.NewLondonSigner(cfg.ChainID), key)
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := g.MarshalBinary()
		tx, err := types.DecodeTxRLP(raw)
		if err != nil {
			t.Fatal(err)
		}
		return tx
	}
	tx := signed(0)
	db.SetTxContext(tx.Hash(), 0)
	tracker := bal.NewTracker()
	r, _, err := ApplyTransactionWithBAL(cfg, db, h, tx, new(gaspool.GasPool).AddGas(h.GasLimit), tracker)
	if err != nil || r.Status != 1 || !bytes.Equal(db.GetCode(sender), wallet) || !bytes.Equal(db.GetCode(initializer), code) {
		t.Fatal("native migration", err, r)
	}
	tracker.Reconcile(db)
	list := tracker.Build(1)
	wire, err := list.EncodeRLP()
	if err != nil {
		t.Fatal(err)
	}
	accounts, err := bal.DecodeRLP(wire)
	if err != nil {
		t.Fatal(err)
	}
	adopted := false
	for _, a := range accounts {
		if a.Address == sender {
			hash, ok := a.AdoptedCodeHash(1)
			adopted = ok && hash == db.GetCodeHash(template)
		}
	}
	if !adopted {
		t.Fatal("native migration emitted bytecode instead of hash-only BAL")
	}
	root := db.GetRoot()
	if _, _, err := ApplyTransaction(cfg, db, h, signed(1), new(gaspool.GasPool).AddGas(h.GasLimit)); err == nil || db.GetRoot() != root {
		t.Fatal("retired key originated ordinary transaction", err)
	}
	auth := types.Authorization{ChainID: cfg.ChainID, Nonce: 1, Address: initializer}
	sig, _ := gethcrypto.Sign(eips.AuthorizationHash(&auth), key)
	auth.V = new(big.Int).SetUint64(uint64(sig[64]))
	auth.R = new(big.Int).SetBytes(sig[:32])
	auth.S = new(big.Int).SetBytes(sig[32:64])
	_ = eips.ProcessAuthorizations(db, []types.Authorization{auth}, cfg.ChainID)
	if !bytes.Equal(db.GetCode(sender), wallet) || db.GetNonce(sender) != 1 {
		t.Fatal("retired key redelegated account")
	}
	frame.Nonce = 1
	frame.Frames[0].Limits.Execution = 5000
	frame.Signatures = nil
	if _, _, err := ApplyTransaction(cfg, db, h, types.NewTransaction(frame), new(gaspool.GasPool).AddGas(h.GasLimit)); err != nil || db.GetNonce(sender) != 2 {
		t.Fatal("migration stranded Frame control", err)
	}
	// Contract-level ECDSA remains until the separate EIP-8151 activation.
	message := gethcrypto.Keccak256([]byte("retired authorization"))
	sig, _ = gethcrypto.Sign(message, key)
	input := make([]byte, 128)
	copy(input, message)
	input[63] = 27 + sig[64]
	copy(input[64:], sig[:64])
	for _, restricted := range []bool{false, true} {
		cfg.EIP8151Time = nil
		if restricted {
			cfg.EIP8151Time = &zero
		}
		evm := vm.NewEVMWithState(vm.BlockContext{}, vm.TxContext{}, vm.Config{}, db)
		if _, err := ConfigureBlockEVM(cfg, h, evm); err != nil {
			t.Fatal(err)
		}
		out, _, err := evm.StaticCall(types.Address{19: 0x90}, types.Address{19: 1}, input, 20000)
		if err != nil || (len(out) == 0) != restricted {
			t.Fatal("companion recovery boundary", restricted, err)
		}
	}
}
