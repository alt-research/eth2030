package focil

import (
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	"github.com/eth2030/eth2030/internal/testutil/proof8504"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

func TestGroupELatest7805IndependentOrdinaryGasReservations(t *testing.T) {
	raw, cfg, h, db, sender := profile1Fixture(t, types.DynamicFeeTxType, big.NewInt(1337))
	var original gethtypes.Transaction
	if err := original.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	key, err := gethcrypto.HexToECDSA("0000000000000000000000000000000000000000000000000000000000000001")
	if err != nil {
		t.Fatal(err)
	}
	signed, err := gethtypes.SignTx(gethtypes.NewTx(&gethtypes.DynamicFeeTx{ChainID: cfg.ChainID, Gas: 25000000, GasFeeCap: big.NewInt(100), GasTipCap: big.NewInt(1), To: original.To(), Value: big.NewInt(1)}), gethtypes.LatestSignerForChainID(cfg.ChainID), key)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = signed.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	h.GasUsed = 8000000
	for _, test := range []struct {
		name                     string
		executionUsed, stateUsed uint64
		enforce                  bool
	}{
		{"header_total_is_not_execution_capacity", 8000000, 0, true},
		{"exact_both_dimensions", h.GasLimit - vm.AmsterdamMaxExecutionGas, h.GasLimit - 25000000, true},
		{"execution_one_over", h.GasLimit - vm.AmsterdamMaxExecutionGas + 1, 0, false},
		{"state_one_over", 0, h.GasLimit - 25000000 + 1, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			pool := new(gaspool.GasPool).AddGas(h.GasLimit)
			if err := pool.ReserveFrame(test.executionUsed, test.stateUsed); err != nil {
				t.Fatal(err)
			}
			executionLeft, stateLeft := pool.Gas(), pool.StateGas()
			for i := 0; i < 3; i++ {
				result, err := EvaluateProfile1WithCapacity(raw, cfg, h, profile1AccountOnly{db}, pool)
				if err != nil || (result.Kind == Profile1Enforceable) != test.enforce {
					t.Fatal(result, err)
				}
				if result.ExecutionReservation != vm.AmsterdamMaxExecutionGas || result.StateReservation != 25000000 {
					t.Fatal("wrong independent reservations", result)
				}
			}
			if pool.Gas() != executionLeft || pool.StateGas() != stateLeft || db.GetNonce(sender) != 0 || db.AddressInAccessList(sender) {
				t.Fatal("independent candidate checks consumed state or gas")
			}
		})
	}
}

type profile1AccountOnly struct{ state.StateDB }

func (s profile1AccountOnly) GetState(types.Address, types.Hash) types.Hash {
	panic("Profile 1 read EVM storage")
}
func (s profile1AccountOnly) Dup() state.StateDB { panic("Profile 1 attempted execution clone") }
func profile1Fixture(t *testing.T, kind uint8, chain *big.Int) ([]byte, *config.ChainConfig, *types.Header, *state.MemoryStateDB, types.Address) {
	t.Helper()
	key, e := gethcrypto.HexToECDSA("0000000000000000000000000000000000000000000000000000000000000001")
	if e != nil {
		t.Fatal(e)
	}
	to := gethcrypto.PubkeyToAddress(key.PublicKey)
	to[0] ^= 0xff
	var data gethtypes.TxData
	switch kind {
	case types.LegacyTxType:
		data = &gethtypes.LegacyTx{Gas: 50000, GasPrice: big.NewInt(100), To: &to, Value: big.NewInt(1)}
	case types.AccessListTxType:
		data = &gethtypes.AccessListTx{ChainID: chain, Gas: 50000, GasPrice: big.NewInt(100), To: &to, Value: big.NewInt(1)}
	case types.SetCodeTxType:
		auth, err := gethtypes.SignSetCode(key, gethtypes.SetCodeAuthorization{ChainID: *uint256.MustFromBig(chain), Address: to, Nonce: 1})
		if err != nil {
			t.Fatal(err)
		}
		data = &gethtypes.SetCodeTx{ChainID: uint256.MustFromBig(chain), Gas: 100000, GasFeeCap: uint256.NewInt(100), GasTipCap: uint256.NewInt(1), To: to, Value: uint256.NewInt(1), AuthList: []gethtypes.SetCodeAuthorization{auth}}
	default:
		data = &gethtypes.DynamicFeeTx{ChainID: chain, Gas: 50000, GasFeeCap: big.NewInt(100), GasTipCap: big.NewInt(1), To: &to, Value: big.NewInt(1)}
	}
	signed, e := gethtypes.SignTx(gethtypes.NewTx(data), gethtypes.LatestSignerForChainID(chain), key)
	if e != nil {
		t.Fatal(e)
	}
	raw, e := signed.MarshalBinary()
	if e != nil {
		t.Fatal(e)
	}
	cfg := *config.TestConfigHogota
	cfg.ChainID = new(big.Int).Set(chain)
	header := &types.Header{Number: big.NewInt(1), Time: 1, GasLimit: 30000000, BaseFee: big.NewInt(10)}
	db := state.NewMemoryStateDB()
	sender := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
	db.AddBalance(sender, new(big.Int).Lsh(big.NewInt(1), 200))
	return raw, &cfg, header, db, sender
}
func TestGroupEProfile1AuthenticatedAccountsWithoutExecution(t *testing.T) {
	for _, kind := range []uint8{types.LegacyTxType, types.AccessListTxType, types.DynamicFeeTxType, types.SetCodeTxType} {
		raw, cfg, h, db, sender := profile1Fixture(t, kind, big.NewInt(1337))
		result, e := EvaluateProfile1(raw, cfg, h, profile1AccountOnly{db})
		if e != nil || result.Kind != Profile1Enforceable || result.Sender != sender {
			t.Fatal(kind, result, e)
		}
		if db.GetNonce(sender) != 0 {
			t.Fatal("eligibility changed nonce")
		}
	}
	wide := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1337))
	raw, cfg, h, db, _ := profile1Fixture(t, types.DynamicFeeTxType, wide)
	if r, e := EvaluateProfile1(raw, cfg, h, profile1AccountOnly{db}); e != nil || r.Kind != Profile1Enforceable {
		t.Fatal("uint256 chain/balance truncated", r, e)
	}
}
func TestGroupEProfile1InvalidSignatureFeeCodeNonceAndResources(t *testing.T) {
	for _, kind := range []string{"bad-signature", "wrong-chain", "nonce", "balance", "ordinary-code", "bad-delegation", "fee", "gas", "malformed"} {
		t.Run(kind, func(t *testing.T) {
			raw, cfg, h, db, sender := profile1Fixture(t, types.DynamicFeeTxType, big.NewInt(1337))
			switch kind {
			case "bad-signature":
				tx, _ := types.DecodeTxRLP(raw)
				inner := tx.Inner().(*types.DynamicFeeTx)
				inner.R = new(big.Int)
				inner.S = new(big.Int)
				raw, _ = types.NewTransaction(inner).EncodeRLP()
			case "wrong-chain":
				cfg.ChainID = big.NewInt(2)
			case "nonce":
				db.SetNonce(sender, 1)
			case "balance":
				db.SubBalance(sender, db.GetBalance(sender))
			case "ordinary-code":
				db.SetCode(sender, []byte{0})
			case "bad-delegation":
				db.SetCode(sender, []byte{0xef, 1, 0})
			case "fee":
				h.BaseFee = big.NewInt(101)
			case "gas":
				h.GasUsed = h.GasLimit - 49999
			case "malformed":
				raw = []byte{0xff}
			}
			result, e := EvaluateProfile1(raw, cfg, h, profile1AccountOnly{db})
			if e != nil || result.Kind != Profile1Exempt {
				t.Fatal(result, e)
			}
		})
	}
	raw, cfg, h, db, sender := profile1Fixture(t, types.DynamicFeeTxType, big.NewInt(1337))
	delegation := append([]byte{0xef, 1, 0}, make([]byte, 20)...)
	db.SetCode(sender, delegation)
	if r, e := EvaluateProfile1(raw, cfg, h, profile1AccountOnly{db}); e != nil || r.Kind != Profile1Enforceable {
		t.Fatal("valid 7702 delegation excluded", r, e)
	}
}
func TestGroupEProfile1OutsideProfilesAndUnavailableContext(t *testing.T) {
	tx := proof8504.Block().Transactions()[0]
	raw, e := tx.EncodeRLP()
	if e != nil {
		t.Fatal(e)
	}
	if r, e := EvaluateProfile1(raw, nil, nil, nil); e != nil || r.Kind != Profile1Outside {
		t.Fatal("Frame incorrectly promoted to Profile 1", r, e)
	}
	raw, cfg, h, _, _ := profile1Fixture(t, types.LegacyTxType, big.NewInt(1337))
	if _, e := EvaluateProfile1(raw, cfg, h, nil); e == nil {
		t.Fatal("missing state excused omission")
	}
}
