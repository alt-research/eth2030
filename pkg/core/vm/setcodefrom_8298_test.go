package vm

import (
	"bytes"
	"errors"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/bal"
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
)

func adoptionEVM8298() (*EVM, *state.MemoryStateDB, *bal.AccessTracker) {
	db := state.NewMemoryStateDB()
	evm := NewEVMWithState(BlockContext{}, TxContext{}, Config{}, db)
	rules := ForkRules{IsAmsterdam: true, IsLondon: true, IsCancun: true, IsPrague: true, IsEIP8298: true}
	evm.SetJumpTable(SelectJumpTable(rules))
	evm.SetForkRules(rules)
	tracker := bal.NewTracker()
	evm.SetBALTracker(tracker, 1)
	return evm, db, tracker
}

func TestSetCodeFrom8298GasValidityAndPaidAccess(t *testing.T) {
	source, target := types.Address{19: 0x41}, types.Address{19: 0x42}
	for _, code := range [][]byte{nil, {0xef}, {0xef, 1, 0}, {0x00}, bytes.Repeat([]byte{0}, 65536)} {
		for _, warm := range []bool{false, true} {
			for _, same := range []bool{false, true} {
				evm, db, tracker := adoptionEVM8298()
				db.SetCode(source, code)
				if same {
					db.SetCode(target, code)
				}
				if warm {
					db.AddAddressToAccessList(source)
				}
				valid := len(code) != 0 && code[0] != 0xef
				want := uint64(3100)
				if warm {
					want = 200
				}
				if valid {
					want += 100
					if !same {
						want += 9000
					}
				}
				stack := NewStack()
				stack.Push(new(big.Int).SetBytes(source[:]))
				contract := NewContract(types.Address{}, target, new(big.Int), want)
				cost, err := gasSetCodeFrom8298(evm, contract, stack, NewMemory(), 0)
				if err != nil || cost != want {
					t.Fatal("gas", cost, want, err)
				}
				if _, err = opSetCodeFrom8298(nil, evm, contract, nil, stack); err != nil || stack.Peek().Uint64() != map[bool]uint64{true: 1}[valid] {
					t.Fatal("validity", err)
				}
				if valid && !bytes.Equal(db.GetCode(target), code) {
					t.Fatal("adoption lost bytes")
				}
				if !db.AddressInAccessList(source) || tracker.Build(1).Len() == 0 {
					t.Fatal("paid invalid lookup absent from BAL")
				}
			}
		}
	}
	// With insufficient source gas, no account lookup enters the BAL or warm set.
	evm, db, tracker := adoptionEVM8298()
	s := NewStack()
	s.Push(new(big.Int).SetBytes(source[:]))
	if _, err := gasSetCodeFrom8298(evm, NewContract(target, target, nil, 3099), s, nil, 0); !errors.Is(err, ErrOutOfGas) || db.AddressInAccessList(source) || tracker.Build(1).Len() != 0 {
		t.Fatal("unpaid source access", err)
	}
}

func TestSetCodeFrom8298ContextsCreationAndRollback(t *testing.T) {
	source, target, helper := types.Address{19: 0x41}, types.Address{19: 0x42}, types.Address{19: 0x43}
	adopt := append([]byte{byte(PUSH20)}, source[:]...)
	adopt = append(adopt, byte(SETCODEFROM), byte(POP), byte(STOP))
	for _, kind := range []string{"call", "delegate", "callcode", "7702"} {
		evm, db, _ := adoptionEVM8298()
		db.SetCode(source, []byte{byte(PUSH1), 1, byte(STOP)})
		db.SetCode(helper, adopt)
		var err error
		switch kind {
		case "call":
			db.SetCode(target, adopt)
			_, _, err = evm.Call(helper, target, nil, 100000, new(big.Int))
		case "delegate":
			_, _, err = evm.DelegateCall(target, helper, nil, 100000)
		case "callcode":
			_, _, err = evm.CallCode(target, helper, nil, 100000, new(big.Int))
		case "7702":
			db.SetCode(target, append([]byte{0xef, 1, 0}, helper[:]...))
			_, _, err = evm.Call(helper, target, nil, 100000, new(big.Int))
		}
		if err != nil || !bytes.Equal(db.GetCode(target), db.GetCode(source)) || !bytes.Equal(db.GetCode(helper), adopt) {
			t.Fatal(kind, err)
		}
		// The adopter retains bytes after its source replaces code.
		db.SetCode(source, []byte{0xfe})
		if len(db.GetCode(target)) != 3 {
			t.Fatal("source replacement changed adopter")
		}
	}
	for _, creation := range []string{"create", "create2"} {
		evm, db, _ := adoptionEVM8298()
		runtime := []byte{0x60, 0x42, 0x5f, 0x52, 0x60, 32, 0x5f, 0xf3}
		db.SetCode(source, runtime)
		// Return the forbidden EF prefix after adoption: it must be ignored.
		init := bytes.Clone(adopt[:len(adopt)-1])
		// Reenter during initcode and persist the adopted runtime's answer.
		init = append(init, byte(PUSH1), 32, byte(PUSH0), byte(PUSH0), byte(PUSH0), byte(PUSH0), byte(ADDRESS), byte(PUSH2), 0xff, 0xff, byte(CALL), byte(POP), byte(PUSH0), byte(MLOAD), byte(PUSH0), byte(SSTORE))
		init = append(init, byte(PUSH1), 0xef, byte(PUSH0), byte(MSTORE8), byte(PUSH1), 1, byte(PUSH0), byte(RETURN))
		var ret []byte
		var address types.Address
		var err error
		if creation == "create" {
			ret, address, _, err = evm.Create(target, init, 1000000, new(big.Int))
		} else {
			ret, address, _, err = evm.Create2(target, init, 1000000, new(big.Int), new(big.Int))
		}
		if err != nil || !bytes.Equal(ret, runtime) || !bytes.Equal(db.GetCode(address), ret) || db.GetState(address, types.Hash{}) != (types.Hash{31: 0x42}) {
			t.Fatal(creation, ret, err)
		}
		// An adopted account created in this transaction remains an invalid source.
		s := NewStack()
		s.Push(new(big.Int).SetBytes(address[:]))
		c := NewContract(target, helper, nil, 100000)
		_, _ = gasSetCodeFrom8298(evm, c, s, nil, 0)
		_, _ = opSetCodeFrom8298(nil, evm, c, nil, s)
		if s.Peek().Sign() != 0 {
			t.Fatal("accepted same-transaction source")
		}
	}
	evm, db, tracker := adoptionEVM8298()
	db.SetCode(source, []byte{0})
	db.SetCode(target, append(bytes.Clone(adopt[:len(adopt)-1]), byte(PUSH0), byte(PUSH0), byte(REVERT)))
	before := bytes.Clone(db.GetCode(target))
	_, _, err := evm.Call(helper, target, nil, 100000, new(big.Int))
	tracker.Reconcile(db)
	if !errors.Is(err, ErrExecutionReverted) || !bytes.Equal(db.GetCode(target), before) || len(evm.codeAdoptions.origins) != 0 {
		t.Fatal("adoption escaped rollback", err)
	}
}

func TestSetCodeFrom8298EarlyGuardsAndImmediateScan(t *testing.T) {
	for _, system := range []bool{false, true} {
		evm, db, tracker := adoptionEVM8298()
		target, source := types.Address{19: 0x41}, types.Address{19: 0x42}
		code := append([]byte{byte(PUSH20)}, source[:]...)
		code = append(code, byte(SETCODEFROM))
		if system {
			evm.SetSystemContracts8298([]types.Address{target})
		} else {
			evm.readOnly = true
		}
		c := NewContract(target, target, nil, 100000)
		c.Code = code
		if _, err := evm.Run(c, nil); !errors.Is(err, ErrWriteProtection) || db.AddressInAccessList(source) || tracker.Build(1).Len() != 0 {
			t.Fatal("early guard accessed source", err)
		}
	}
	for _, immediate := range []OpCode{DUPN, SWAPN, EXCHANGE} {
		if HasOpcode8298([]byte{byte(immediate), byte(SETCODEFROM)}, SETCODEFROM) || HasOpcode8298([]byte{byte(immediate), byte(DELEGATECALL)}, DELEGATECALL) {
			t.Fatal("8024 immediate treated as instruction")
		}
	}
	if HasOpcode8298([]byte{byte(PUSH1), 0xa7, byte(STOP)}, SETCODEFROM) || !HasOpcode8298([]byte{byte(PUSH1), 1, 0xa7}, SETCODEFROM) {
		t.Fatal("PUSH immediate misclassified")
	}
}

func TestSetCodeFrom8298LoadedCodeReentryAndPersistentAvailability(t *testing.T) {
	disk := rawdb.NewMemoryDB()
	db := state.NewTrieStateDB(disk)
	template, target := types.Address{19: 0x81}, types.Address{19: 0x82}
	// The adopted runtime returns 0x42. Reentry immediately sees this runtime,
	// while the enclosing CODESIZE still reports the loaded migration program.
	db.SetCode(template, []byte{0x60, 0x42, 0x5f, 0x52, 0x60, 32, 0x5f, 0xf3})
	code := append([]byte{byte(PUSH20)}, template[:]...)
	code = append(code, byte(SETCODEFROM), byte(POP), byte(CODESIZE), byte(PUSH0), byte(MSTORE), byte(ADDRESS), byte(EXTCODESIZE), byte(PUSH1), 32, byte(MSTORE))
	code = append(code, byte(PUSH1), 32, byte(PUSH1), 64, byte(PUSH0), byte(PUSH0), byte(PUSH0), byte(ADDRESS), byte(PUSH2), 0xff, 0xff, byte(CALL), byte(POP), byte(PUSH1), 96, byte(PUSH0), byte(RETURN))
	db.SetCode(target, code)
	evm := NewEVMWithState(BlockContext{}, TxContext{}, Config{}, db)
	rules := ForkRules{IsAmsterdam: true, IsEIP8298: true, IsPrague: true, IsCancun: true}
	evm.SetJumpTable(SelectJumpTable(rules))
	evm.SetForkRules(rules)
	out, _, err := evm.Call(types.Address{19: 0x83}, target, nil, 1_000_000, new(big.Int))
	if err != nil || len(out) != 96 || new(big.Int).SetBytes(out[:32]).Uint64() != uint64(len(code)) || new(big.Int).SetBytes(out[32:64]).Uint64() != 8 || new(big.Int).SetBytes(out[64:]).Uint64() != 0x42 {
		t.Fatal("loaded code/introspection/reentry divergence", out, err)
	}
	db.SetCode(template, []byte{0xfe})
	root, err := db.Commit()
	if err != nil {
		t.Fatal(err)
	}
	reopened := state.NewTrieStateDB(disk)
	if root == types.EmptyRootHash {
		t.Fatal("adoption committed an empty trie")
	}
	if !bytes.Equal(reopened.GetCode(target), []byte{0x60, 0x42, 0x5f, 0x52, 0x60, 32, 0x5f, 0xf3}) || !bytes.Equal(reopened.GetCode(template), []byte{0xfe}) {
		t.Fatal("source replacement/restart lost adopted bytecode")
	}
}
