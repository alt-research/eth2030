package state

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
	"github.com/eth2030/eth2030/rlp"
	nativeTrie "github.com/eth2030/eth2030/trie"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethtrie "github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	triedatabase "github.com/ethereum/go-ethereum/triedb/database"
)

var ErrNoHistoryTrie = errors.New("state has no history trie")

// historyView is immutable. Pending nodes remain readable until the caller's
// atomic block batch succeeds; preparing or copying a view publishes nothing.
type historyView struct {
	db           rawdb.Database
	root         types.Hash
	nodes, codes map[types.Hash][]byte
	parent       *historyView
}

type historyTrieReader struct {
	view *historyView
	ctx  context.Context
}

func (r historyTrieReader) NodeReader(gethcommon.Hash) (triedatabase.NodeReader, error) {
	return r, nil
}
func (r historyTrieReader) Node(_ gethcommon.Hash, _ []byte, hash gethcommon.Hash) ([]byte, error) {
	return r.view.node(r.ctx, types.Hash(hash))
}
func historyContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func (v *historyView) lookup(hash types.Hash, code bool) ([]byte, error) {
	for p := v; p != nil; p = p.parent {
		values := p.nodes
		if code {
			values = p.codes
		}
		if data, ok := values[hash]; ok {
			return data, nil
		}
	}
	key := historyNodeKey(hash)
	if code {
		key = append([]byte("snap-code/"), hash[:]...)
	}
	return v.db.Get(key)
}
func (v *historyView) data(ctx context.Context, hash types.Hash, code bool) ([]byte, error) {
	ctx = historyContext(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	data, err := v.lookup(hash, code)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, fmt.Errorf("state history: missing/unreadable content %s: %v", hash, err)
	}
	if crypto.Keccak256Hash(data) != hash {
		return nil, fmt.Errorf("state history: corrupt content %s", hash)
	}
	return data, nil
}
func (v *historyView) node(ctx context.Context, hash types.Hash) ([]byte, error) {
	return v.data(ctx, hash, false)
}
func (v *historyView) trie(ctx context.Context, root types.Hash) (*gethtrie.Trie, error) {
	return gethtrie.New(gethtrie.TrieID(gethcommon.Hash(root)), historyTrieReader{v, historyContext(ctx)})
}
func (v *historyView) published() (*historyView, error) {
	available, err := HistoryStateAvailable(v.db, v.root)
	if err != nil {
		return nil, err
	}
	if available {
		return &historyView{db: v.db, root: v.root}, nil
	}
	return v, nil
}
func decodeHistoryAccount(body []byte) (rlpAccount, error) {
	var a rlpAccount
	if err := rlp.DecodeBytes(body, &a); err != nil {
		return a, err
	}
	if a.Balance == nil || a.Balance.Sign() < 0 || a.Balance.BitLen() > 256 || len(a.Root) != 32 || len(a.CodeHash) != 32 {
		return a, fmt.Errorf("state history: invalid account")
	}
	return a, nil
}
func (v *historyView) account(ctx context.Context, address types.Address) (*rlpAccount, error) {
	tr, err := v.trie(ctx, v.root)
	if err != nil {
		return nil, err
	}
	body, err := tr.Get(crypto.Keccak256(address[:]))
	if err != nil || len(body) == 0 {
		return nil, err
	}
	a, err := decodeHistoryAccount(body)
	return &a, err
}
func (v *historyView) storage(ctx context.Context, root types.Hash, key types.Hash) (types.Hash, error) {
	tr, err := v.trie(ctx, root)
	if err != nil {
		return types.Hash{}, err
	}
	body, err := tr.Get(crypto.Keccak256(key[:]))
	if err != nil || len(body) == 0 {
		return types.Hash{}, err
	}
	var value []byte
	if err := rlp.DecodeBytes(body, &value); err != nil {
		return types.Hash{}, err
	}
	if len(value) == 0 || len(value) > 32 || value[0] == 0 {
		return types.Hash{}, fmt.Errorf("state history: noncanonical storage")
	}
	return types.BytesToHash(value), nil
}
func collectHistoryNodes(out map[types.Hash][]byte, set *trienode.NodeSet) {
	if set == nil {
		return
	}
	for _, n := range set.Nodes {
		if !n.IsDeleted() {
			out[types.Hash(n.Hash)] = bytes.Clone(n.Blob)
		}
	}
}

func prepareHistory(ctx context.Context, base *historyView, mem *MemoryStateDB) (*historyView, error) {
	ctx = historyContext(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var err error
	base, err = base.published()
	if err != nil {
		return nil, err
	}
	if len(mem.historyDirty) == 0 {
		return base, nil
	}
	accounts, err := base.trie(ctx, base.root)
	if err != nil {
		return nil, err
	}
	next := &historyView{db: base.db, parent: base, nodes: make(map[types.Hash][]byte), codes: make(map[types.Hash][]byte)}
	for address := range mem.historyDirty {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		key := crypto.Keccak256(address[:])
		obj := mem.stateObjects[address]
		if obj == nil || obj.selfDestructed || obj.deleted {
			if err := accounts.Delete(key); err != nil {
				return nil, err
			}
			continue
		}
		storageRoot := obj.dbStorageRoot
		if storageRoot == (types.Hash{}) {
			storageRoot = types.EmptyRootHash
		}
		slots, err := base.trie(ctx, storageRoot)
		if err != nil {
			return nil, err
		}
		changed := mem.historySlots[address]
		if obj.dbStorageRoot == (types.Hash{}) {
			changed = make(map[types.Hash]struct{}, len(obj.committedStorage)+len(obj.dirtyStorage))
			for k := range obj.committedStorage {
				changed[k] = struct{}{}
			}
			for k := range obj.dirtyStorage {
				changed[k] = struct{}{}
			}
		}
		for slot := range changed {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			value := mem.GetState(address, slot)
			hashed := crypto.Keccak256(slot[:])
			if value == (types.Hash{}) {
				err = slots.Delete(hashed)
			} else {
				var body []byte
				body, err = rlp.EncodeToBytes(bytes.TrimLeft(value[:], "\x00"))
				if err == nil {
					err = slots.Update(hashed, body)
				}
			}
			if err != nil {
				return nil, err
			}
		}
		root, set := slots.Commit(false)
		collectHistoryNodes(next.nodes, set)
		codeHash := types.BytesToHash(obj.account.CodeHash)
		if len(obj.account.CodeHash) == 0 {
			codeHash = types.EmptyCodeHash
		}
		if codeHash != types.EmptyCodeHash {
			if len(obj.code) == 0 {
				body, err := accounts.Get(key)
				if err != nil {
					return nil, err
				}
				previous, err := decodeHistoryAccount(body)
				if err != nil || types.BytesToHash(previous.CodeHash) != codeHash {
					return nil, fmt.Errorf("state history: missing modified code")
				}
			} else {
				if crypto.Keccak256Hash(obj.code) != codeHash {
					return nil, fmt.Errorf("state history: invalid modified code")
				}
				next.codes[codeHash] = bytes.Clone(obj.code)
			}
		}
		body, err := rlp.EncodeToBytes(rlpAccount{Nonce: obj.account.Nonce, Balance: obj.account.Balance, Root: root[:], CodeHash: codeHash[:]})
		if err != nil {
			return nil, err
		}
		if err := accounts.Update(key, body); err != nil {
			return nil, err
		}
	}
	root, set := accounts.Commit(false)
	collectHistoryNodes(next.nodes, set)
	next.root = types.Hash(root)
	return next, nil
}

func flushHistoryView(db rawdb.KeyValueReader, batch rawdb.Batch, v *historyView) error {
	for p := v; p != nil; p = p.parent {
		present, err := HistoryStateAvailable(db, p.root)
		if err != nil {
			return err
		}
		if present {
			break
		}
		for hash, data := range p.nodes {
			if crypto.Keccak256Hash(data) != hash {
				return fmt.Errorf("state history: invalid prepared node")
			}
			if err := immutableHistoryPut(db, batch, historyNodeKey(hash), data); err != nil {
				return err
			}
		}
		for hash, code := range p.codes {
			if crypto.Keccak256Hash(code) != hash {
				return fmt.Errorf("state history: invalid prepared code")
			}
			if err := immutableHistoryPut(db, batch, append([]byte("snap-code/"), hash[:]...), code); err != nil {
				return err
			}
		}
	}
	return batch.Put(historyRootKey(v.root), []byte{1})
}

func (m *MemoryStateDB) markHistory(address types.Address) {
	if m.historyDirty == nil {
		m.historyDirty = make(map[types.Address]struct{})
	}
	m.historyDirty[address] = struct{}{}
	m.historyRevision++
}
func (m *MemoryStateDB) markHistorySlot(address types.Address, key types.Hash) {
	m.markHistory(address)
	if m.historySlots == nil {
		m.historySlots = make(map[types.Address]map[types.Hash]struct{})
	}
	if m.historySlots[address] == nil {
		m.historySlots[address] = make(map[types.Hash]struct{})
	}
	m.historySlots[address][key] = struct{}{}
}
func (m *MemoryStateDB) bindHistory(v *historyView) error {
	// Only the modified working set is visited after the one-time full import.
	for address := range m.historyDirty {
		if obj := m.stateObjects[address]; obj != nil && !obj.selfDestructed && !obj.deleted {
			account, err := v.account(context.Background(), address)
			if err != nil {
				return err
			}
			if account != nil {
				obj.dbStorageRoot = types.BytesToHash(account.Root)
			}
		}
	}
	m.historyBase = v
	m.historyDirty = nil
	m.historySlots = nil
	m.historyRevision++
	return nil
}

// ReadError surfaces a lazy storage failure rather than serving a fabricated
// zero balance/slot or accepting an execution result derived from missing data.
func ReadError(st any) error {
	if reader, ok := st.(interface{ Error() error }); ok {
		return reader.Error()
	}
	return nil
}
func SetReadContext(st any, ctx context.Context) {
	if reader, ok := st.(interface{ SetReadContext(context.Context) }); ok {
		reader.SetReadContext(historyContext(ctx))
	}
}
func (s *SnapStateDB) Error() error                       { return s.historyErr }
func (s *SnapStateDB) SetReadContext(ctx context.Context) { s.historyCtx = historyContext(ctx) }
func (s *SnapStateDB) historyFailure(err error) {
	if s.historyErr == nil {
		s.historyErr = err
	}
}
func (s *SnapStateDB) preparedHistory() (*historyView, error) {
	if s.historyErr != nil {
		return nil, s.historyErr
	}
	if s.historyPrepared != nil && s.historyRevision == s.memory.historyRevision {
		return s.historyPrepared, nil
	}
	view, err := prepareHistory(s.historyCtx, s.history, s.memory)
	if err != nil {
		s.historyFailure(err)
		return nil, err
	}
	s.historyPrepared = view
	s.historyRevision = s.memory.historyRevision
	return view, nil
}

// OpenTrieHistory opens an authenticated root and resolves only accessed trie
// paths. AuditTrieHistory remains the explicit complete-availability check.
func OpenTrieHistory(ctx context.Context, db rawdb.Database, root types.Hash) (*SnapStateDB, error) {
	if err := historyContext(ctx).Err(); err != nil {
		return nil, err
	}
	present, err := HistoryStateAvailable(db, root)
	if err != nil {
		return nil, err
	}
	if !present {
		return nil, rawdb.ErrNotFound
	}
	if root == (types.Hash{}) {
		return nil, fmt.Errorf("state history: zero state root")
	}
	view := &historyView{db: db, root: root}
	if _, err := view.trie(ctx, root); err != nil {
		return nil, err
	}
	return &SnapStateDB{memory: NewMemoryStateDB(), history: view, historyCtx: historyContext(ctx)}, nil
}

// historyOverlayDB is used only for explicit full export/audit, never point
// reads or incremental commits. It also sees prepared-but-unpublished nodes.
type historyOverlayDB struct {
	rawdb.Database
	view *historyView
}

func (d historyOverlayDB) Get(key []byte) ([]byte, error) {
	if bytes.HasPrefix(key, []byte("mpt-history-node/")) && len(key) == len("mpt-history-node/")+32 {
		value, err := d.view.lookup(types.BytesToHash(key[len("mpt-history-node/"):]), false)
		return bytes.Clone(value), err
	}
	if bytes.HasPrefix(key, []byte("snap-code/")) && len(key) == len("snap-code/")+32 {
		value, err := d.view.lookup(types.BytesToHash(key[len("snap-code/"):]), true)
		return bytes.Clone(value), err
	}
	if bytes.Equal(key, historyRootKey(d.view.root)) {
		return []byte{1}, nil
	}
	return d.Database.Get(key)
}
func (s *SnapStateDB) historySnapshot() (*StateSnapshot, error) {
	v, err := s.preparedHistory()
	if err != nil {
		return nil, err
	}
	full, err := openFullTrieHistory(historyContext(s.historyCtx), historyOverlayDB{Database: v.db, view: v}, v.root)
	if err != nil {
		return nil, err
	}
	return full.base, nil
}

type historyProofNodes [][]byte

func (p *historyProofNodes) Put(_ []byte, value []byte) error {
	*p = append(*p, bytes.Clone(value))
	return nil
}
func (p *historyProofNodes) Delete([]byte) error { return fmt.Errorf("proof deletion is unsupported") }
func (s *SnapStateDB) HistoricalProof(address types.Address, keys []types.Hash) (*nativeTrie.AccountProof, error) {
	if s.history == nil {
		return nil, ErrNoHistoryTrie
	}
	v, err := s.preparedHistory()
	if err != nil {
		return nil, err
	}
	tr, err := v.trie(s.historyCtx, v.root)
	if err != nil {
		return nil, err
	}
	hashed := crypto.Keccak256(address[:])
	body, err := tr.Get(hashed)
	if err != nil {
		return nil, err
	}
	result := &nativeTrie.AccountProof{Address: address, Balance: new(big.Int), CodeHash: types.EmptyCodeHash, StorageHash: types.EmptyRootHash}
	if len(body) > 0 {
		a, err := decodeHistoryAccount(body)
		if err != nil {
			return nil, err
		}
		result.Nonce = a.Nonce
		result.Balance = a.Balance
		result.CodeHash = types.BytesToHash(a.CodeHash)
		result.StorageHash = types.BytesToHash(a.Root)
	}
	var proof historyProofNodes
	if err := tr.Prove(hashed, &proof); err != nil {
		return nil, err
	}
	result.AccountProof = proof
	storage, err := v.trie(s.historyCtx, result.StorageHash)
	if err != nil {
		return nil, err
	}
	for _, key := range keys {
		if err := historyContext(s.historyCtx).Err(); err != nil {
			return nil, err
		}
		hashed := crypto.Keccak256(key[:])
		body, err := storage.Get(hashed)
		if err != nil {
			return nil, err
		}
		value := new(big.Int)
		if len(body) > 0 {
			var raw []byte
			if err := rlp.DecodeBytes(body, &raw); err != nil {
				return nil, err
			}
			if len(raw) == 0 || len(raw) > 32 || raw[0] == 0 {
				return nil, fmt.Errorf("state history: invalid proof storage")
			}
			value.SetBytes(raw)
		}
		var nodes historyProofNodes
		if err := storage.Prove(hashed, &nodes); err != nil {
			return nil, err
		}
		result.StorageProof = append(result.StorageProof, nativeTrie.StorageProof{Key: key, Value: value, Proof: nodes})
	}
	return result, nil
}

func (s *SnapStateDB) loadHistoryCode(address types.Address) {
	if s.history == nil {
		return
	}
	obj := s.memory.stateObjects[address]
	if obj == nil || obj.deleted || len(obj.code) != 0 {
		return
	}
	hash := types.BytesToHash(obj.account.CodeHash)
	if hash == types.EmptyCodeHash || hash == (types.Hash{}) {
		return
	}
	code, err := s.history.data(s.historyCtx, hash, true)
	if err != nil {
		s.historyFailure(err)
		return
	}
	obj.code = bytes.Clone(code)
}
