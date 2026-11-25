package chain

import (
	"math/big"

	"github.com/eth2030/eth2030/core/gas"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
)

// Stored bodies have passed block admission. Restore their nonserialized sender
// cache without applying today's low-S or chain-ID rules to historical signatures.
func restoreStoredSender(tx *types.Transaction) {
	switch inner := tx.Inner().(type) {
	case *types.AATx:
		tx.SetSender(inner.Sender)
	case *types.FrameTx:
		tx.SetSender(inner.Sender)
	case *types.FrameTx8141:
		tx.SetSender(inner.Sender)
	case *types.PQTransaction:
		tx.SetSender(types.PQPubKeyToAddress(inner.PQPublicKey))
	default:
		if tx.Type() > types.SetCodeTxType && tx.Type() != types.MultiDimFeeTxType {
			return
		}
		v, r, s := tx.RawSignatureValues()
		if v == nil || r == nil || s == nil {
			return
		}
		var parity byte
		if tx.Type() == types.LegacyTxType {
			switch {
			case v.Cmp(big.NewInt(27)) == 0 || v.Cmp(big.NewInt(28)) == 0:
				parity = byte(v.Uint64() - 27)
			case v.Cmp(big.NewInt(35)) >= 0:
				parity = byte(new(big.Int).Sub(v, big.NewInt(35)).Bit(0))
			default:
				return
			}
		} else {
			if !v.IsUint64() || v.Uint64() > 1 {
				return
			}
			parity = byte(v.Uint64())
		}
		if sender, err := types.RecoverPlain(tx.SigningHash(), r, s, parity); err == nil {
			tx.SetSender(sender)
		}
	}
}

func (bc *Blockchain) deriveStoredReceipt(receipt *types.Receipt, tx *types.Transaction, header *types.Header) {
	// Canonical frame fees are reconstructed by their dedicated receipt decoder.
	if _, frame := tx.Inner().(*types.FrameTx8141); frame {
		return
	}
	if price := tx.GasPrice(); price != nil {
		receipt.EffectiveGasPrice = new(big.Int).Set(price)
	}
	switch tx.Type() {
	case types.DynamicFeeTxType, types.BlobTxType, types.SetCodeTxType, types.MultiDimFeeTxType:
		if header.BaseFee != nil {
			receipt.EffectiveGasPrice = new(big.Int).Add(header.BaseFee, tx.GasTipCap())
			if receipt.EffectiveGasPrice.Cmp(tx.GasFeeCap()) > 0 {
				receipt.EffectiveGasPrice.Set(tx.GasFeeCap())
			}
		}
	}
	if tx.Type() <= types.SetCodeTxType && tx.To() == nil && tx.Sender() != nil {
		receipt.ContractAddress = vm.CreateAddressFromNonce(*tx.Sender(), tx.Nonce())
	}
	if used := tx.BlobGas(); used > 0 {
		receipt.BlobGasUsed = used
		if header.ExcessBlobGas != nil {
			receipt.BlobGasPrice = gas.BlobBaseFeeForHeader(bc.config, header)
		}
	}
	if used := tx.CalldataGas(); used > 0 && header.CalldataExcessGas != nil {
		receipt.CalldataGasUsed = used
		receipt.CalldataGasPrice = gas.CalcCalldataBaseFeeFromHeader(header)
	}
}
