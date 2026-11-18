package execution

import (
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/eips"
	"github.com/eth2030/eth2030/core/types"
	"testing"
)

func TestGlamsterdamDoesNotInjectFactoryAtFork(t *testing.T) {
	db := newBuilderRequestTestState()
	p := NewStateProcessor(config.TestConfigGlamsterdan)
	if _, err := p.Process(types.NewBlock(newTestHeader(), &types.Body{}), db); err != nil {
		t.Fatal(err)
	}
	if db.Exist(eips.FactoryAddress) || db.Exist(types.HexToAddress("0x12")) {
		t.Fatal("fork processing injected factory state")
	}
}
