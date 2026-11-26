package txpool

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"math/big"
	"sort"
	"sync"
)

// KeyedFramePolicy8250Delta is a bounded research consumer for D01's isolated
// nonce-only delta. It is deliberately not TxPool/TxData and has no public RPC,
// gossip, block-import or fork activation route. Callers supply exact contexts.
type KeyedFramePolicy8250Delta struct {
	mu      sync.Mutex
	entries map[types.Address]*keyedFrameEntry
}
type keyedFrameEntry struct {
	tx         *types.FrameTx8250Delta
	validation *execution.FramePrefixValidation
	profile    string
}

func NewKeyedFramePolicy8250Delta() *KeyedFramePolicy8250Delta {
	return &KeyedFramePolicy8250Delta{entries: make(map[types.Address]*keyedFrameEntry)}
}
func sameKeyedIdentity(a, b *types.FrameTx8250Delta) bool {
	if a.Sender != b.Sender || a.NonceSeq != b.NonceSeq || len(a.NonceKeys) != len(b.NonceKeys) {
		return false
	}
	for i, key := range a.NonceKeys {
		if key.Cmp(b.NonceKeys[i]) != 0 {
			return false
		}
	}
	return true
}
func (p *KeyedFramePolicy8250Delta) Admit(cfg *config.ChainConfig, db state.StateDB, header *types.Header, tx *types.FrameTx8250Delta, blobFee *big.Int) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if cfg == nil || db == nil || header == nil || tx == nil {
		return errors.New("isolated keyed policy requires complete context")
	}
	raw, err := tx.Encode()
	if err != nil {
		return err
	}
	if len(raw) > MaxTxSize {
		return ErrOversizedRLP
	}
	// Freeze caller-owned slices, fees and nonce keys using the isolated codec.
	frozen, err := types.DecodeFrameTx8250Delta(raw[1:])
	if err != nil {
		return err
	}
	old := p.entries[frozen.Sender]
	if old != nil && (!sameKeyedIdentity(old.tx, frozen) || !frameFeeBumped(old.tx.Fees.MaxFee, frozen.Fees.MaxFee) || !frameFeeBumped(old.tx.Fees.MaxPriorityFeePerGas, frozen.Fees.MaxPriorityFeePerGas)) {
		return ErrReplacementUnderpriced
	}
	if old == nil && len(p.entries) >= MaxPoolSize {
		return ErrTxPoolFull
	}
	result, err := execution.ValidateFrame8250DeltaPrefix(cfg, db, header, frozen, blobFee)
	if err != nil {
		return err
	}
	reserved := frameMaxFee(frozen.Fees.MaxFee)
	end, _ := frozen.ValidationPrefix8250(header.Time)
	paymaster := frozen.Frames[end].Flags == types.FrameApprovePayment && db.GetCodeSize(result.Payer) > 0
	for sender, entry := range p.entries {
		if sender != frozen.Sender && entry.validation.Payer == result.Payer {
			reserved.Add(reserved, frameMaxFee(entry.tx.Fees.MaxFee))
			otherEnd, e := entry.tx.ValidationPrefix8250(header.Time)
			if paymaster && e == nil && entry.tx.Frames[otherEnd].Flags == types.FrameApprovePayment {
				return errors.New("noncanonical keyed paymaster already pending")
			}
		}
	}
	if db.GetBalance(result.Payer).Cmp(reserved) < 0 {
		return ErrInsufficientFunds
	}
	if err := state.ReadError(db); err != nil {
		return err
	}
	p.entries[frozen.Sender] = &keyedFrameEntry{tx: frozen, validation: result, profile: keyedPolicyProfile(cfg, header)}
	return nil
}
func (p *KeyedFramePolicy8250Delta) Remove(sender types.Address) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.entries, sender)
}
func (p *KeyedFramePolicy8250Delta) Contains(sender types.Address) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.entries[sender] != nil
}
func (p *KeyedFramePolicy8250Delta) Revalidate(cfg *config.ChainConfig, db state.StateDB, header *types.Header, blobFee *big.Int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for sender, entry := range p.entries {
		if cfg == nil || !cfg.IsHogota(header.Time) {
			delete(p.entries, sender)
			continue
		}
		// Age/fees must still be checked when nonce/storage dependencies are stable.
		if _, err := entry.tx.ValidationPrefix8250(header.Time); err != nil {
			delete(p.entries, sender)
			continue
		}
		if _, err := entry.tx.Payment(header.BaseFee, blobFee); err != nil {
			delete(p.entries, sender)
			continue
		}
		for _, f := range entry.tx.Frames {
			if f.IsRecentRootVerifier() {
				refs, _ := types.DecodeRecentRootReferences(f.Data)
				for _, ref := range refs {
					if header.SlotNumber == nil || !ref.IsRecent(*header.SlotNumber) {
						delete(p.entries, sender)
					}
				}
			}
		}
		if entry.profile == keyedPolicyProfile(cfg, header) && entry.validation.Unchanged(db) {
			continue
		}
		result, err := execution.ValidateFrame8250DeltaPrefix(cfg, db, header, entry.tx, blobFee)
		if err != nil {
			delete(p.entries, sender)
		} else {
			entry.validation = result
			entry.profile = keyedPolicyProfile(cfg, header)
		}
	}
	// Deterministic aggregate exposure on head/reorg balance changes.
	senders := make([]types.Address, 0, len(p.entries))
	for sender := range p.entries {
		senders = append(senders, sender)
	}
	sort.Slice(senders, func(i, j int) bool { return bytes.Compare(senders[i][:], senders[j][:]) < 0 })
	totals := make(map[types.Address]*big.Int)
	paymasters := make(map[types.Address]bool)
	for _, sender := range senders {
		entry := p.entries[sender]
		payer := entry.validation.Payer
		end, _ := entry.tx.ValidationPrefix8250(header.Time)
		codedPay := entry.tx.Frames[end].Flags == types.FrameApprovePayment && db.GetCodeSize(payer) > 0
		if codedPay && paymasters[payer] {
			delete(p.entries, sender)
			continue
		}
		if totals[payer] == nil {
			totals[payer] = new(big.Int)
		}
		next := new(big.Int).Add(totals[payer], frameMaxFee(entry.tx.Fees.MaxFee))
		if db.GetBalance(payer).Cmp(next) < 0 || state.ReadError(db) != nil {
			delete(p.entries, sender)
		} else {
			totals[payer] = next
			if codedPay {
				paymasters[payer] = true
			}
		}
	}
}

func keyedPolicyProfile(cfg *config.ChainConfig, header *types.Header) string {
	encoded, _ := json.Marshal(cfg)
	rules, _ := json.Marshal(cfg.Rules(header.Number, cfg.IsMerge(), header.Time))
	return string(encoded) + string(rules)
}
