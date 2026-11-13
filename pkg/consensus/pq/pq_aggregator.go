// pq_aggregator.go implements PQ aggregator role types, duty selection, and
// the DefaultPQAggregator for collecting and aggregating XMSS signatures.
// P2P integration (LEAN-3.3/3.4): the aggregator broadcasts AggregateRequests
// on the "pq_agg_request" topic and publishes STARKSignatureAggregation results
// on the "pq_agg_result" topic via the AggGossipPublisher interface.
package pq

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/eth2030/eth2030/crypto"
)

// AggGossipPublisher abstracts the P2P gossip layer for the PQ aggregator.
// It is satisfied by *p2p.TopicManager; using an interface avoids an import
// cycle between the consensus and p2p packages.
type AggGossipPublisher interface {
	// Publish sends data to all subscribers of the named topic string.
	// The topic string is the canonical name (e.g. "pq_agg_request").
	Publish(topicName string, data []byte) error
}

const (
	topicPQAggRequest = "pq_agg_request"
	topicPQAggResult  = "pq_agg_result"
)

// PQ aggregator errors.
var (
	ErrAggregatorNoBundles       = errors.New("pq_aggregator: no signature bundles")
	ErrAggregatorXMSSUnsupported = errors.New("pq_aggregator: XMSS statement verifier unavailable")
	ErrAggregatorCommittee       = errors.New("pq_aggregator: no authorized committee key for attestation")
)

// PQAggregatorDuty describes an aggregator's assignment for a specific slot.
type PQAggregatorDuty struct {
	ValidatorIndex uint64
	Slot           uint64
	Epoch          uint64
}

// XMSSSignatureBundle holds one validator's XMSS signature contribution.
type XMSSSignatureBundle struct {
	ValidatorIndex uint64
	Signature      []byte // XMSS signature bytes
	PublicKey      []byte // serialized pubkey (leanSig or raw)
}

// AggregateRequest describes the parameters for a signature collection request.
type AggregateRequest struct {
	Slot        uint64
	MessageHash [32]byte
	Validators  []uint64 // expected validator indices
}

// PQAggregator is the interface for collecting and producing PQ aggregates.
type PQAggregator interface {
	CollectSignatures(slot uint64, validators []uint64) ([]XMSSSignatureBundle, error)
	ProduceAggregate(bundles []XMSSSignatureBundle) (*STARKSignatureAggregation, error)
	PropagateAggregate(agg *STARKSignatureAggregation) error
}

// SelectAggregators selects 1-4 deterministic aggregators per slot.
// It uses keccak256(slot_bytes || epochRandao[:]) to derive a seed, then
// selects indices from that seed, skipping the proposerIndex.
func SelectAggregators(slot, epoch uint64, epochRandao [32]byte, numValidators uint64, proposerIndex uint64) []PQAggregatorDuty {
	if numValidators == 0 {
		return nil
	}

	// Compute seed = keccak256(slot_bytes || epochRandao).
	var slotBuf [8]byte
	putUint64BE(slotBuf[:], slot)
	seed := crypto.Keccak256(slotBuf[:], epochRandao[:])

	// Count = 1 + (seed[0] % 4) → gives 1-4 aggregators.
	count := int(1 + seed[0]%4)

	duties := make([]PQAggregatorDuty, 0, count)
	seen := make(map[uint64]bool)

	for i := 0; len(duties) < count; i++ {
		// Derive the next candidate using successive bytes of the seed.
		// Combine seed with iteration counter for spread.
		var iterBuf [8]byte
		putUint64BE(iterBuf[:], uint64(i))
		iterSeed := crypto.Keccak256(seed, iterBuf[:])
		candidate := binary64FromBytes(iterSeed) % numValidators

		if candidate == proposerIndex || seen[candidate] {
			// Try the next candidate by iterating further.
			if i > int(numValidators)*4 {
				// Guard: avoid infinite loop when all slots are taken.
				break
			}
			continue
		}
		seen[candidate] = true
		duties = append(duties, PQAggregatorDuty{
			ValidatorIndex: candidate,
			Slot:           slot,
			Epoch:          epoch,
		})
	}

	return duties
}

// putUint64BE writes v as big-endian uint64 into b (must be len >= 8).
func putUint64BE(b []byte, v uint64) {
	b[0] = byte(v >> 56)
	b[1] = byte(v >> 48)
	b[2] = byte(v >> 40)
	b[3] = byte(v >> 32)
	b[4] = byte(v >> 24)
	b[5] = byte(v >> 16)
	b[6] = byte(v >> 8)
	b[7] = byte(v)
}

// binary64FromBytes interprets the first 8 bytes of b as a big-endian uint64.
func binary64FromBytes(b []byte) uint64 {
	return uint64(b[0])<<56 | uint64(b[1])<<48 | uint64(b[2])<<40 | uint64(b[3])<<32 |
		uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7])
}

// CommitteeKeyLookup resolves an authorized ML-DSA key for the specified duty.
type CommitteeKeyLookup func(slot, committee, validator uint64) ([]byte, bool)

// DefaultPQAggregator is a concrete PQ aggregator implementation.
// Set Publisher to enable real P2P broadcast; if nil, operations are local-only.
type DefaultPQAggregator struct {
	aggregator *STARKSignatureAggregator
	collected  []XMSSSignatureBundle
	mu         sync.Mutex
	// Publisher is the gossip backend used for P2P broadcast (LEAN-3.3/3.4).
	Publisher     AggGossipPublisher
	committeeKeys CommitteeKeyLookup
}

// NewDefaultPQAggregator creates a new DefaultPQAggregator without a P2P publisher.
// Call SetPublisher before use in production to enable network broadcast.
func NewDefaultPQAggregator() *DefaultPQAggregator {
	return &DefaultPQAggregator{
		aggregator: NewSTARKSignatureAggregator(),
		collected:  make([]XMSSSignatureBundle, 0),
	}
}

// SetPublisher wires the P2P gossip publisher so the aggregator can broadcast
// AggregateRequests and propagate results over the network (LEAN-3.3/3.4).
func (a *DefaultPQAggregator) SetPublisher(pub AggGossipPublisher) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.Publisher = pub
}

// SetCommitteeKeyLookup configures the consensus membership check for publishing.
func (a *DefaultPQAggregator) SetCommitteeKeyLookup(lookup CommitteeKeyLookup) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.committeeKeys = lookup
}

// AddSignatureBundle adds a received bundle to the collection set.
func (a *DefaultPQAggregator) AddSignatureBundle(bundle XMSSSignatureBundle) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.collected = append(a.collected, bundle)
	return nil
}

// CollectSignatures broadcasts an AggregateRequest on the "pq_agg_request" P2P
// topic (LEAN-3.3) and returns locally collected bundles. In a full implementation
// the caller would wait up to t=3s of the slot for responses; here we broadcast
// and return whatever has been accumulated via AddSignatureBundle.
func (a *DefaultPQAggregator) CollectSignatures(slot uint64, validators []uint64) ([]XMSSSignatureBundle, error) {
	// Broadcast the collection request so remote validators can respond.
	a.mu.Lock()
	publisher := a.Publisher
	a.mu.Unlock()
	if publisher != nil {
		req := AggregateRequest{Slot: slot, Validators: validators}
		data, err := json.Marshal(req)
		if err != nil {
			return nil, fmt.Errorf("pq_aggregator: marshal aggregate request: %w", err)
		}
		if err := publisher.Publish(topicPQAggRequest, data); err != nil {
			// Non-fatal: log and continue with locally collected bundles.
			_ = err
		}
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	result := make([]XMSSSignatureBundle, len(a.collected))
	copy(result, a.collected)
	return result, nil
}

// ProduceAggregate requires an XMSS verifier before these bundles can be used.
func (a *DefaultPQAggregator) ProduceAggregate(bundles []XMSSSignatureBundle) (*STARKSignatureAggregation, error) {
	if len(bundles) == 0 {
		return nil, ErrAggregatorNoBundles
	}
	return nil, ErrAggregatorXMSSUnsupported
}

// PropagateAggregate checks the signed batch before broadcasting its
// original attestations on the "pq_agg_result" topic.
func (a *DefaultPQAggregator) PropagateAggregate(agg *STARKSignatureAggregation) error {
	if agg == nil {
		return ErrSTARKAggNilResult
	}
	if _, err := a.aggregator.Verify(agg); err != nil {
		return err
	}
	a.mu.Lock()
	publisher, lookup := a.Publisher, a.committeeKeys
	a.mu.Unlock()
	if publisher != nil {
		if lookup == nil {
			return ErrAggregatorCommittee
		}
		for _, att := range agg.Signatures {
			expected, ok := lookup(att.Slot, att.CommitteeIndex, att.ValidatorIndex)
			if !ok || !bytes.Equal(expected, att.PQPublicKey) {
				return ErrAggregatorCommittee
			}
		}
		data, err := json.Marshal(agg)
		if err != nil {
			return fmt.Errorf("pq_aggregator: marshal stark aggregate: %w", err)
		}
		if err := publisher.Publish(topicPQAggResult, data); err != nil {
			return fmt.Errorf("pq_aggregator: publish pq_agg_result: %w", err)
		}
	}
	return nil
}
