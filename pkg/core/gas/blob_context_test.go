package gas

import (
	"github.com/eth2030/eth2030/core/types"
	"math"
	"testing"
)

func TestBlobHeaderArithmeticRejectsOverflowAndInvalidUnits(t *testing.T) {
	max, used := uint64(math.MaxUint64), uint64(GasPerBlob)
	parent := &types.Header{ExcessBlobGas: &max, BlobGasUsed: &used}
	if _, err := CalcExcessBlobGasWithRules(parent, PragueBlobSchedule, true); err == nil {
		t.Fatal("excess overflow silently wrapped")
	}
	for _, invalid := range []BlobSchedule{{Target: 1, Max: 0, UpdateFraction: 1}, {Target: 2, Max: 1, UpdateFraction: 1}, {Target: 1, Max: math.MaxUint64, UpdateFraction: 1}, {Target: 1, Max: 2, UpdateFraction: 0}} {
		if _, err := CalcExcessBlobGasWithRules(&types.Header{}, invalid, true); err == nil {
			t.Fatalf("invalid schedule admitted: %+v", invalid)
		}
	}
	if _, err := CalcExcessBlobGasWithRules(nil, PragueBlobSchedule, true); err == nil {
		t.Fatal("missing parent admitted")
	}
}
