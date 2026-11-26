package types

import (
	"errors"
	"math"
	"math/big"
	"math/bits"

	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	MaxNonceKeys8250               = 16
	KeyedNonceAccessGas8250        = 12100
	KeyedNonceFirstUseStateGas8250 = 97920
)

var NonceManager8250 = HexToAddress("0x8250968C12e01A19d6F667b9B2F3b3A4d0e51cB7")
var NonceManagerCode8250 = []byte{0x60, 0, 0x60, 0, 0xfd}

// FrameTx8250Delta replaces only the EIP-8141 nonce, inheriting its two-field
// fee budget (EIPs #11968 and #12414). It is also the independent semantic
// representation used by the nonce conformance vectors.
type FrameTx8250Delta struct {
	ChainID             *big.Int
	NonceKeys           []*big.Int
	NonceSeq            uint64
	Sender              Address
	Frames              []Frame8141
	Signatures          []FrameSignature
	Fees                FrameFeeBudget
	BlobVersionedHashes []Hash
}

// ExecutionFields returns the unchanged 8141 fields. NonceSeq is deliberately
// not a legacy nonce: the selected nonce domains must be checked separately.
func (tx *FrameTx8250Delta) ExecutionFields() *FrameTx8141 {
	return &FrameTx8141{ChainID: tx.ChainID, Sender: tx.Sender, Frames: tx.Frames,
		Signatures: tx.Signatures, Fees: tx.Fees, BlobVersionedHashes: tx.BlobVersionedHashes}
}

func ValidateNonceKeys8250(keys []*big.Int, seq uint64) error {
	if len(keys) > MaxNonceKeys8250 || (len(keys) == 0 && seq == math.MaxUint64) {
		return errors.New("EIP-8250 invalid nonce set or sequence")
	}
	for i, key := range keys {
		if key == nil || !frameUint256(key) || (i > 0 && keys[i-1].Cmp(key) >= 0) {
			return errors.New("EIP-8250 nonce keys must be uint256 and strictly increasing")
		}
	}
	return nil
}

func NonceSlot8250(sender Address, key *big.Int) Hash {
	var preimage [64]byte
	copy(preimage[12:32], sender[:])
	key.FillBytes(preimage[32:])
	return Hash(gethcrypto.Keccak256Hash(preimage[:]))
}

func NonceKeysHash8250(keys []*big.Int) Hash {
	data := make([]byte, 32*(1+len(keys)))
	new(big.Int).SetUint64(uint64(len(keys))).FillBytes(data[:32])
	for i, key := range keys {
		key.FillBytes(data[(i+1)*32 : (i+2)*32])
	}
	return Hash(gethcrypto.Keccak256Hash(data))
}

func (tx *FrameTx8250Delta) Encode() ([]byte, error) {
	if err := ValidateNonceKeys8250(tx.NonceKeys, tx.NonceSeq); err != nil {
		return nil, err
	}
	payload, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return nil, err
	}
	return append([]byte{FrameTxType}, payload...), nil
}

func DecodeFrameTx8250Delta(payload []byte) (*FrameTx8250Delta, error) {
	tx := new(FrameTx8250Delta)
	if err := rlp.DecodeBytes(payload, tx); err != nil {
		return nil, err
	}
	if err := ValidateNonceKeys8250(tx.NonceKeys, tx.NonceSeq); err != nil {
		return nil, err
	}
	return tx, nil
}

func (tx *FrameTx8250Delta) Hash() (Hash, error) {
	wire, err := tx.Encode()
	if err != nil {
		return Hash{}, err
	}
	return Hash(gethcrypto.Keccak256Hash(wire)), nil
}

func (tx *FrameTx8250Delta) SigningHash() (Hash, error) {
	signatures := make([][]any, 0, len(tx.Signatures))
	for _, sig := range tx.Signatures {
		var witness any = sig.Signature
		if len(sig.Message) == 0 {
			witness = uint64(len(sig.Signature))
		}
		signatures = append(signatures, []any{sig.Scheme, addressPtrToBytes(sig.Signer), sig.Message, witness})
	}
	payload, err := rlp.EncodeToBytes([]any{bigOrZero(tx.ChainID), tx.NonceKeys, tx.NonceSeq, tx.Sender, tx.Frames, signatures, tx.Fees, tx.BlobVersionedHashes})
	if err != nil {
		return Hash{}, err
	}
	return Hash(gethcrypto.Keccak256Hash([]byte{FrameTxType}, payload)), nil
}

func (tx *FrameTx8250Delta) Validate() error {
	if err := ValidateNonceKeys8250(tx.NonceKeys, tx.NonceSeq); err != nil {
		return err
	}
	base := tx.ExecutionFields()
	base.Signatures = nil
	if err := base.Validate(); err != nil {
		return err
	}
	if _, err := tx.GasBudget(); err != nil {
		return err
	}
	digest, err := tx.SigningHash()
	if err != nil {
		return err
	}
	for _, sig := range tx.Signatures {
		if err := sig.validate(tx.Sender, digest); err != nil {
			return err
		}
	}
	return nil
}

func (tx *FrameTx8250Delta) GasBudget() (FrameGasBudget, error) {
	if err := ValidateNonceKeys8250(tx.NonceKeys, tx.NonceSeq); err != nil {
		return FrameGasBudget{}, err
	}
	base, err := tx.ExecutionFields().GasBudget()
	if err != nil {
		return FrameGasBudget{}, err
	}
	data, err := rlp.EncodeToBytes(tx.NonceKeys)
	if err != nil {
		return FrameGasBudget{}, err
	}
	if len(tx.NonceKeys) == 0 || tx.NonceSeq != math.MaxUint64 {
		seq, err := rlp.EncodeToBytes(tx.NonceSeq)
		if err != nil {
			return FrameGasBudget{}, err
		}
		data = append(data, seq...)
	}
	access := uint64(len(tx.NonceKeys)) * KeyedNonceAccessGas8250
	var exec uint64
	for _, f := range tx.Frames {
		exec += f.Limits.Execution
	}
	base.Intrinsic += CalldataTokenGas(data) + access
	base.CalldataFloor += uint64(len(data))*64 + access
	base.ExecutionReservation = max(base.Intrinsic+exec, base.CalldataFloor)
	if base.ExecutionReservation > frame8141ExecutionGasCap {
		return FrameGasBudget{}, errors.New("EIP-8250 execution reservation exceeds transaction cap")
	}
	var carry uint64
	base.MaxGas, carry = bits.Add64(base.ExecutionReservation, base.StateReservation, 0)
	if carry != 0 {
		return FrameGasBudget{}, errors.New("EIP-8250 maximum gas overflow")
	}
	return base, nil
}

func (tx *FrameTx8250Delta) Payment(baseFee, blobFee *big.Int) (*FramePayment, error) {
	budget, err := tx.GasBudget()
	if err != nil {
		return nil, err
	}
	return tx.ExecutionFields().paymentForBudget(baseFee, blobFee, budget)
}

// NewFrameTransaction8250 creates an immutable native transaction with the
// eight-field envelope. Construction copies all inputs, including nonce keys.
func NewFrameTransaction8250(profile *FrameTx8250Delta) *Transaction {
	base := profile.ExecutionFields()
	base.Nonce = profile.NonceSeq
	base.nonce8250 = true
	base.nonceKeys8250 = profile.NonceKeys
	return NewTransaction(base)
}

// NonceProfile8250 returns the selected nonce schema, or nil for pre-fork 8141.
// Nonce is the replay sequence in this profile, not the account nonce.
func (tx *FrameTx8141) NonceProfile8250() *FrameTx8250Delta {
	if !tx.nonce8250 {
		return nil
	}
	return &FrameTx8250Delta{ChainID: tx.ChainID, NonceKeys: tx.nonceKeys8250,
		NonceSeq: tx.Nonce, Sender: tx.Sender, Frames: tx.Frames, Signatures: tx.Signatures,
		Fees: tx.Fees, BlobVersionedHashes: tx.BlobVersionedHashes}
}

// SameReplayIdentity compares the exact public replacement identity.
func (tx *FrameTx8141) SameReplayIdentity(other *FrameTx8141) bool {
	if tx.Sender != other.Sender || tx.Nonce != other.Nonce || tx.nonce8250 != other.nonce8250 || len(tx.nonceKeys8250) != len(other.nonceKeys8250) {
		return false
	}
	for i, key := range tx.nonceKeys8250 {
		if key.Cmp(other.nonceKeys8250[i]) != 0 {
			return false
		}
	}
	return true
}

// HasKeyedNonces separates replay sequences from the legacy account nonce.
func (tx *FrameTx8141) HasKeyedNonces() bool { return tx.nonce8250 && len(tx.nonceKeys8250) > 0 }
