package gas

import (
	"math"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
)

func TestAmsterdamIntrinsicStateIndependentPublishedCases(t *testing.T) {
	from, to := types.Address{19: 1}, types.Address{19: 2}
	access := types.AccessList{{Address: to, StorageKeys: []types.Hash{{1}}}}
	for _, test := range []struct {
		name       string
		to         *types.Address
		value      int64
		data       []byte
		access     types.AccessList
		auth       uint64
		gas, floor uint64
	}{
		{name: "self", to: &from, value: 1, gas: 12000, floor: 12000}, {name: "other_zero", to: &to, gas: 15000, floor: 15000}, {name: "other_value", to: &to, value: 1, gas: 21000, floor: 21000}, {name: "create", gas: 24000, floor: 24000}, {name: "create_value", value: 1, gas: 24000, floor: 24000}, {name: "self_data", to: &from, data: []byte{0, 1}, gas: 12020, floor: 12128}, {name: "access_data", to: &to, access: access, gas: 23228, floor: 18328}, {name: "two_authorizations", to: &to, auth: 2, gas: 30632, floor: 15000}, {name: "create_word_cost", data: make([]byte, 33), gas: 24136, floor: 26112},
	} {
		t.Run(test.name, func(t *testing.T) {
			actual, err := CalculateAmsterdamIntrinsic(from, test.to, big.NewInt(test.value), test.data, test.access, test.auth)
			if err != nil || actual.Gas != test.gas || actual.Floor != test.floor {
				t.Fatalf("intrinsic/floor=%+v err=%v", actual, err)
			}
		})
	}
	if _, err := CalculateAmsterdamIntrinsic(from, &to, nil, nil, nil, math.MaxUint64); err == nil {
		t.Fatal("authorization count overflow accepted")
	}
}
func TestAmsterdamDistinctExecutionAndTotalCaps(t *testing.T) {
	g := AmsterdamIntrinsic{Gas: 15000, Floor: 15000}
	for _, limit := range []uint64{15000, vm.AmsterdamMaxExecutionGas, vm.AmsterdamMaxExecutionGas + 1, vm.AmsterdamMaxTotalGas} {
		if err := g.Validate(limit); err != nil {
			t.Fatal(err)
		}
		execution, reservoir, err := g.ExecutionBudget(limit)
		if err != nil {
			t.Fatal(err)
		}
		if execution+reservoir+g.Gas != limit || execution+g.Gas > vm.AmsterdamMaxExecutionGas {
			t.Fatal("transaction split lost gas or exceeded execution cap")
		}
	}
	if g.Validate(14999) == nil || g.Validate(vm.AmsterdamMaxTotalGas+1) == nil {
		t.Fatal("invalid lower/upper boundary accepted")
	}
	if (AmsterdamIntrinsic{Gas: 15000, Floor: vm.AmsterdamMaxExecutionGas + 1}).Validate(vm.AmsterdamMaxTotalGas) == nil {
		t.Fatal("floor exceeding execution cap accepted")
	}
}
