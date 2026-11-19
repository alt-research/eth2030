package payload

import (
	"github.com/eth2030/eth2030/core/eips"
	"github.com/eth2030/eth2030/core/state"
)

// Construct the owned Amsterdam fixture before funds and roots are assigned.
func newBuilderRequestTestState() *state.MemoryStateDB {
	db := state.NewMemoryStateDB()
	eips.ApplyBuilderRequestGenesis(db)
	return db
}
