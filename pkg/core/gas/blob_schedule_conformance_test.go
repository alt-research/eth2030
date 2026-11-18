package gas

import (
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/types"
	geth4844 "github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

func TestCanonicalBlobScheduleMatchesIndependentClient(t *testing.T) {
	prague, osaka, bpo1, bpo2 := uint64(10), uint64(30), uint64(20), uint64(40)
	native := *config.TestConfig
	native.PragueTime, native.OsakaTime, native.BPO1Time, native.BPO2Time = &prague, &osaka, &bpo1, &bpo2
	native.AmsterdamTime = nil
	reference := *params.AllDevChainProtocolChanges
	reference.PragueTime, reference.OsakaTime, reference.BPO1Time, reference.BPO2Time = &prague, &osaka, &bpo1, &bpo2
	reference.AmsterdamTime, reference.BogotaTime = nil, nil
	reference.BlobScheduleConfig = &params.BlobScheduleConfig{Cancun: params.DefaultCancunBlobConfig, Prague: params.DefaultPragueBlobConfig, BPO1: params.DefaultBPO1BlobConfig, BPO2: params.DefaultBPO2BlobConfig}
	for _, timestamp := range []uint64{9, 10, 19, 20, 29, 30, 39, 40} {
		sched := GetBlobSchedule(&native, timestamp)
		entry := GetBlobScheduleEntry(&native, timestamp)
		if int(sched.Max) != geth4844.MaxBlobsPerBlock(&reference, timestamp) || entry.Max != sched.Max || entry.Target != sched.Target || entry.BaseFeeUpdateFraction != sched.UpdateFraction {
			t.Errorf("time %d inconsistent schedule consumers: main=%+v entry=%+v", timestamp, sched, entry)
		}
		for _, excess := range []uint64{0, 2_000_000, 14_000_000} {
			h := &gethtypes.Header{Time: timestamp, ExcessBlobGas: &excess}
			want := geth4844.CalcBlobFee(&reference, h)
			got := CalcBlobBaseFeeV2WithFraction(excess, big.NewInt(1_000_000_000), sched.UpdateFraction)
			if got.Cmp(want) != 0 {
				t.Errorf("time %d excess %d blob fee %v want %v", timestamp, excess, got, want)
			}
		}
		parentUsed := uint64(6) * GasPerBlob
		parentExcess := uint64(2_000_000)
		parent := &types.Header{Number: big.NewInt(1), Time: timestamp - 1, BaseFee: big.NewInt(1000), BlobGasUsed: &parentUsed, ExcessBlobGas: &parentExcess}
		refParent := &gethtypes.Header{Number: big.NewInt(1), Time: timestamp - 1, BaseFee: big.NewInt(1000), BlobGasUsed: &parentUsed, ExcessBlobGas: &parentExcess}
		want := geth4844.CalcExcessBlobGas(&reference, refParent, timestamp)
		if got := CalcExcessBlobGasV2ForHeader(parent, &native, timestamp); got != want {
			t.Errorf("time %d excess transition %d want %d", timestamp, got, want)
		}
	}
}
