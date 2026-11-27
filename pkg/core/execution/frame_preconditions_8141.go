package execution

import (
	"errors"
	"math/big"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/gas"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
)

type framePreparation8141 struct {
	profile     *types.FrameTx8250Delta
	legacyNonce uint64
	budget      types.FrameGasBudget
	payment     *types.FramePayment
	blobFee     *big.Int
}

// ValidateFramePreconditions8141 shares the executor's checks before any frame:
// context, structure/signatures, selected nonce state, aggregate fee budget and
// both gas reservations. It performs no EVM/proof execution, nonce consumption,
// payment, state writes or gas-pool reservation. Success is not full validity.
// Callers must surface state.ReadError before interpreting an invalid candidate.
func ValidateFramePreconditions8141(cfg *config.ChainConfig, db state.StateDB, header *types.Header, tx *types.FrameTx8141, gp *gaspool.GasPool) error {
	prepared, err := prepareFrame8141(cfg, db, header, tx, nil, nil)
	if err != nil {
		return err
	}
	if gp == nil {
		return errors.New("Frame preconditions require gas capacity")
	}
	if !gp.FitsFrame(prepared.budget.ExecutionReservation, prepared.budget.StateReservation) {
		return gaspool.ErrGasPoolExhausted
	}
	return nil
}

func prepareFrame8141(cfg *config.ChainConfig, db state.StateDB, header *types.Header, tx *types.FrameTx8141, profile *types.FrameTx8250Delta, blobBaseFee *big.Int) (_ *framePreparation8141, err error) {
	if err := validateFrameContext8141(cfg, header, tx, profile != nil); err != nil {
		return nil, err
	}
	// Explicit research callers pass a profile; public callers use the schema
	// already checked against this exact block context above.
	if profile == nil {
		profile = tx.NonceProfile8250()
	}

	if db == nil {
		return nil, errors.New("Frame preconditions require account state")
	}
	defer func() {
		if readErr := state.ReadError(db); readErr != nil {
			err = readErr
		}
	}()
	var legacyNonce uint64
	var budget types.FrameGasBudget
	if profile == nil {
		if err := tx.ValidateWithExtensions(cfg.FrameExtensions(header.Time)); err != nil {
			return nil, err
		}
		legacyNonce = db.GetNonce(tx.Sender)
		if legacyNonce != tx.Nonce {
			return nil, errors.New("EIP-8141 nonce mismatch")
		}
		budget, err = tx.GasBudget()
	} else {
		if err := profile.ValidateWithExtensions(cfg.FrameExtensions(header.Time)); err != nil {
			return nil, err
		}
		if reader, ok := db.(interface{ FrameLegacyNonce8250(types.Address) uint64 }); ok && len(profile.NonceKeys) > 0 {
			legacyNonce = reader.FrameLegacyNonce8250(tx.Sender)
		} else {
			legacyNonce = db.GetNonce(tx.Sender)
		}
		if err := validateNonceSet8250(db, profile); err != nil {
			return nil, err
		}
		budget, err = profile.GasBudget()
	}
	if err != nil {
		return nil, err
	}
	blobFee := big.NewInt(1)
	if header.ExcessBlobGas != nil {
		blobFee = gas.BlobBaseFeeForHeader(cfg, header)
	}
	if blobBaseFee != nil {
		blobFee = new(big.Int).Set(blobBaseFee)
	}
	payment, err := tx.Payment(header.BaseFee, blobFee)
	if profile != nil {
		payment, err = profile.Payment(header.BaseFee, blobFee)
	}
	if err != nil {
		return nil, err
	}
	return &framePreparation8141{profile: profile, legacyNonce: legacyNonce, budget: budget, payment: payment, blobFee: blobFee}, nil
}
