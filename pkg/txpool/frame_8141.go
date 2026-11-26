package txpool

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"math"
	"math/big"
	"sort"
	"strconv"

	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/framepay"
	"github.com/eth2030/eth2030/core/gas"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto/bls"
)

func frameFeeBumped(old, next *big.Int) bool { return frameFeeBumpedPercent(old, next, PriceBump) }
func frameFeeBumpedPercent(old, next *big.Int, percent uint64) bool {
	if old == nil {
		old = new(big.Int)
	}
	if next == nil {
		return false
	}
	// Round upwards and require an actual increase, including zero-price replacements.
	multiplier := new(big.Int).Add(new(big.Int).SetUint64(percent), big.NewInt(100))
	threshold := new(big.Int).Mul(old, multiplier)
	threshold.Add(threshold, big.NewInt(99)).Div(threshold, big.NewInt(100))
	return next.Cmp(old) > 0 && next.Cmp(threshold) >= 0
}

// Validation remains pure with respect to persistent pool metadata.
func (pool *TxPool) validateFrame8141(tx *types.Transaction) error {
	_, err := pool.validateFrame8141Result(tx)
	return err
}
func (pool *TxPool) frameProfile() string {
	encoded, _ := json.Marshal(pool.frameConfig)
	rules, _ := json.Marshal(pool.frameConfig.Rules(new(big.Int).SetUint64(pool.frameNumber), pool.frameConfig.IsMerge(), pool.frameTime))
	return string(encoded) + string(rules) + strconv.FormatBool(pool.config.ExperimentalFramePaymasterV1)
}
func (pool *TxPool) validateFrame8141Result(tx *types.Transaction) (*execution.FramePrefixValidation, error) {
	frame := tx.Inner().(*types.FrameTx8141)
	if pool.frameConfig == nil || !pool.frameConfig.IsMerge() || !pool.frameConfig.IsHogota(pool.frameTime) {
		return nil, errors.New("EIP-8141 requires Bogotá")
	}
	prefixEnd, err := frame.ValidationPrefix8141(pool.frameTime)
	if err != nil {
		return nil, err
	}
	raw, err := tx.EncodeRLP()
	if err != nil {
		return nil, err
	}
	if len(raw) > MaxTxSize {
		return nil, ErrOversizedRLP
	}
	if pool.lookup.Get(tx.Hash()) != tx {
		if err := validateFrameBlobProofs8141(tx); err != nil {
			return nil, err
		}
	}
	db, ok := pool.state.(state.StateDB)
	if !ok {
		return nil, errors.New("Frame validation requires a snapshot-capable StateDB")
	}
	if frame.Nonce != db.GetNonce(frame.Sender) {
		return nil, errors.New("Frame public nonce must match the current sender nonce")
	}
	budget, err := frame.GasBudget()
	if err != nil {
		return nil, err
	}
	if budget.ExecutionReservation > pool.config.BlockGasLimit || budget.StateReservation > pool.config.BlockGasLimit {
		return nil, ErrGasLimit
	}
	blobFee := pool.blobBaseFee
	if blobFee == nil {
		blobFee = big.NewInt(1)
	}
	payment, err := frame.Payment(pool.baseFee, blobFee)
	if err != nil {
		return nil, err
	}
	if pool.config.MinGasPrice != nil && payment.GasPrice().Cmp(pool.config.MinGasPrice) < 0 {
		return nil, ErrUnderpriced
	}
	header := &types.Header{Number: new(big.Int).SetUint64(pool.frameNumber), GasLimit: pool.config.BlockGasLimit, Time: pool.frameTime, BaseFee: pool.baseFee}
	if pool.frameSlot != nil && *pool.frameSlot < math.MaxUint64 {
		slot := *pool.frameSlot + 1
		header.SlotNumber = &slot
	}
	for _, f := range frame.Frames[:prefixEnd+1] {
		if f.IsRecentRootVerifier() {
			if header.SlotNumber == nil || string(db.GetCode(types.RecentRootAddress)) != string(types.RecentRootCode) {
				return nil, errors.New("recent-root validation requires exact code and next slot")
			}
			refs, _ := types.DecodeRecentRootReferences(f.Data)
			for _, ref := range refs {
				if !ref.IsRecent(*header.SlotNumber) || db.GetState(types.RecentRootAddress, ref.StorageKey()) != ref.EntryHash() {
					return nil, errors.New("recent-root dependency expired or changed")
				}
			}
		}
	}
	validation := pool.lookup.frameValidations[tx.Hash()]
	profile := pool.frameProfile()
	if validation == nil || validation.Profile != profile || !validation.Unchanged(db) {
		validation, err = execution.ValidateFrame8141Dependencies(pool.frameConfig, db, header, tx, blobFee, pool.config.ExperimentalFramePaymasterV1)
		if err != nil {
			return nil, err
		}
		validation.Profile = profile
	}
	payer := validation.Payer
	reserved := frameMaxFee(frame.Fees.MaxFee)
	canonical := pool.config.ExperimentalFramePaymasterV1 && frame.Frames[prefixEnd].Flags == types.FrameApprovePayment && framepay.IsV1(db.GetCode(payer))
	if canonical {
		reserved.Add(reserved, framepay.PendingWithdrawal(db, payer))
	}
	noncanonical := frame.Frames[prefixEnd].Flags == types.FrameApprovePayment && db.GetCodeSize(payer) > 0 && !canonical
	for hash, pending := range pool.lookup.all {
		other, ok := pending.Inner().(*types.FrameTx8141)
		if !ok {
			continue
		}
		if other.Sender == frame.Sender && other.Nonce == frame.Nonce {
			continue
		}
		if other.Sender == frame.Sender {
			return nil, errors.New("one pending Frame transaction per sender")
		}
		if pool.lookup.framePayers[hash] == payer {
			reserved.Add(reserved, frameMaxFee(other.Fees.MaxFee))
			otherEnd, e := other.ValidationPrefix8141(pool.frameTime)
			if noncanonical && e == nil && other.Frames[otherEnd].Flags == types.FrameApprovePayment {
				return nil, errors.New("non-canonical Frame paymaster already has a pending transaction")
			}
		}
	}
	if db.GetBalance(payer).Cmp(reserved) < 0 {
		return nil, ErrInsufficientFunds
	}
	if err := state.ReadError(db); err != nil {
		return nil, err
	}
	tx.SetSender(frame.Sender)
	return validation, nil
}

func (lookup *txLookup) installFrameValidation(hash types.Hash, validation *execution.FramePrefixValidation) {
	lookup.indexFrameExpiry(hash)
	// A revalidated transaction may have taken a different helper/storage path.
	if old := lookup.frameValidations[hash]; old != nil {
		for dep := range old.Dependencies {
			delete(lookup.frameDependencies[dep], hash)
			if len(lookup.frameDependencies[dep]) == 0 {
				delete(lookup.frameDependencies, dep)
			}
		}
	}
	lookup.frameValidations[hash] = validation
	lookup.framePayers[hash] = validation.Payer
	for dep := range validation.Dependencies {
		if lookup.frameDependencies[dep] == nil {
			lookup.frameDependencies[dep] = make(map[types.Hash]struct{})
		}
		lookup.frameDependencies[dep][hash] = struct{}{}
	}
}
func (pool *TxPool) removeFrameLocked(hash types.Hash) {
	tx := pool.lookup.Get(hash)
	if tx == nil {
		return
	}
	from := tx.FrameSender()
	pool.lookup.Remove(hash)
	for _, lists := range []map[types.Address]*txSortedList{pool.pending, pool.queue} {
		if list := lists[from]; list != nil {
			list.Remove(tx.Nonce())
			if list.Len() == 0 {
				delete(lists, from)
			}
		}
	}
}
func (pool *TxPool) revalidateFrames8141() {
	pool.expireFrames8141()
	// Solvency conflicts have a deterministic survivor set, independent of Go map order.
	var txs []*types.Transaction
	for _, tx := range pool.lookup.all {
		if isCanonicalFrame8141(tx) {
			txs = append(txs, tx)
		}
	}
	sort.Slice(txs, func(i, j int) bool { a, b := txs[i].Hash(), txs[j].Hash(); return bytes.Compare(a[:], b[:]) < 0 })
	for _, tx := range txs {
		v, err := pool.validateFrame8141Result(tx)
		if err != nil {
			pool.removeFrameLocked(tx.Hash())
		} else {
			pool.lookup.installFrameValidation(tx.Hash(), v)
		}
	}
}

// frameExpiry orders explicit deadlines and root expiration slots separately
// through a stable local eviction key. No expiry sorts after finite expiry.
func frameExpiry(tx *types.Transaction) uint64 {
	frame, ok := tx.Inner().(*types.FrameTx8141)
	if !ok {
		return math.MaxUint64
	}
	if len(frame.Frames) > 0 {
		f := frame.Frames[0]
		if f.Target != nil && *f.Target == types.FrameExpiryVerifier && len(f.Data) == 8 {
			return binary.BigEndian.Uint64(f.Data)
		}
	}
	return math.MaxUint64
}
func transactionPoolBytes(tx *types.Transaction) uint64 {
	raw, _ := tx.EncodeRLP()
	size := uint64(len(raw))
	if sc := tx.BlobSidecar(); sc != nil {
		for _, list := range [][][]byte{sc.Blobs, sc.Commitments, sc.Proofs} {
			for _, part := range list {
				size += uint64(len(part)) + 9
			}
		}
	}
	return size
}
func (pool *TxPool) fitsBytes(tx *types.Transaction, from types.Address) bool {
	limit := pool.config.MaxBytes
	if limit == 0 {
		limit = 512 << 20
	}
	size := transactionPoolBytes(tx)
	if size > limit {
		return false
	}
	for _, pending := range pool.lookup.all {
		if pool.senderOf(pending) == from && pending.Nonce() == tx.Nonce() {
			continue
		}
		part := transactionPoolBytes(pending)
		if part > limit-size {
			return false
		}
		size += part
	}
	return true
}

func validateFrameBlobProofs8141(tx *types.Transaction) error {
	if err := types.ValidateFrameSidecar8141(tx); err != nil {
		return err
	}
	if len(tx.BlobHashes()) == 0 {
		return nil
	}
	return verifyCellProofSidecar(tx.BlobSidecar())
}

func isCanonicalFrame8141(tx *types.Transaction) bool {
	_, ok := tx.Inner().(*types.FrameTx8141)
	return ok
}

func fullBlobProof(blob, commitment []byte) ([]byte, error) {
	if len(commitment) != 48 {
		return nil, errors.New("invalid blob commitment")
	}
	backend, err := bls.NewGoEthKZGBackend()
	if err != nil {
		return nil, err
	}
	proof, err := backend.ComputeBlobProof(blob, [48]byte(commitment))
	if err != nil {
		return nil, err
	}
	return proof[:], nil
}

// ValidateFrameForBuild does not reuse admission caches: earlier selected
// transactions and skipped slots can change validation in the candidate state.
func (pool *TxPool) ValidateFrameForBuild(tx *types.Transaction, db state.StateDB, header *types.Header) error {
	pool.mu.RLock()
	defer pool.mu.RUnlock()
	blobFee := gas.BlobBaseFeeForHeader(pool.frameConfig, header)
	_, err := execution.ValidateFrame8141Dependencies(pool.frameConfig, db, header, tx, blobFee, pool.config.ExperimentalFramePaymasterV1)
	return err
}

// FrameBuildResult applies a local three-consecutive-unpaid-attempt policy.
// Success resets the counter; removed transactions cannot regain metadata.
func (pool *TxPool) FrameBuildResult(hash types.Hash, approved bool) {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	tx := pool.lookup.Get(hash)
	if tx == nil || !isCanonicalFrame8141(tx) {
		return
	}
	if approved {
		delete(pool.lookup.frameBuildFailures, hash)
		return
	}
	pool.lookup.frameBuildFailures[hash]++
	if pool.lookup.frameBuildFailures[hash] >= 3 {
		pool.removeFrameLocked(hash)
	}
}

func frameMaxFee(value *big.Int) *big.Int {
	if value == nil {
		return new(big.Int)
	}
	return new(big.Int).Set(value)
}
