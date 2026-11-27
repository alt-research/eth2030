package heze

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"sync"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
)

type testView struct {
	clock Clock
	state DependentState
	err   error
}

func (v *testView) Clock() (Clock, error)                                     { return v.clock, v.err }
func (v *testView) DependentState(uint64, types.Hash) (DependentState, error) { return v.state, v.err }
func fixture(t *testing.T) (*testView, SignedInclusionList) {
	t.Helper()
	v := &testView{clock: Clock{100, 8999, 12000}, state: DependentState{KnownBlock: true, StateAvailable: true, BlockSlot: 64, ShufflingDependentSlot: 64, PossibleDependent: true, ValidatorPubkeys: make(map[uint64][48]byte), ForkVersion: [4]byte{8}}}
	for i := range v.state.Committee {
		v.state.Committee[i] = uint64(i)
		v.state.ValidatorPubkeys[uint64(i)] = crypto.BLSPubkeyFromSecret(big.NewInt(int64(i + 1)))
	}
	m := InclusionList{100, 7, types.Hash{31: 0xaa}, [][]byte{{0xff}, {0, 0xc0}}}
	signed := SignedInclusionList{Message: m}
	sign(&signed, v)
	return v, signed
}
func sign(s *SignedInclusionList, v *testView) {
	root := s.Message.SigningRoot(v.state.ForkVersion, v.state.GenesisValidatorsRoot)
	s.Signature = crypto.BLSSign(big.NewInt(int64(s.Message.ValidatorIndex+1)), root[:])
}
func encoded(t *testing.T, s SignedInclusionList) []byte {
	t.Helper()
	b, e := s.MarshalSSZ()
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func fromHex(t *testing.T, s string) []byte {
	t.Helper()
	b, e := hex.DecodeString(s)
	if e != nil {
		t.Fatal(e)
	}
	return b
}

func TestGroupEHezeIndependentSSZRootsAndDomain(t *testing.T) {
	raw, e := os.ReadFile("testdata/ssz.json")
	if e != nil {
		t.Fatal(e)
	}
	var f struct {
		Cases []struct {
			Name            string
			Slot, Validator uint64
			DependentRoot   string `json:"dependent_root"`
			Transactions    []string
			MessageSSZ      string `json:"message_ssz"`
			SignedSSZ       string `json:"signed_ssz"`
			Root            string
			ForkVersion     string `json:"fork_version"`
			Genesis         string `json:"genesis_validators_root"`
			SigningRoot     string `json:"signing_root"`
		}
	}
	if e = json.Unmarshal(raw, &f); e != nil {
		t.Fatal(e)
	}
	for _, c := range f.Cases {
		t.Run(c.Name, func(t *testing.T) {
			m := InclusionList{Slot: c.Slot, ValidatorIndex: c.Validator}
			copy(m.DependentRoot[:], fromHex(t, c.DependentRoot))
			for _, x := range c.Transactions {
				m.Transactions = append(m.Transactions, fromHex(t, x))
			}
			data, e := m.MarshalSSZ()
			if e != nil || !bytes.Equal(data, fromHex(t, c.MessageSSZ)) {
				t.Fatal("message SSZ", e)
			}
			root := m.HashTreeRoot()
			if hex.EncodeToString(root[:]) != c.Root {
				t.Fatal("independent progressive root", root.Hex())
			}
			var fork [4]byte
			var genesis [32]byte
			copy(fork[:], fromHex(t, c.ForkVersion))
			copy(genesis[:], fromHex(t, c.Genesis))
			r := m.SigningRoot(fork, genesis)
			if hex.EncodeToString(r[:]) != c.SigningRoot {
				t.Fatal("independent signing root", r.Hex())
			}
			s := SignedInclusionList{Message: m}
			for i := range s.Signature {
				s.Signature[i] = 0x99
			}
			if !bytes.Equal(encoded(t, s), fromHex(t, c.SignedSSZ)) {
				t.Fatal("signed SSZ")
			}
			var decoded SignedInclusionList
			if e = decoded.UnmarshalSSZ(fromHex(t, c.SignedSSZ)); e != nil || !equal(m, decoded.Message) {
				t.Fatal("decode", e)
			}
		})
	}
}
func TestGroupEHezeSizeAndOffsetBoundaries(t *testing.T) {
	_, s := fixture(t)
	s.Message.Transactions = make([][]byte, 8192)
	for i := range s.Message.Transactions {
		s.Message.Transactions[i] = []byte{0xff}
	}
	raw := encoded(t, s)
	if len(raw) != MaxSignedSize {
		t.Fatal("maximum serialized size", len(raw))
	}
	var decoded SignedInclusionList
	if err := decoded.UnmarshalSSZ(raw); err != nil {
		t.Fatal(err)
	}
	if err := CheckTransactions(decoded.Message.Transactions); err != nil {
		t.Fatal("old transaction-count cap leaked", err)
	}
	for _, b := range [][]byte{raw[:151], append(append([]byte(nil), raw...), 0), {0, 0, 0, 0}, {8, 0, 0, 0, 4, 0, 0, 0}} {
		if len(b) < 100 {
			if _, e := DecodeTransactions(b); e == nil {
				t.Fatal("bad list offsets accepted")
			}
		} else if e := decoded.UnmarshalSSZ(b); e == nil {
			t.Fatal("bad container accepted")
		}
	}
	bad := append([]byte(nil), raw...)
	bad[0] = 99
	if e := decoded.UnmarshalSSZ(bad); e == nil {
		t.Fatal("noncanonical signed offset")
	}
	bad = append([]byte(nil), raw...)
	bad[148] = 51
	if e := decoded.UnmarshalSSZ(bad); e == nil {
		t.Fatal("noncanonical message offset")
	}
	if CheckTransactions([][]byte{bytes.Repeat([]byte{1}, 8193)}) == nil || CheckTransactions([][]byte{{1}, {}}) == nil {
		t.Fatal("byte/empty bounds")
	}
	req := Request{Slot: 100, DependentRoot: s.Message.DependentRoot, Indices: Bits{1, 0x80}}
	var r Request
	if e := r.UnmarshalSSZ(req.MarshalSSZ()); e != nil || r != req {
		t.Fatal("request encoding", e)
	}
	if e := r.UnmarshalSSZ(append(req.MarshalSSZ(), 0)); e == nil {
		t.Fatal("trailing request bytes")
	}
}
func TestGroupEHezeAuthenticatedGossipAndFailureAtomicity(t *testing.T) {
	v, s := fixture(t)
	for _, kind := range []string{"missing-context", "unknown-block", "missing-state", "future-dependent", "wrong-branch", "nonmember", "missing-key", "wrong-fork", "wrong-genesis", "bad-signature", "past-slot", "future-slot", "empty-entry", "too-large"} {
		t.Run(kind, func(t *testing.T) {
			v2 := *v
			v2.state = v.state
			v2.state.ValidatorPubkeys = make(map[uint64][48]byte)
			for k, p := range v.state.ValidatorPubkeys {
				v2.state.ValidatorPubkeys[k] = p
			}
			s2 := s.Clone()
			var ctx ConsensusView = &v2
			switch kind {
			case "missing-context":
				ctx = nil
			case "unknown-block":
				v2.state.KnownBlock = false
			case "missing-state":
				v2.state.StateAvailable = false
			case "future-dependent":
				v2.state.BlockSlot++
			case "wrong-branch":
				v2.state.PossibleDependent = false
			case "nonmember":
				s2.Message.ValidatorIndex = 99
			case "missing-key":
				delete(v2.state.ValidatorPubkeys, 7)
			case "wrong-fork":
				v2.state.ForkVersion[0]++
			case "wrong-genesis":
				v2.state.GenesisValidatorsRoot[0]++
			case "bad-signature":
				s2.Signature[0] ^= 1
			case "past-slot":
				s2.Message.Slot--
			case "future-slot":
				s2.Message.Slot++
			case "empty-entry":
				s2.Message.Transactions = [][]byte{{1}, {}}
			case "too-large":
				s2.Message.Transactions = [][]byte{bytes.Repeat([]byte{1}, 8193)}
			}
			store := NewStore()
			if err := store.Gossip(encoded(t, s2), ctx); err == nil {
				t.Fatal("invalid gossip accepted")
			}
			if len(store.entries) != 0 {
				t.Fatal("failed message changed store")
			}
			if err := store.Gossip(encoded(t, s), v); err != nil {
				t.Fatal("failed message consumed quota", err)
			}
		})
	}
	store := NewStore()
	if err := store.Gossip(encoded(t, s), v); err != nil {
		t.Fatal("opaque malformed txs must be propagated", err)
	}
}
func TestGroupEHezeBranchTimelinessEquivocationAndOwnership(t *testing.T) {
	t.Run("fractional-millisecond-deadline", func(t *testing.T) {
		if !timely(Clock{Slot: 1, IntoSlotMS: 9, SlotDurationMS: 13}, 1) || timely(Clock{Slot: 1, IntoSlotMS: 10, SlotDurationMS: 13}, 1) {
			t.Fatal("strict rational deadline rounded early")
		}
	})
	v, s := fixture(t)
	store := NewStore()
	if err := store.Gossip(encoded(t, s), v); err != nil {
		t.Fatal(err)
	}
	k := key(s.Message)
	out := store.Transactions(k, true)
	out[0][0] = 1
	if store.Transactions(k, true)[0][0] != 0xff {
		t.Fatal("caller mutated store")
	}
	v.clock.IntoSlotMS = 9000
	late := s.Clone()
	late.Message.ValidatorIndex = 8
	late.Message.Transactions = [][]byte{{3}}
	sign(&late, v)
	if err := store.Gossip(encoded(t, late), v); err != nil {
		t.Fatal(err)
	}
	if len(store.Transactions(k, true)) != 2 || len(store.Transactions(k, false)) != 3 {
		t.Fatal("timeliness strict boundary")
	}
	conflict := s.Clone()
	conflict.Message.Transactions = [][]byte{{4}}
	sign(&conflict, v)
	if err := store.Gossip(encoded(t, conflict), v); err != nil {
		t.Fatal("late conflict", err)
	}
	if len(store.Transactions(k, true)) != 0 {
		t.Fatal("equivocation did not remove timely constraint")
	}
	first, second, ok := store.Evidence(k, 7)
	if !ok || second == nil || equal(first.Message, second.Message) {
		t.Fatal("evidence missing")
	}
	second.Message.Transactions[0][0] = 9
	_, second, _ = store.Evidence(k, 7)
	if second.Message.Transactions[0][0] != 4 {
		t.Fatal("evidence alias")
	}
	other := s.Clone()
	other.Message.DependentRoot[0] = 1
	sign(&other, v)
	if err := store.Gossip(encoded(t, other), v); err != nil {
		t.Fatal("branch conflation", err)
	}
	if len(store.Transactions(key(other.Message), false)) != 2 {
		t.Fatal("branch lost")
	}
	if err := store.Gossip(encoded(t, s), v); err == nil {
		t.Fatal("third valid message accepted")
	}
	bits := store.Bits(k, v.state.Committee, false)
	if bits.Has(7) || !bits.Has(8) || !bits.Includes(store.Bits(k, v.state.Committee, true)) {
		t.Fatal("bid view bits")
	}
}
func TestGroupEHezeRequestsRetentionAndReauthenticatedRestart(t *testing.T) {
	t.Run("expired-cache-needs-no-pruned-state", func(t *testing.T) {
		v, signed := fixture(t)
		s := NewStore()
		if err := s.Import(signed, v); err != nil {
			t.Fatal(err)
		}
		data, err := s.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		v.clock.Slot += MinRequestSlots + 2
		v.state.StateAvailable = false
		restored, err := Restore(data, v)
		if err != nil || len(restored.Transactions(key(signed.Message), false)) != 0 {
			t.Fatal("expired cache blocked restart", err)
		}
		v.clock.Slot = signed.Message.Slot + MinRequestSlots + 1
		if _, err := Restore(data, v); err == nil {
			t.Fatal("required retained state unavailable but accepted")
		}
		v.clock.Slot = signed.Message.Slot - 1
		v.state.StateAvailable = true
		if _, err := Restore(data, v); err == nil {
			t.Fatal("future persisted list accepted")
		}
	})
	v, s := fixture(t)
	store := NewStore()
	if err := store.Import(s, v); err != nil {
		t.Fatal(err)
	}
	var bits Bits
	bits.Set(7)
	req := Request{100, s.Message.DependentRoot, bits}
	got, e := store.ByIndices(req, v.state.Committee, 101, 0)
	if e != nil || len(got) != 1 {
		t.Fatal("previous-slot request", e)
	}
	if _, e = store.ByIndices(req, v.state.Committee, 102, 0); !errors.Is(e, ErrResourceUnavailable) {
		t.Fatal("old request", e)
	}
	data, e := store.MarshalBinary()
	if e != nil {
		t.Fatal(e)
	}
	restored, e := Restore(data, v)
	if e != nil || len(restored.Transactions(key(s.Message), true)) != 2 {
		t.Fatal("restart", e)
	}
	wrong := *v
	wrong.state = v.state
	wrong.state.ForkVersion[0]++
	if _, e = Restore(data, &wrong); e == nil {
		t.Fatal("restart failed to reauthenticate")
	}
	if store.Prune(102) != 0 || len(store.Transactions(key(s.Message), true)) != 2 {
		t.Fatal("pruned extra decision slot")
	}
	if store.Prune(103) != 1 {
		t.Fatal("failed to prune expired member")
	}
	v.clock.Slot = 101
	if err := NewStore().Gossip(encoded(t, s), v); err == nil {
		t.Fatal("gossip accepted previous slot")
	}
	if err := NewStore().Import(s, v); err != nil {
		t.Fatal("import rejected previous slot", err)
	}
}
func TestGroupEHezeConcurrentQuotaAndCommittee(t *testing.T) {
	v, s := fixture(t)
	store := NewStore()
	raw := encoded(t, s)
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _ = store.Gossip(raw, v) }()
	}
	wg.Wait()
	if store.gossipCount(s.Message) != 2 {
		t.Fatal("non-atomic message quota")
	}
	c, e := SelectCommittee([][]uint64{{4, 8}, {9}})
	if e != nil || c[0] != 4 || c[1] != 8 || c[15] != 4 {
		t.Fatal("ordered/repeated committee", e)
	}
}
func TestGroupEHezePayloadForkchoiceAndOptimisticTransitions(t *testing.T) {
	a, b, c := types.Hash{31: 1}, types.Hash{31: 2}, types.Hash{31: 3}
	p := NewPayloadDecisions(a)
	yes, no := true, false
	if err := p.Record(b, NotValidated, nil, true); err != nil {
		t.Fatal(err)
	}
	if ok, _ := p.IsSatisfied(b); !ok {
		t.Fatal("optimistic default")
	}
	if err := p.Record(c, Valid, &no, true); err != nil {
		t.Fatal(err)
	}
	if ok, _ := p.IsSatisfied(c); ok {
		t.Fatal("unsatisfied valid payload")
	}
	ctx := ExtensionContext{PayloadSlot: 100, CurrentSlot: 101, PayloadTimely: true, PayloadDataAvailable: true}
	if !p.ShouldExtend(b, ctx) || p.ShouldExtend(c, ctx) {
		t.Fatal("fork-choice did not separate execution and satisfaction")
	}
	if err := p.Record(b, Valid, &no, true); err != nil {
		t.Fatal(err)
	}
	if ok, _ := p.IsSatisfied(b); ok {
		t.Fatal("optimistic transition not recorded")
	}
	if ok, _ := p.IsSatisfied(a); !ok {
		t.Fatal("ancestor overwritten")
	}
	if err := p.Record(a, Valid, &no, true); err != nil {
		t.Fatal(err)
	}
	if ok, _ := p.IsSatisfied(a); !ok {
		t.Fatal("known valid satisfaction overwritten after pruning")
	}
	if err := p.Record(c, Invalidated, nil, true); err != nil {
		t.Fatal(err)
	}
	if err := p.Record(types.Hash{31: 4}, Valid, nil, true); err == nil {
		t.Fatal("missing valid Boolean")
	}
	if err := p.Record(types.Hash{31: 5}, Valid, &yes, false); err != nil {
		t.Fatal(err)
	}
	if ok, _ := p.IsSatisfied(types.Hash{31: 5}); ok {
		t.Fatal("unavailable payload extended")
	}
}
