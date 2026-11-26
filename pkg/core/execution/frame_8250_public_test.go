package execution

import (
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	"math"
	"testing"
)

func TestFrame8250PublicExecutionSchemaBoundary(t *testing.T) {
	for _, domain := range []string{"legacy", "keyed", "single_use"} {
		t.Run(domain, func(t *testing.T) {
			base, db, cfg, header := frameExecutionFixture(t)
			activation := header.Time
			cfg.EIP8250Time = &activation
			db.SetNonce(types.NonceManager8250, 1)
			db.SetCode(types.NonceManager8250, types.NonceManagerCode8250)
			profile := delta8250(base)
			if domain != "legacy" {
				profile = delta8250(base, 42)
				profile.Frames[0].Limits.State = 97920
			}
			if domain == "single_use" {
				profile.NonceSeq = math.MaxUint64
			}
			signDelta8250(t, profile)
			tx := types.NewFrameTransaction8250(profile)
			before := types.NewBlock(header, nil).Header()
			before.Time--
			root := db.GetRoot()
			if _, _, err := ReplayFrame8141(cfg, nil, db, before, tx, nil, vm.Config{}); err == nil {
				t.Fatal("pre-fork keyed envelope executed")
			}
			if db.GetRoot() != root {
				t.Fatal("schema rejection changed state")
			}
			old := signFrameExecution(t, base, false)
			if _, _, err := applyFrame8141(cfg, nil, db, header, old, new(gaspool.GasPool).AddGas(header.GasLimit), nil); err == nil {
				t.Fatal("post-fork legacy envelope executed")
			}
			receipt, _, err := applyFrame8141(cfg, nil, db, header, tx, new(gaspool.GasPool).AddGas(header.GasLimit), nil)
			if err != nil {
				t.Fatal(err)
			}
			hash, _ := profile.Hash()
			if receipt.TxHash != hash || receipt.TxHash != tx.Hash() {
				t.Fatal("native public hash differs")
			}
			wantNonce := uint64(0)
			if domain == "legacy" {
				wantNonce = 1
			}
			if db.GetNonce(base.Sender) != wantNonce {
				t.Fatal("wrong nonce domain consumed")
			}
			if _, _, err := applyFrame8141(cfg, nil, db, header, tx, new(gaspool.GasPool).AddGas(header.GasLimit), nil); err == nil {
				t.Fatal("public replay accepted")
			}
		})
	}
}
