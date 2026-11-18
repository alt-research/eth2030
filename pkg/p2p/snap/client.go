package snap

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/eth2030/eth2030/bal"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
	"github.com/eth2030/eth2030/p2p/wire"
	"github.com/eth2030/eth2030/rlp"
	"github.com/eth2030/eth2030/sync/statesync"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
	gethtrie "github.com/ethereum/go-ethereum/trie"
)

type pendingResponse struct {
	code uint64
	ch   chan wire.Msg
}

// Client is a negotiated snap peer with response correlation. Verification is bound to
// the requested pivot root, not to an untrusted server completion flag.
type Client struct {
	id        string
	version   uint
	transport wire.Transport
	seq       atomic.Uint64
	mu        sync.Mutex
	pending   map[uint64]pendingResponse
	done      chan struct{}
}

func (c *Client) ID() string { return c.id }

func (c *Client) Version() uint {
	if c.version == 0 {
		return ProtocolVersion
	}
	return c.version
}

// WithContext creates a request-scoped adapter sharing the peer connection.
func (c *Client) WithContext(ctx context.Context) *VerifiedPeer {
	return &VerifiedPeer{Client: c, ctx: ctx, state: &verifiedState{storageRoots: make(map[types.Hash]types.Hash)}}
}

type VerifiedPeer struct {
	*Client
	ctx   context.Context
	state *verifiedState
}

// verifiedState is scoped to a single pivot attempt. Only account ranges
// authenticated against stateRoot may add storage roots.
type verifiedState struct {
	mu           sync.Mutex
	storageRoots map[types.Hash]types.Hash
	stateRoot    types.Hash
	rootSet      bool
}

// VerifiedPeers shares proof-verified account roots across transports for one
// pivot attempt, so another peer can continue after a transport fails.
func VerifiedPeers(ctx context.Context, clients []*Client) []*VerifiedPeer {
	state := &verifiedState{storageRoots: make(map[types.Hash]types.Hash)}
	peers := make([]*VerifiedPeer, len(clients))
	for i, client := range clients {
		peers[i] = &VerifiedPeer{Client: client, ctx: ctx, state: state}
	}
	return peers
}

func (c *Client) request(ctx context.Context, code uint64, packet func(uint64) any, out any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	stopClose := context.AfterFunc(ctx, func() { _ = c.transport.Close() })
	defer stopClose()
	id := c.seq.Add(1)
	ch := make(chan wire.Msg, 1)
	c.mu.Lock()
	c.pending[id] = pendingResponse{code + 1, ch}
	c.mu.Unlock()
	defer func() { c.mu.Lock(); delete(c.pending, id); c.mu.Unlock() }()
	err := sendSnapMsg(c.transport, code, packet(id))
	stopClose()
	if err != nil {
		return err
	}
	select {
	case <-c.done:
		return fmt.Errorf("snap: peer disconnected")
	case <-ctx.Done():
		return ctx.Err()
	case msg := <-ch:
		if decoder, ok := out.(interface{ DecodeRLP([]byte) error }); ok {
			return decoder.DecodeRLP(msg.Payload)
		}
		return rlp.DecodeBytes(msg.Payload, out)
	}
}

func (c *Client) deliver(msg wire.Msg) error {
	limit := HardResponseLimit
	if c.Version() >= ProtocolVersion2 && msg.Code == BlockAccessListsMsg {
		limit = BALHardResponseLimit
	}
	if len(msg.Payload) > limit {
		return ErrRequestTooLarge
	}
	// Every response starts with its request ID.
	content, _, err := gethrlp.SplitList(msg.Payload)
	if err != nil {
		return err
	}
	id, _, err := gethrlp.SplitUint64(content)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if req, ok := c.pending[id]; ok && req.code == msg.Code {
		select {
		case req.ch <- msg:
		default:
		}
	}
	return nil
}

func verifyRange(root, origin types.Hash, keys, values, proof [][]byte) (bool, error) {
	if root == types.EmptyRootHash && len(keys) == 0 && len(proof) == 0 {
		return false, nil
	}
	if len(proof) == 0 {
		return gethtrie.VerifyRangeProof(gethcommon.Hash(root), origin[:], keys, values, nil)
	}
	db := memorydb.New()
	for _, node := range proof {
		if err := db.Put(crypto.Keccak256(node), node); err != nil {
			return false, err
		}
	}
	return gethtrie.VerifyRangeProof(gethcommon.Hash(root), origin[:], keys, values, db)
}

// A zero-width query needs an inclusion/absence proof, not a proof that every
// leaf to its right is exhausted. Range verification rejects valid bounded
// absence when another leaf exists beyond the requested point.
func verifyPoint(root, key types.Hash, keys, values, proof [][]byte) (int, error) {
	var actual []byte
	if root != types.EmptyRootHash {
		db := memorydb.New()
		for _, node := range proof {
			if err := db.Put(crypto.Keccak256(node), node); err != nil {
				return -1, err
			}
		}
		var err error
		actual, err = gethtrie.VerifyProof(gethcommon.Hash(root), key[:], db)
		if err != nil {
			return -1, err
		}
	}
	index := -1
	for i, candidate := range keys {
		if bytes.Equal(candidate, key[:]) {
			if index >= 0 {
				return -1, fmt.Errorf("duplicate snap point value")
			}
			index = i
		}
	}
	if actual == nil {
		if index >= 0 {
			return -1, fmt.Errorf("snap point contradicts authenticated absence")
		}
		return -1, nil
	}
	if index < 0 || !bytes.Equal(actual, values[index]) {
		return -1, fmt.Errorf("snap point value does not match proof")
	}
	return index, nil
}

func (p *VerifiedPeer) RequestAccountRange(req statesync.AccountRangeRequest) (*statesync.AccountRangeResponse, error) {
	p.state.mu.Lock()
	mismatch := p.state.rootSet && p.state.stateRoot != req.Root
	p.state.mu.Unlock()
	if mismatch {
		return nil, fmt.Errorf("snap: peer adapter pivot changed")
	}
	var response AccountRangePacket
	if err := p.request(p.ctx, GetAccountRangeMsg, func(id uint64) any { return &GetAccountRangePacket{id, req.Root, req.Origin, req.Limit, req.Bytes} }, &response); err != nil {
		return nil, err
	}
	out := &statesync.AccountRangeResponse{ID: req.ID, Proof: response.Proof}
	var keys, values [][]byte
	roots := make(map[types.Hash]types.Hash)
	for _, item := range response.Accounts {
		full, err := gethtypes.FullAccountRLP(item.Body)
		if err != nil {
			return nil, err
		}
		acc, err := gethtypes.FullAccount(item.Body)
		if err != nil {
			return nil, err
		}
		keys = append(keys, item.Hash.Bytes())
		values = append(values, full)
		out.Accounts = append(out.Accounts, statesync.AccountData{Hash: item.Hash, Nonce: acc.Nonce, Balance: acc.Balance.ToBig(), Root: types.Hash(acc.Root), CodeHash: types.BytesToHash(acc.CodeHash)})
		roots[item.Hash] = types.Hash(acc.Root)
	}
	var more bool
	var err error
	if req.Origin == req.Limit {
		var index int
		index, err = verifyPoint(req.Root, req.Origin, keys, values, response.Proof)
		roots = make(map[types.Hash]types.Hash)
		if index < 0 {
			out.Accounts = nil
		} else {
			out.Accounts = []statesync.AccountData{out.Accounts[index]}
			roots[out.Accounts[0].Hash] = out.Accounts[0].Root
		}
	} else {
		more, err = verifyRange(req.Root, req.Origin, keys, values, response.Proof)
	}
	if err != nil {
		return nil, fmt.Errorf("snap: account range proof: %w", err)
	}
	// The boundary leaf may lie beyond Limit and is needed by the range proof.
	// It is not a leaf of this partition and must not enqueue duplicate storage.
	filtered := make([]statesync.AccountData, 0, len(out.Accounts))
	roots = make(map[types.Hash]types.Hash)
	for _, account := range out.Accounts {
		if bytes.Compare(account.Hash[:], req.Origin[:]) >= 0 && bytes.Compare(account.Hash[:], req.Limit[:]) <= 0 {
			filtered = append(filtered, account)
			roots[account.Hash] = account.Root
		}
	}
	out.Accounts = filtered
	p.state.mu.Lock()
	defer p.state.mu.Unlock()
	if p.state.rootSet && p.state.stateRoot != req.Root {
		return nil, fmt.Errorf("snap: peer adapter pivot changed")
	}
	p.state.stateRoot, p.state.rootSet = req.Root, true
	for key, root := range roots {
		p.state.storageRoots[key] = root
	}
	out.More = more && (len(keys) == 0 || bytes.Compare(keys[len(keys)-1], req.Limit[:]) < 0)
	return out, nil
}

func (p *VerifiedPeer) RequestStorageRange(req statesync.StorageRangeRequest) (*statesync.StorageRangeResponse, error) {
	if len(req.Accounts) != 1 {
		return nil, fmt.Errorf("snap: verified storage requests require one account")
	}
	account := req.Accounts[0]
	p.state.mu.Lock()
	root, ok := p.state.storageRoots[account]
	ok = ok && p.state.rootSet && p.state.stateRoot == req.Root
	p.state.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("snap: storage account not verified")
	}
	var response StorageRangesPacket
	if err := p.request(p.ctx, GetStorageRangesMsg, func(id uint64) any {
		return &GetStorageRangesPacket{id, req.Root, req.Accounts, req.Origin[:], req.Limit[:], req.Bytes}
	}, &response); err != nil {
		return nil, err
	}
	if len(response.Slots) != 1 {
		return nil, fmt.Errorf("snap: missing storage range")
	}
	out := &statesync.StorageRangeResponse{ID: req.ID, Proof: response.Proof}
	var keys, values [][]byte
	for _, slot := range response.Slots[0] {
		keys = append(keys, slot.Hash.Bytes())
		values = append(values, slot.Body)
		out.Slots = append(out.Slots, statesync.StorageData{AccountHash: account, SlotHash: slot.Hash, Value: slot.Body})
	}
	var more bool
	var err error
	if req.Origin == req.Limit {
		var index int
		index, err = verifyPoint(root, req.Origin, keys, values, response.Proof)
		if index < 0 {
			out.Slots = nil
		} else {
			out.Slots = []statesync.StorageData{out.Slots[index]}
		}
	} else {
		more, err = verifyRange(root, req.Origin, keys, values, response.Proof)
	}
	if err != nil {
		return nil, fmt.Errorf("snap: storage range proof: %w", err)
	}
	out.More = more && (len(keys) == 0 || bytes.Compare(keys[len(keys)-1], req.Limit[:]) < 0)
	return out, nil
}

func (p *VerifiedPeer) RequestBytecodes(req statesync.BytecodeRequest) (*statesync.BytecodeResponse, error) {
	var response ByteCodesPacket
	if err := p.request(p.ctx, GetByteCodesMsg, func(id uint64) any { return &GetByteCodesPacket{id, req.Hashes, SoftResponseLimit} }, &response); err != nil {
		return nil, err
	}
	allowed := make(map[types.Hash]int)
	for i, hash := range req.Hashes {
		allowed[hash] = i
	}
	previous := -1
	out := &statesync.BytecodeResponse{ID: req.ID}
	for _, code := range response.Codes {
		hash := crypto.Keccak256Hash(code)
		index, ok := allowed[hash]
		if !ok || index <= previous {
			return nil, fmt.Errorf("snap: unrequested bytecode")
		}
		delete(allowed, hash)
		previous = index
		out.Codes = append(out.Codes, statesync.BytecodeData{Hash: hash, Code: code})
	}
	return out, nil
}

func (p *VerifiedPeer) RequestTrieNodes(root types.Hash, paths [][]byte) ([][]byte, error) {
	if p.Version() >= ProtocolVersion2 {
		return nil, fmt.Errorf("snap/2 removed trie-node healing")
	}
	sets := make([]TrieNodePathSet, len(paths))
	for i, path := range paths {
		if len(path) > 32 {
			sets[i] = TrieNodePathSet{path[:32], path[32:]}
		} else {
			sets[i] = TrieNodePathSet{path}
		}
	}
	var response TrieNodesPacket
	err := p.request(p.ctx, GetTrieNodesMsg, func(id uint64) any { return &GetTrieNodesPacket{id, root, sets, SoftResponseLimit} }, &response)
	return response.Nodes, err
}

// RequestBlockAccessLists verifies each positional response against the exact
// requested header. Missing data is nil; an authenticated empty BAL is c0.
func (c *Client) RequestBlockAccessLists(ctx context.Context, headers []*types.Header, budget uint64) ([][]byte, error) {
	if c.Version() < ProtocolVersion2 {
		return nil, fmt.Errorf("snap peer does not support BAL catch-up")
	}
	if len(headers) > MaxBlockAccessLists {
		return nil, ErrRequestTooLarge
	}
	hashes := make([]types.Hash, len(headers))
	for i, header := range headers {
		if header == nil {
			return nil, fmt.Errorf("missing BAL header")
		}
		hashes[i] = header.Hash()
	}
	var response BlockAccessListsPacket
	err := c.request(ctx, GetBlockAccessListsMsg, func(id uint64) any { return &GetBlockAccessListsPacket{id, hashes, budget} }, &response)
	if err != nil {
		return nil, err
	}
	if len(response.Lists) > len(headers) {
		return nil, fmt.Errorf("snap/2 unsolicited BAL tail")
	}
	for i, data := range response.Lists {
		if data == nil {
			continue
		}
		if _, err := bal.DecodeForHeader(data, headers[i]); err != nil {
			return nil, err
		}
	}
	return response.Lists, nil
}

func (h *ServerHandler) Clients() []*Client {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]*Client, 0, len(h.clients))
	for _, client := range h.clients {
		out = append(out, client)
	}
	return out
}
