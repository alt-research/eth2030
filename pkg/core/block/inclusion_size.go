package block

import (
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/frameproof"
	"github.com/eth2030/eth2030/core/gas"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/types"
)

type builderRLPBudget struct {
	sizer        *types.BlockAppendSizer
	header       *types.Header
	witnesses    [][]byte
	calldataUsed uint64
}

func newBuilderRLPBudget(cfg *config.ChainConfig, header *types.Header, attrs *BuildBlockAttributes) (*builderRLPBudget, error) {
	if cfg == nil || !cfg.IsOsaka(header.Time) {
		return nil, nil
	}
	h := header.Copy()
	zero := types.Hash{}
	withdrawals := attrs.Withdrawals
	if cfg.IsShanghai(h.Time) {
		if withdrawals == nil {
			withdrawals = []*types.Withdrawal{}
		}
		h.WithdrawalsHash = &zero
	}
	if cfg.IsPrague(h.Time) {
		h.RequestsHash = &zero
	}
	if cfg.IsAmsterdam(h.Time) {
		h.BlockAccessListHash = &zero
	}
	body := &types.Body{Withdrawals: withdrawals}
	if cfg.IsEIP8504(h.Time) {
		h.DependencyWitnessesHash = &zero
		body.DependencyWitnesses = [][]byte{}
	}
	if cfg.IsGlamsterdan(h.Time) {
		limits := [3]uint64{h.GasLimit, 0, gas.CalcCalldataGasLimit(h.GasLimit)}
		usage := [3]uint64{}
		excess := [3]uint64{}
		if h.CalldataExcessGas != nil {
			excess[2] = *h.CalldataExcessGas
		}
		h.GasLimitVec = &limits
		h.GasUsedVec = &usage
		h.ExcessGasVec = &excess
	}
	sizer, err := types.NewBlockAppendSizer(types.NewBlock(h, body))
	if err != nil {
		return nil, err
	}
	return &builderRLPBudget{sizer: sizer, header: h, witnesses: body.DependencyWitnesses}, nil
}

func (b *builderRLPBudget) append(txs []*types.Transaction, tx *types.Transaction, totalGas, blobGas uint64) (bool, error) {
	if b == nil {
		return true, nil
	}
	h := b.header.Copy()
	h.GasUsed = totalGas
	if h.BlobGasUsed != nil {
		h.BlobGasUsed = &blobGas
	}
	calldata := b.calldataUsed + tx.CalldataGas()
	if h.CalldataGasUsed != nil {
		h.CalldataGasUsed = &calldata
	}
	if h.GasUsedVec != nil {
		usage := [3]uint64{totalGas, 0, calldata}
		h.GasUsedVec = &usage
	}
	witnesses := b.witnesses
	if tx.HasDependencies8504() {
		var err error
		all := append(append([]*types.Transaction(nil), txs...), tx)
		witnesses, err = frameproof.Collect(all)
		if err != nil {
			return false, err
		}
	}
	size, err := b.sizer.SizeWithHeader(tx, h, witnesses)
	if err != nil || size > MaxRLPBlockSize {
		return false, err
	}
	if err := b.sizer.CommitAppend(tx); err != nil {
		return false, err
	}
	b.calldataUsed = calldata
	b.witnesses = witnesses
	return true, nil
}

// appendWithPools projects the same gas_used integer as final block accounting,
// including state-dominated prefixes and refunds. Failed appends commit nothing.
func (b *builderRLPBudget) appendWithPools(txs []*types.Transaction, tx *types.Transaction, gp *gaspool.GasPool, gasLimit, blobGas uint64) (bool, error) {
	if b == nil {
		return true, nil
	}
	return b.append(txs, tx, gp.BlockGasUsed(gasLimit), blobGas)
}
