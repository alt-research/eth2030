package e2e_test

import (
	"github.com/eth2030/eth2030/core/eips"
	"github.com/eth2030/eth2030/core/state"
)

func newBuilderRequestTestState() *state.MemoryStateDB {
	db := state.NewMemoryStateDB()
	eips.ApplyBuilderRequestGenesis(db)
	return db
}
