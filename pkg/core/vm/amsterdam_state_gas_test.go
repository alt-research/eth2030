package vm

import "testing"

func TestAmsterdamStateGasChargeRefillAndAtomicFailure(t *testing.T) {
	c := &Contract{Gas: 100, StateGasReservoir: 50, StateGasBaseline: 50}
	if !c.ChargeStateGas(70) || c.Gas != 80 || c.StateGasReservoir != 0 || c.StateGasFromGasLeft != 20 || c.StateGasUsed != 70 {
		t.Fatal("reservoir was not drained before execution borrowing")
	}
	c.RefillStateGas(30)
	if c.Gas != 100 || c.StateGasReservoir != 10 || c.StateGasFromGasLeft != 0 || c.StateGasUsed != 40 {
		t.Fatal("refill did not repay execution before reservoir")
	}
	before := *c
	if c.ChargeStateGas(111) || c.Gas != before.Gas || c.StateGasReservoir != before.StateGasReservoir || c.StateGasUsed != before.StateGasUsed {
		t.Fatal("failed charge changed either budget")
	}
}
func TestAmsterdamCrossFrameRefillSuccessRevertAndExceptional(t *testing.T) {
	for _, exit := range []string{"success", "revert", "exceptional"} {
		t.Run(exit, func(t *testing.T) {
			parent := &Contract{Gas: 500}
			if !parent.ChargeStateGas(100) {
				t.Fatal("parent state allocation failed")
			}
			parent.Gas -= 100
			child := &Contract{Gas: 100}
			child.RefillStateGas(100) // Clear the parent's transaction-new slot.
			if exit != "success" {
				child.RollbackStateGas(exit == "exceptional")
			}
			parent.MergeStateGas(child)
			wantGas, wantState := uint64(500), int64(0)
			if exit == "revert" {
				wantGas, wantState = 400, 100
			}
			if exit == "exceptional" {
				wantGas, wantState = 300, 100
			}
			if parent.Gas != wantGas || parent.StateGasUsed != wantState || parent.StateGasReservoir != 0 {
				t.Fatalf("%s merge gas=%d state=%d reservoir=%d", exit, parent.Gas, parent.StateGasUsed, parent.StateGasReservoir)
			}
		})
	}
}
func TestAmsterdamRollbackKeepsCommittedAuthorizationState(t *testing.T) {
	c := &Contract{Gas: 1000}
	if !c.ChargeStateGas(50) {
		t.Fatal("authorization charge failed")
	}
	c.CommitStateGas()
	if !c.ChargeStateGas(100) {
		t.Fatal("target charge failed")
	}
	c.RollbackStateGas(false)
	if c.Gas != 950 || c.StateGasUsed != 50 || c.StateGasCommitted != 50 {
		t.Fatal("frame revert undid persistent authorization state")
	}
	c.RestorePreExecutionStateGas(0)
	if c.Gas != 1000 || c.StateGasUsed != 0 || c.StateGasCommitted != 0 {
		t.Fatal("pre-execution failure kept rolled-back authorizations")
	}
}
func TestAmsterdamFundedReservoirSurvivesExceptionalHalt(t *testing.T) {
	c := &Contract{Gas: 100, StateGasReservoir: 60, StateGasBaseline: 60}
	if !c.ChargeStateGas(100) {
		t.Fatal("charge failed")
	}
	c.RollbackStateGas(true)
	if c.Gas != 0 || c.StateGasReservoir != 60 || c.StateGasUsed != 0 {
		t.Fatal("exceptional halt consumed state that was rolled back")
	}
}
