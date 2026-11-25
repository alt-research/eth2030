package types

import "testing"

func TestFrame8141ReceiptRejectsImpossibleFrameEffects(t *testing.T) {
	for _, result := range []FrameResult8141{
		{Status: 0, GasUsed: FrameGasLimits{State: 1}},
		{Status: 0, Logs: []*Log{{}}},
		{Status: 2, GasUsed: FrameGasLimits{Execution: 1}},
		{Status: 2, Logs: []*Log{{}}},
		{Status: 3},
	} {
		receipt := &Receipt{Type: FrameTxType, FrameResults8141: []FrameResult8141{result}}
		if _, err := receipt.EncodeRLP(); err == nil {
			t.Fatalf("encoded impossible frame effects: %+v", result)
		}
		frame := frame8141ResultRLP{Status: result.Status, GasUsed: result.GasUsed}
		for _, log := range result.Logs {
			frame.Logs = append(frame.Logs, frame8141LogRLP{Address: log.Address})
		}
		if err := validateFrame8141Receipt(frame); err == nil {
			t.Fatal("decoder validation accepted impossible effects")
		}
	}
}
