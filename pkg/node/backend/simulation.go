package backend

import (
	"bytes"
	"fmt"
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/execution"
	coregas "github.com/eth2030/eth2030/core/gas"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	rpcbackend "github.com/eth2030/eth2030/rpc/backend"
	rpctypes "github.com/eth2030/eth2030/rpc/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"math/big"
)

// parseSimulationMessage keeps complete call input for method-specific execution.
func parseSimulationMessage(args rpctypes.CallArgs, header *types.Header, base state.StateDB, cfg *config.ChainConfig) (*config.Message, error) {
	var err error
	invalid := func(field string, err error) error {
		return fmt.Errorf("%w: %s: %v", rpcbackend.ErrInvalidCallArgs, field, err)
	}
	address := func(value string) (types.Address, error) {
		raw, err := hexutil.Decode(value)
		if err != nil {
			return types.Address{}, err
		}
		if len(raw) != types.AddressLength {
			return types.Address{}, fmt.Errorf("address must contain 20 bytes")
		}
		return types.BytesToAddress(raw), nil
	}

	if cfg != nil {
		rules := cfg.Rules(header.Number, cfg.IsMerge(), header.Time)
		if args.AccessList != nil && !rules.IsBerlin {
			return nil, invalid("accessList", fmt.Errorf("requires Berlin"))
		}
		if (args.MaxFeePerGas != nil || args.MaxPriorityFeePerGas != nil) && !rules.IsLondon {
			return nil, invalid("dynamic fees", fmt.Errorf("requires London"))
		}
		if args.AuthorizationList != nil && !rules.IsPrague {
			return nil, invalid("authorizationList", fmt.Errorf("requires Prague"))
		}
		if (args.BlobVersionedHashes != nil || args.MaxFeePerBlobGas != nil) && !rules.IsCancun {
			return nil, invalid("blob fields", fmt.Errorf("requires Cancun"))
		}
	}
	if args.Type != nil {
		kind, err := hexutil.DecodeUint64(*args.Type)
		if err != nil || kind > 255 {
			return nil, invalid("type", fmt.Errorf("must be a byte"))
		}
	}
	from := types.Address{}
	if args.From != nil {
		from, err = address(*args.From)
		if err != nil {
			return nil, invalid("from", err)
		}
	}
	var to *types.Address
	if args.To != nil {
		a, err := address(*args.To)
		if err != nil {
			return nil, invalid("to", err)
		}
		to = &a
	}
	limit := min(header.GasLimit, uint64(50_000_000))
	if args.Gas != nil {
		limit, err = hexutil.DecodeUint64(*args.Gas)
		if err != nil {
			return nil, invalid("gas", err)
		}
	}
	price, value := new(big.Int), new(big.Int)
	if args.GasPrice != nil {
		price, err = hexutil.DecodeBig(*args.GasPrice)
		if err != nil {
			return nil, invalid("gasPrice", err)
		}
	}
	if args.Value != nil {
		value, err = hexutil.DecodeBig(*args.Value)
		if err != nil {
			return nil, invalid("value", err)
		}
	}
	nonce := base.GetNonce(from)
	if args.Nonce != nil {
		nonce, err = hexutil.DecodeUint64(*args.Nonce)
		if err != nil {
			return nil, invalid("nonce", err)
		}
	}
	input := args.Input
	if input == nil {
		input = args.Data
	}
	var data []byte
	if args.Input != nil && args.Data != nil {
		left, e1 := hexutil.Decode(*args.Input)
		right, e2 := hexutil.Decode(*args.Data)
		if e1 != nil || e2 != nil || !bytes.Equal(left, right) {
			return nil, invalid("input/data", fmt.Errorf("fields must contain identical bytes"))
		}
	}
	if input != nil {
		data, err = hexutil.Decode(*input)
		if err != nil {
			return nil, invalid("input", err)
		}
	}
	list := make(types.AccessList, 0, len(args.AccessList))
	for _, item := range args.AccessList {
		a, err := address(item.Address)
		if err != nil {
			return nil, invalid("accessList address", err)
		}
		tuple := types.AccessTuple{Address: a, StorageKeys: make([]types.Hash, 0, len(item.StorageKeys))}
		for _, key := range item.StorageKeys {
			raw, err := hexutil.Decode(key)
			if err != nil {
				return nil, invalid("storage key", err)
			}
			if len(raw) != types.HashLength {
				return nil, invalid("storage key", fmt.Errorf("must contain 32 bytes"))
			}
			tuple.StorageKeys = append(tuple.StorageKeys, types.BytesToHash(raw))
		}
		list = append(list, tuple)
	}

	if price.BitLen() > 256 || value.BitLen() > 256 {
		return nil, invalid("amount", fmt.Errorf("quantity exceeds uint256"))
	}
	msg := &config.Message{From: from, To: to, Nonce: nonce, GasLimit: limit, GasPrice: price, Value: value, Data: data, AccessList: list}
	if args.GasPrice != nil && (args.MaxFeePerGas != nil || args.MaxPriorityFeePerGas != nil) {
		return nil, invalid("fees", fmt.Errorf("gasPrice and dynamic fees are mutually exclusive"))
	}
	if args.MaxFeePerGas != nil || args.MaxPriorityFeePerGas != nil {
		cap, tip := new(big.Int), new(big.Int)
		if args.MaxFeePerGas != nil {
			cap, err = hexutil.DecodeBig(*args.MaxFeePerGas)
			if err != nil {
				return nil, invalid("maxFeePerGas", err)
			}
		}
		if args.MaxPriorityFeePerGas != nil {
			tip, err = hexutil.DecodeBig(*args.MaxPriorityFeePerGas)
			if err != nil {
				return nil, invalid("maxPriorityFeePerGas", err)
			}
		}
		if cap.BitLen() > 256 || tip.BitLen() > 256 || tip.Cmp(cap) > 0 {
			return nil, invalid("fees", fmt.Errorf("invalid dynamic fee caps"))
		}
		msg.TxType = types.DynamicFeeTxType
		msg.GasFeeCap, msg.GasTipCap = cap, tip
		msg.GasPrice = new(big.Int).Set(tip)
		if header.BaseFee != nil {
			msg.GasPrice.Add(msg.GasPrice, header.BaseFee)
		}
		if msg.GasPrice.Cmp(cap) > 0 {
			msg.GasPrice.Set(cap)
		}
	}
	if args.AuthorizationList != nil {
		msg.TxType = types.SetCodeTxType
		for _, wire := range args.AuthorizationList {
			auth := types.Authorization{}
			auth.ChainID, err = hexutil.DecodeBig(wire.ChainID)
			if err != nil {
				return nil, invalid("authorization chainId", err)
			}
			auth.Address, err = address(wire.Address)
			if err != nil {
				return nil, invalid("authorization address", err)
			}
			auth.Nonce, err = hexutil.DecodeUint64(wire.Nonce)
			if err != nil {
				return nil, invalid("authorization nonce", err)
			}
			for _, field := range []struct {
				text string
				dest **big.Int
			}{{wire.V, &auth.V}, {wire.R, &auth.R}, {wire.S, &auth.S}} {
				*field.dest, err = hexutil.DecodeBig(field.text)
				if err != nil {
					return nil, invalid("authorization signature", err)
				}
			}
			msg.AuthList = append(msg.AuthList, auth)
		}
	}

	if args.BlobVersionedHashes != nil || args.MaxFeePerBlobGas != nil {
		if len(args.BlobVersionedHashes) == 0 || to == nil || args.AuthorizationList != nil {
			return nil, invalid("blob fields", fmt.Errorf("nonempty hashes and recipient required; authorizationList incompatible"))
		}
		msg.TxType = types.BlobTxType
		for _, wire := range args.BlobVersionedHashes {
			raw, err := hexutil.Decode(wire)
			if err != nil || len(raw) != 32 || raw[0] != coregas.BlobTxHashVersion {
				return nil, invalid("blobVersionedHashes", fmt.Errorf("invalid versioned hash"))
			}
			msg.BlobHashes = append(msg.BlobHashes, types.BytesToHash(raw))
		}
		msg.BlobGasFeeCap = new(big.Int)
		if args.MaxFeePerBlobGas != nil {
			msg.BlobGasFeeCap, err = hexutil.DecodeBig(*args.MaxFeePerBlobGas)
			if err != nil || msg.BlobGasFeeCap.BitLen() > 256 {
				return nil, invalid("maxFeePerBlobGas", fmt.Errorf("invalid uint256"))
			}
		}
	}
	return msg, nil
}

func (b *RPCBackend) callState(number rpctypes.BlockNumber) (*types.Header, state.StateDB, error) {
	header := b.HeaderByNumber(number)
	if header == nil {
		return nil, nil, fmt.Errorf("block not found")
	}
	db, err := b.StateAtHeader(header)
	if err != nil {
		return nil, nil, err
	}
	return header, db.Dup(), nil
}
func (b *RPCBackend) executeCall(args rpctypes.CallArgs, number rpctypes.BlockNumber, transactionRules bool, tracer *vm.StructLogTracer, overrides ...rpctypes.StateOverride) (*execution.ExecutionResult, error) {
	header, db, err := b.callState(number)
	if err != nil {
		return nil, err
	}
	if err := applySimulationOverrides(db, overrides...); err != nil {
		return nil, err
	}
	msg, err := parseSimulationMessage(args, header, db, b.node.Blockchain().Config())
	if err != nil {
		return nil, err
	}
	cfg := vm.Config{}
	if tracer != nil {
		cfg = vm.Config{Debug: true, Tracer: tracer}
	}
	runtime, err := simulationPrecompiles(b.node.Blockchain().Config(), header, db, mergeStateOverrides(overrides))
	if err != nil {
		return nil, err
	}
	return execution.SimulateMessageWithConfig(b.node.Blockchain().Config(), b.node.Blockchain().GetHashFn(), &simulationBalanceState{StateDB: db}, header, msg, new(gaspool.GasPool).AddGas(msg.GasLimit), execution.SimulationConfig{TransactionRules: transactionRules, VMConfig: cfg, Overrides: execution.EVMOverrides{Precompiles: runtime}})
}

func (b *RPCBackend) Call(args rpctypes.CallArgs, number rpctypes.BlockNumber, overrides ...rpctypes.StateOverride) ([]byte, uint64, error) {
	result, err := b.executeCall(args, number, false, nil, overrides...)
	if err != nil {
		return nil, 0, err
	}
	return result.ReturnData, result.UsedGas, result.Err
}
func (b *RPCBackend) TraceCall(args rpctypes.CallArgs, number rpctypes.BlockNumber, overrides ...rpctypes.StateOverride) (*vm.StructLogTracer, error) {
	tracer := vm.NewStructLogTracer()
	result, err := b.executeCall(args, number, false, tracer, overrides...)
	if err != nil {
		return nil, err
	}
	tracer.CaptureEnd(result.ReturnData, result.UsedGas, result.Err)
	return tracer, nil
}
func (b *RPCBackend) EstimateGas(args rpctypes.CallArgs, number rpctypes.BlockNumber, overrides ...rpctypes.StateOverride) (uint64, error) {
	header, base, err := b.callState(number)
	if err != nil {
		return 0, err
	}
	if err := applySimulationOverrides(base, overrides...); err != nil {
		return 0, err
	}
	msg, err := parseSimulationMessage(args, header, base, b.node.Blockchain().Config())
	if err != nil {
		return 0, err
	}
	hi := header.GasLimit
	if args.Gas != nil {
		hi = min(hi, msg.GasLimit)
	}
	if b.node.Blockchain().Config().IsAmsterdam(header.Time) {
		hi = min(hi, vm.AmsterdamMaxTotalGas)
	} else if b.node.Blockchain().Config().IsOsaka(header.Time) {
		hi = min(hi, vm.AmsterdamMaxExecutionGas)
	}
	maximumPrice := msg.GasPrice
	if msg.GasFeeCap != nil {
		maximumPrice = msg.GasFeeCap
	}
	if maximumPrice.Sign() > 0 {
		available := new(big.Int).Sub(base.GetBalance(msg.From), msg.Value)
		if available.Sign() < 0 {
			return 0, execution.ErrInsufficientBalance
		}
		allowance := new(big.Int).Div(available, maximumPrice)
		if allowance.IsUint64() {
			hi = min(hi, allowance.Uint64())
		}
	}
	runtime, err := simulationPrecompiles(b.node.Blockchain().Config(), header, base, mergeStateOverrides(overrides))
	if err != nil {
		return 0, err
	}
	run := func(limit uint64) error {
		candidate := *msg
		candidate.GasLimit = limit
		result, err := execution.SimulateMessageWithConfig(b.node.Blockchain().Config(), b.node.Blockchain().GetHashFn(), &simulationBalanceState{StateDB: base.Dup()}, header, &candidate, new(gaspool.GasPool).AddGas(limit), execution.SimulationConfig{TransactionRules: true, Overrides: execution.EVMOverrides{Precompiles: runtime}})
		if err != nil {
			return err
		}
		return result.Err
	}
	if err := run(hi); err != nil {
		return 0, err
	}
	lo := uint64(0)
	for lo+1 < hi {
		mid := lo + (hi-lo)/2
		if run(mid) != nil {
			lo = mid
		} else {
			hi = mid
		}
	}
	return hi, nil
}

func applySimulationOverrides(db state.StateDB, overrides ...rpctypes.StateOverride) error {
	for _, set := range overrides {
		for text, ov := range set {
			raw, err := hexutil.Decode(text)
			if err != nil || len(raw) != 20 {
				return fmt.Errorf("%w: override address", rpcbackend.ErrInvalidCallArgs)
			}
			address := types.BytesToAddress(raw)
			invalid := func(err error) error {
				return fmt.Errorf("%w: account override: %v", rpcbackend.ErrInvalidCallArgs, err)
			}
			if ov.State != nil && ov.StateDiff != nil {
				return invalid(fmt.Errorf("state and stateDiff are mutually exclusive"))
			}
			if ov.Balance != nil {
				value, err := hexutil.DecodeBig(*ov.Balance)
				if err != nil {
					return invalid(err)
				}
				if value.BitLen() > 256 {
					return invalid(fmt.Errorf("balance exceeds uint256"))
				}
				difference := new(big.Int).Sub(value, db.GetBalance(address))
				if difference.Sign() > 0 {
					db.AddBalance(address, difference)
				} else if difference.Sign() < 0 {
					db.SubBalance(address, difference.Neg(difference))
				}
			}
			if ov.Nonce != nil {
				value, err := hexutil.DecodeUint64(*ov.Nonce)
				if err != nil {
					return invalid(err)
				}
				db.SetNonce(address, value)
			}
			if ov.Code != nil {
				code, err := hexutil.Decode(*ov.Code)
				if err != nil {
					return invalid(err)
				}
				db.SetCode(address, code)
			}
			values := ov.StateDiff
			if ov.State != nil {
				values = ov.State
				source, ok := db.(interface {
					StorageSnapshot(types.Address) map[types.Hash]types.Hash
				})
				if !ok {
					return invalid(fmt.Errorf("state backend cannot replace complete storage"))
				}
				for slot := range source.StorageSnapshot(address) {
					db.SetState(address, slot, types.Hash{})
				}
			}
			for key, value := range values {
				k, err := hexutil.Decode(key)
				if err != nil || len(k) != 32 {
					return invalid(fmt.Errorf("storage key must contain 32 bytes"))
				}
				v, err := hexutil.Decode(value)
				if err != nil || len(v) != 32 {
					return invalid(fmt.Errorf("storage value must contain 32 bytes"))
				}
				db.SetState(address, types.BytesToHash(k), types.BytesToHash(v))
			}
		}
	}
	return nil
}

func mergeStateOverrides(overrides []rpctypes.StateOverride) rpctypes.StateOverride {
	result := make(rpctypes.StateOverride)
	for _, set := range overrides {
		for address, value := range set {
			result[address] = value
		}
	}
	return result
}
