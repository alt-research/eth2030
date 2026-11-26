package execution

import (
	"errors"
	"math"
	"math/big"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
)

// ExecuteFrame8250Delta is the independent nonce-rule oracle entry point.
// It bypasses public schema selection and requires a disposable research state.
// Production transactions use NewFrameTransaction8250 and the ordinary native
// dispatcher, with explicit EIP8250Time activation and predeployed NONCE_MANAGER.
// Both entry points reuse the same frame loop, journals, settlement and BAL.
func ExecuteFrame8250Delta(cfg *config.ChainConfig, db state.StateDB, header *types.Header,
	tx *types.FrameTx8250Delta, gp *gaspool.GasPool, tracker vm.BALTracker,
) (*types.Receipt, uint64, error) {
	if err := tx.Validate(); err != nil {
		return nil, 0, err
	}
	hash, err := tx.Hash()
	if err != nil {
		return nil, 0, err
	}
	db.SetTxContext(hash, 0)
	return executeFrame8141(cfg, nil, db, header, types.NewTransaction(tx.ExecutionFields()), gp, tracker, tx, false, nil)
}

func validateNonceSet8250(db state.StateDB, tx *types.FrameTx8250Delta) error {
	if len(tx.NonceKeys) == 0 {
		if db.GetNonce(tx.Sender) != tx.NonceSeq {
			return errors.New("EIP-8250 legacy nonce mismatch")
		}
		return nil
	}
	for _, key := range tx.NonceKeys {
		value := db.GetState(types.NonceManager8250, types.NonceSlot8250(tx.Sender, key))
		want := tx.NonceSeq
		if want == math.MaxUint64 {
			want = 0
		}
		if new(big.Int).SetBytes(value[:]).Cmp(new(big.Int).SetUint64(want)) != 0 {
			return errors.New("EIP-8250 keyed nonce mismatch")
		}
	}
	return nil
}

// ValidateFrame8250DeltaPrefix is an explicitly isolated policy consumer. It
// creates no public decoding, activation, import or persistent database route.
func ValidateFrame8250DeltaPrefix(cfg *config.ChainConfig, db state.StateDB, header *types.Header, tx *types.FrameTx8250Delta, blobFee *big.Int) (*FramePrefixValidation, error) {
	if header == nil || tx == nil || db == nil {
		return nil, errors.New("isolated keyed prefix requires complete context")
	}
	if _, err := tx.ValidationPrefix8250WithExtensions(header.Time, cfg.FrameExtensions(header.Time)); err != nil {
		return nil, err
	}
	copied := &frameReadSet{StateDB: db.Dup(), sender: tx.Sender, reads: make(map[FrameDependency]struct{})}
	if copied.GetNonce(types.NonceManager8250) == 0 || string(copied.GetCode(types.NonceManager8250)) != string(types.NonceManagerCode8250) {
		return nil, errors.New("isolated EIP-8250 profile requires deployed nonce manager")
	}
	for _, f := range tx.Frames {
		if f.IsRecentRootVerifier() {
			if header.SlotNumber == nil || string(copied.GetCode(types.RecentRootAddress)) != string(types.RecentRootCode) {
				return nil, errors.New("recent-root runtime or slot mismatch")
			}
			refs, _ := types.DecodeRecentRootReferences(f.Data)
			for _, ref := range refs {
				if !ref.IsRecent(*header.SlotNumber) || copied.GetState(types.RecentRootAddress, ref.StorageKey()) != ref.EntryHash() {
					return nil, errors.New("recent-root dependency changed")
				}
			}
		}
	}
	hash, _ := tx.Hash()
	copied.SetTxContext(hash, 0)
	tracing := vm.Config{OnFrameLegacyNonceRead: func(addr types.Address) { copied.record(addr, FrameNonceDependency, types.Hash{}) }}
	receipt, _, err := executeFrame8141(cfg, nil, copied, header, types.NewTransaction(tx.ExecutionFields()), new(gaspool.GasPool).AddGas(header.GasLimit), nil, tx, true, blobFee, tracing)
	if err != nil {
		return nil, err
	}
	if err := copied.checkPublicBalances(tx.Sender, receipt.Payer); err != nil {
		return nil, err
	}
	result := &FramePrefixValidation{Payer: receipt.Payer, Dependencies: make(map[FrameDependency]string, len(copied.reads))}
	for dep := range copied.reads {
		result.Dependencies[dep] = dep.Value(db)
	}
	if err := errors.Join(state.ReadError(db), state.ReadError(copied)); err != nil {
		return nil, err
	}
	return result, nil
}
