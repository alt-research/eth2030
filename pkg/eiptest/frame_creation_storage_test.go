package eiptest

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func TestFrameCSTFCallUsesNativeCreationRules(t *testing.T) {
	s := New(t)
	for _, timestamp := range []uint64{99, 100} {
		t.Run(new(big.Int).SetUint64(timestamp).String(), func(t *testing.T) {
			cfg := *config.TestConfig
			activation := uint64(100)
			cfg.HogotaTime = &activation
			probe := &STF{s: &Sim{bc: s.bc, cfg: &cfg}}
			factory := types.Address{19: 0x90}
			target := types.Address(gethcrypto.CreateAddress([20]byte(factory), 0))
			db := state.NewMemoryStateDB()
			db.SetCode(factory, []byte{0x5f, 0x5f, 0x5f, 0xf0, 0x5f, 0x52, 0x60, 32, 0x5f, 0xf3})
			db.SetState(target, types.Hash{0: 0xff}, types.Hash{31: 1})
			header := &types.Header{Time: timestamp, Number: big.NewInt(1), GasLimit: 1000000, BaseFee: big.NewInt(1)}
			root := db.GetRoot()
			ret, err := probe.evmCall(db.Dup(), types.Address{19: 0x99}, factory, nil, header)
			if err != nil || len(ret) != 32 || bytes.Equal(ret, make([]byte, 32)) != (timestamp == 100) {
				t.Fatalf("probe fork differs from native creation: return=%x err=%v", ret, err)
			}
			if db.GetRoot() != root {
				t.Fatal("disposable probe changed live state")
			}
		})
	}
}
