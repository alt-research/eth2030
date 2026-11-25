package gaspool

import "errors"

// ErrGasPoolExhausted is returned when the block gas pool has insufficient gas.
var ErrGasPoolExhausted = errors.New("gas pool exhausted")

// GasPool tracks the amount of gas available during block execution.
type GasPool struct {
	execution   uint64
	state       uint64
	initialized bool
}

// AddGas adds gas to the pool.
func (gp *GasPool) AddGas(amount uint64) *GasPool {
	gp.execution += amount
	if !gp.initialized {
		gp.state = amount
		gp.initialized = true
	}
	return gp
}

// SubGas subtracts gas from the pool, returning an error if insufficient.
func (gp *GasPool) SubGas(amount uint64) error {
	if gp.execution < amount {
		return ErrGasPoolExhausted
	}
	gp.execution -= amount
	return nil
}

// Gas returns the amount of gas remaining in the pool.
func (gp *GasPool) Gas() uint64 {
	return gp.execution
}

// StateGas returns the independent EIP-8037 state pool remaining in the block.
func (gp *GasPool) StateGas() uint64 { return gp.state }

// FitsFrame checks both reservations without adding the two dimensions.
func (gp *GasPool) FitsFrame(execution, state uint64) bool {
	return execution <= gp.execution && state <= gp.state
}

func (gp *GasPool) ReserveFrame(execution, state uint64) error {
	if !gp.FitsFrame(execution, state) {
		return ErrGasPoolExhausted
	}
	gp.execution -= execution
	gp.state -= state
	return nil
}

func (gp *GasPool) RefundFrame(execution, state uint64) {
	gp.execution += execution
	gp.state += state
}
