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
	"github.com/eth2030/eth2030/core/frameproof"
	"github.com/eth2030/eth2030/core/gas"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
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
	extensions, _ := json.Marshal(pool.frameConfig.FrameExtensions(pool.frameTime))
	return string(encoded) + string(rules) + string(extensions) + strconv.FormatBool(pool.frameConfig.IsEIP8298(pool.frameTime)) + strconv.FormatBool(pool.frameConfig.IsEIP8151(pool.frameTime)) + strconv.FormatBool(pool.config.ExperimentalFramePaymasterV1) + strconv.FormatBool(pool.config.ExperimentalFramePaymaster12041)
}
func (pool *TxPool) validateFrame8141Result(tx *types.Transaction) (*execution.FramePrefixValidation, error) {
	validation, err := pool.prepareFrame8141Result(tx)
	if err != nil {
		return nil, err
	}
	frame := tx.Inner().(*types.FrameTx8141)
	if err := pool.validateFrameResources8141(frame, validation, true); err != nil {
		return nil, err
	}
	return validation, nil
}

// Prepare against the head independently of other pending transactions. This
// stage can remove stale entries before shared reservations are arbitrated.
func (pool *TxPool) prepareFrame8141Result(tx *types.Transaction) (*execution.FramePrefixValidation, error) {
	paymasterProfile, err := framepay.SelectProfile(pool.config.ExperimentalFramePaymasterV1, pool.config.ExperimentalFramePaymaster12041)
	if err != nil {
		return nil, err
	}
	frame := tx.Inner().(*types.FrameTx8141)
	if pool.frameConfig == nil || !pool.frameConfig.IsMerge() || !pool.frameConfig.IsHogota(pool.frameTime) {
		return nil, errors.New("EIP-8141 requires Bogotá")
	}
	if (frame.NonceProfile8250() != nil) != pool.frameConfig.IsEIP8250(pool.frameTime) {
		return nil, errors.New("Frame nonce schema does not match EIP-8250 activation")
	}
	prefixEnd, err := frame.ValidationPrefix8141WithExtensions(pool.frameTime, pool.frameConfig.FrameExtensions(pool.frameTime))
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
		if err := frameproof.VerifyTransaction(tx, pool.frameConfig.FrameExtensions(pool.frameTime), true); err != nil {
			return nil, err
		}
		if err := validateFrameBlobProofs8141(tx); err != nil {
			return nil, err
		}
	}
	db, ok := pool.state.(state.StateDB)
	if !ok {
		return nil, errors.New("Frame validation requires a snapshot-capable StateDB")
	}
	if !frame.HasKeyedNonces() && frame.Nonce != db.GetNonce(frame.Sender) {
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
		validation, err = execution.ValidateFrame8141DependenciesWithPaymaster(pool.frameConfig, db, header, tx, blobFee, paymasterProfile)
		if err != nil {
			return nil, err
		}
		validation.Profile = profile
	}
	if err := pool.validateFrameResources8141(frame, validation, false); err != nil {
		return nil, err
	}
	tx.SetSender(frame.Sender)
	return validation, nil
}

// EIP-8141 reservations use the signed max_fee, with one withdrawal per payer.
// Aggregate checks run only after all head-invalid entries have left the pool.
func (pool *TxPool) validateFrameResources8141(frame *types.FrameTx8141, validation *execution.FramePrefixValidation, aggregate bool) (result error) {
	db := pool.state.(state.StateDB)
	defer func() {
		if err := state.ReadError(db); err != nil {
			result = err
		}
	}()
	if aggregate {
		if pool.frameConfig.IsEIP8298(pool.frameTime) {
			for dep := range validation.Dependencies {
				if dep.Kind != execution.FrameCodeDependency {
					continue
				}
				code := db.GetCode(dep.Address)
				if !vm.HasOpcode8298(code, vm.DELEGATECALL) && !vm.HasOpcode8298(code, vm.CALLCODE) {
					continue
				}
				dependents := 0
				for hash := range pool.lookup.frameDependencies[dep] {
					other := pool.lookup.Get(hash)
					if other != nil && !other.Inner().(*types.FrameTx8141).SameReplayIdentity(frame) {
						dependents++
					}
				}
				if dependents >= MaxProxyDependents8298 {
					return errors.New("SETCODEFROM proxy dependency pending limit")
				}
			}
		}
		if err := pool.validateFrameSigners8151(frame, validation); err != nil {
			return err
		}
	}
	payer := validation.Payer
	reserved := frameMaxFee(frame.Fees.MaxFee)
	pay := frameHasPayPrefix(frame.Frames)
	paymasterProfile, err := framepay.SelectProfile(pool.config.ExperimentalFramePaymasterV1, pool.config.ExperimentalFramePaymaster12041)
	if err != nil {
		return err
	}
	canonical := pay && paymasterProfile.Matches(db.GetCode(payer))
	if canonical {
		reserved.Add(reserved, paymasterProfile.PendingWithdrawal(db, payer))
	}
	noncanonical := pay && db.GetCodeSize(payer) > 0 && !canonical
	if aggregate {
		for hash, pending := range pool.lookup.all {
			other, ok := pending.Inner().(*types.FrameTx8141)
			if !ok || other.SameReplayIdentity(frame) {
				continue
			}
			if other.Sender == frame.Sender {
				return errors.New("one pending Frame transaction per sender")
			}
			if pool.lookup.framePayers[hash] == payer {
				reserved.Add(reserved, frameMaxFee(other.Fees.MaxFee))
				if noncanonical && frameHasPayPrefix(other.Frames) {
					return errors.New("non-canonical Frame paymaster already has a pending transaction")
				}
			}
		}
	}
	if db.GetBalance(payer).Cmp(reserved) < 0 {
		return ErrInsufficientFunds
	}
	return nil
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
			list.RemoveTx(tx)
			if list.Len() == 0 {
				delete(lists, from)
			}
		}
	}
	if !keyedFrame8250(tx) {
		pool.reconcileScalarPending(from)
	}
}
func (pool *TxPool) revalidateFrames8141() {
	pool.evictRetiredOrigins8298()
	pool.expireFrames8141()
	var txs []*types.Transaction
	for _, tx := range pool.lookup.all {
		if isCanonicalFrame8141(tx) {
			txs = append(txs, tx)
		}
	}
	// Preparation is deterministic but does not arbitrate against stale peers.
	sort.Slice(txs, func(i, j int) bool { a, b := txs[i].Hash(), txs[j].Hash(); return bytes.Compare(a[:], b[:]) < 0 })
	var valid []*types.Transaction
	for _, tx := range txs {
		validation, err := pool.prepareFrame8141Result(tx)
		if err != nil {
			pool.removeFrameLocked(tx.Hash())
			continue
		}
		pool.lookup.installFrameValidation(tx.Hash(), validation)
		valid = append(valid, tx)
	}
	// All metadata now describes the same head. Evict nearest expiry, then
	// lowest effective priority, retaining deterministic hash ties.
	sort.Slice(valid, func(i, j int) bool {
		a, b := valid[i], valid[j]
		if ae, be := frameExpiry(a), frameExpiry(b); ae != be {
			return ae < be
		}
		ap, bp := EffectiveGasPriceAt(a, pool.baseFee, pool.blobBaseFee), EffectiveGasPriceAt(b, pool.baseFee, pool.blobBaseFee)
		if order := ap.Cmp(bp); order != 0 {
			return order < 0
		}
		ah, bh := a.Hash(), b.Hash()
		return bytes.Compare(ah[:], bh[:]) < 0
	})
	for _, tx := range valid {
		frame := tx.Inner().(*types.FrameTx8141)
		if err := pool.validateFrameResources8141(frame, pool.lookup.frameValidations[tx.Hash()], true); err != nil {
			pool.removeFrameLocked(tx.Hash())
		}
	}
}

// frameExpiry orders explicit timestamp deadlines. Root expiry is indexed
// separately in canonical slot units. No deadline sorts after a finite one.
func frameExpiry(tx *types.Transaction) uint64 {
	frame, ok := tx.Inner().(*types.FrameTx8141)
	if !ok {
		return math.MaxUint64
	}
	first := 0
	for first < len(frame.Frames) && frame.Frames[first].Mode == types.ModeProof8504 {
		first++
	}
	if first < len(frame.Frames) {
		f := frame.Frames[first]
		if f.Target != nil && *f.Target == types.FrameExpiryVerifier && len(f.Data) == 8 {
			return binary.BigEndian.Uint64(f.Data)
		}
	}
	return math.MaxUint64
}
func transactionPoolBytes(tx *types.Transaction) uint64 {
	if tx.HasDependencies8504() {
		raw, err := types.EncodeDependencyWrapper8504(tx)
		if err != nil {
			return math.MaxUint64
		}
		return uint64(len(raw))
	}
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
		if pool.senderOf(pending) == from && samePoolIdentity(pending, tx) {
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
	paymasterProfile, err := framepay.SelectProfile(pool.config.ExperimentalFramePaymasterV1, pool.config.ExperimentalFramePaymaster12041)
	if err != nil {
		return err
	}
	validation, err := execution.ValidateFrame8141DependenciesWithPaymaster(pool.frameConfig, db, header, tx, blobFee, paymasterProfile)
	if err != nil {
		return err
	}
	frame := tx.Inner().(*types.FrameTx8141)
	if frameHasPayPrefix(frame.Frames) && paymasterProfile.Matches(db.GetCode(validation.Payer)) {
		// Earlier selected transactions can request a withdrawal. Recheck the
		// selected version's availability in the candidate pre-state, without
		// charging reservations for unrelated transactions still in the pool.
		required := frameMaxFee(frame.Fees.MaxFee)
		required.Add(required, paymasterProfile.PendingWithdrawal(db, validation.Payer))
		insolvent := db.GetBalance(validation.Payer).Cmp(required) < 0
		if err := state.ReadError(db); err != nil {
			return err
		}
		if insolvent {
			return ErrInsufficientFunds
		}
	}
	return state.ReadError(db)
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

// Already-admitted frames have a validated immutable prefix. Classification
// must not re-verify every other transaction's signatures while accounting
// for one payer's exposure. Only VERIFY/1 belongs to the public pay prefix.
func frameHasPayPrefix(frames []types.Frame8141) bool {
	for _, frame := range frames {
		if frame.Mode == types.ModeVerify && frame.Flags == types.FrameApprovePayment {
			return true
		}
	}
	return false
}
