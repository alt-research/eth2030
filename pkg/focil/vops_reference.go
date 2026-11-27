package focil

// This file implements determinate predicates of the Informational EIP-8369
// reference model. It deliberately exposes no consensus activation switch.
import (
	"errors"
	"math/bits"

	"github.com/eth2030/eth2030/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

var ErrProfile2NotSpecified = errors.New("EIP-8369 Profile 2 enforcement unavailable: enforcing Standards Track extension required")

// Profile2Prerequisites enumerates unresolved upstream requirements, rather
// than letting callers self-certify completeness with a boolean feature flag.
func Profile2Prerequisites() []string {
	return []string{
		"Standards Track enforcing extension and conformance vectors",
		"single composed 8141/8250/8272 payload and signature hash",
		"claim encoding, carriage, commitment and default-index semantics",
		"claim count/bytes, exact identity, lookup and deduplication limits",
		"branch retention and pruning of committed claims",
		"execution-validated canonical diffs including keyed nonces and recent roots",
		"executable access matrix including deployment and delegation",
		"index-zero system updates and reconstruction at the claimed index",
		"isolated replay context, warm state and cumulative gas source",
		"distinct code body count and byte bounds within replay budgets",
		"final AA_VOPS_SLOT_COUNT and per-transaction/per-IL VERIFY budgets",
		"full pipeline deadline benchmarks and total attester procedure",
		"post-reveal omission duty under ePBS or disabled Profile 2",
	}
}

func RequireProfile2Enforcement() error { return ErrProfile2NotSpecified }

// ClaimedIndex is a local reference predicate, not a new wire encoding.
// Invalid or missing claims default to the payload end, including fresh keys.
func ClaimedIndex(index *uint64, malformed bool, transactionCount uint64) uint64 {
	if malformed || index == nil || *index > transactionCount {
		return transactionCount
	}
	return *index
}

type ReferenceBudget struct{ PerIL, PerTransaction uint64 }
type ReferenceOccurrence struct {
	Admitted                  bool
	SignatureCost, PrefixCost uint64
	Reason                    string
}

// FillProfile2Reference processes every occurrence before exact-byte dedup.
// Caps are explicit research inputs; the candidate 2**20 is not a constant here.
// Prefix cost counts both declared gas dimensions; this convention is local
// research only until the enforcing extension selects its composed gas model.
func FillProfile2Reference(occurrences []*types.FrameTx8141, cap ReferenceBudget, extensions types.FrameExtensions) ([]ReferenceOccurrence, uint64, error) {
	if cap.PerIL == 0 || cap.PerTransaction == 0 || cap.PerTransaction > cap.PerIL {
		return nil, 0, errors.New("invalid reference budget")
	}
	remaining := cap.PerIL
	out := make([]ReferenceOccurrence, len(occurrences))
	for i, tx := range occurrences {
		r := &out[i]
		if tx == nil || tx.ValidateStructure8141(extensions) != nil || len(tx.BlobVersionedHashes) != 0 {
			r.Reason = "not structurally eligible"
			continue
		}
		end, shape := referencePrefix(tx)
		// A mismatched but recognizable prefix still incurs its declared debit
		// after signatures pass. Unrecognizable shapes use the full declared
		// frame prefix as a conservative bounded research accounting surface.
		for _, sig := range tx.Signatures {
			var n uint64
			switch sig.Scheme {
			case types.FrameSignatureArbitrary:
				n = 100
			case types.FrameSignatureSecp256k1:
				n = 2800
			case types.FrameSignatureP256:
				n = 6700
			default:
				return nil, remaining, errors.New("unrecognized signature cost")
			}
			var carry uint64
			r.SignatureCost, carry = bits.Add64(r.SignatureCost, n, 0)
			if carry != 0 {
				return nil, remaining, errors.New("signature cost overflow")
			}
		}
		for _, f := range tx.Frames[:end+1] {
			for _, n := range []uint64{f.Limits.Execution, f.Limits.State} {
				var carry uint64
				r.PrefixCost, carry = bits.Add64(r.PrefixCost, n, 0)
				if carry != 0 {
					return nil, remaining, errors.New("prefix cost overflow")
				}
			}
		}
		total, carry := bits.Add64(r.SignatureCost, r.PrefixCost, 0)
		if carry != 0 || total > cap.PerTransaction || total > remaining {
			r.Reason = "does not fit"
			continue
		}
		remaining -= r.SignatureCost
		if tx.VerifyProtocolSignatures8141() != nil {
			r.Reason = "invalid protocol signature"
			continue
		}
		remaining -= r.PrefixCost
		if !shape {
			r.Reason = "outside four admitted prefixes"
			continue
		}
		r.Admitted = true
	}
	return out, remaining, nil
}

// referencePrefix ignores only an optional leading expiry frame. In
// particular, recent-root and PROOF frames are not silently stripped.
func referencePrefix(tx *types.FrameTx8141) (end int, admitted bool) {
	end = len(tx.Frames) - 1
	start := 0
	if tx.Frames[0].Target != nil && *tx.Frames[0].Target == types.FrameExpiryVerifier {
		start++
	}
	if start < len(tx.Frames) && tx.Frames[start].Mode == types.ModeDefault && tx.Frames[start].Flags == 0 {
		start++
	}
	if start >= len(tx.Frames) {
		return end, false
	}
	v := tx.Frames[start]
	target := tx.Sender
	if v.Target != nil {
		target = *v.Target
	}
	if v.Mode != types.ModeVerify || target != tx.Sender {
		return end, false
	}
	end = start
	switch v.Flags {
	case types.FrameApproveExecution | types.FrameApprovePayment:
	case types.FrameApproveExecution:
		end++
		if end >= len(tx.Frames) {
			return len(tx.Frames) - 1, false
		}
		if tx.Frames[end].Mode != types.ModeVerify || tx.Frames[end].Flags != types.FrameApprovePayment {
			return end, false
		}
	default:
		return end, false
	}
	for _, f := range tx.Frames[:end+1] {
		if f.Flags&types.FrameAtomicBatch != 0 {
			return end, false
		}
	}
	for _, f := range tx.Frames[end+1:] {
		if f.Mode == types.ModeVerify {
			return end, false
		}
	}
	return end, true
}

// ReferenceSurface describes ordinary storage only. Protocol-managed nonce and
// recent-root entries are authenticated separately by their existing native
// consumers; mappings/proxy slots cannot enter through this predicate.
type ReferenceSurface struct {
	Sender, Payer types.Address
	SlotCount     uint64
}

func (s ReferenceSurface) AllowsStorage(address types.Address, slot types.Hash) bool {
	if s.SlotCount == 0 || address != s.Sender && address != s.Payer {
		return false
	}
	for _, b := range slot[:24] {
		if b != 0 {
			return false
		}
	}
	var n uint64
	for _, b := range slot[24:] {
		n = n<<8 | uint64(b)
	}
	return n < s.SlotCount
}
func AuthenticateReferenceCode(code []byte, expected types.Hash) error {
	if types.Hash(gethcrypto.Keccak256Hash(code)) != expected {
		return errors.New("AA-VOPS authenticated code unavailable")
	}
	return nil
}
