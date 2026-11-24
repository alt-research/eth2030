package node

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	nativechain "github.com/eth2030/eth2030/core/chain"
	nativeconfig "github.com/eth2030/eth2030/core/config"
	nativerawdb "github.com/eth2030/eth2030/core/rawdb"
	nativestate "github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	nativeeth "github.com/eth2030/eth2030/eth"
	nativep2p "github.com/eth2030/eth2030/p2p"
	nativesnap "github.com/eth2030/eth2030/p2p/snap"
	nativetxpool "github.com/eth2030/eth2030/txpool"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	gethcore "github.com/ethereum/go-ethereum/core"
	gethrawdb "github.com/ethereum/go-ethereum/core/rawdb"
	gethbal "github.com/ethereum/go-ethereum/core/types/bal"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	getheth "github.com/ethereum/go-ethereum/eth/protocols/eth"
	gethsnap "github.com/ethereum/go-ethereum/eth/protocols/snap"
	"github.com/ethereum/go-ethereum/ethdb"
	gethp2p "github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/params"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

const independentBALNetwork = 1337

type gethBALProvider struct {
	chain    *gethcore.BlockChain
	db       ethdb.Database
	mu       sync.Mutex
	status   map[enode.ID]chan struct{}
	traceMu  sync.Mutex
	traceDir string
	traces   []map[string]any
}

func (p *gethBALProvider) signal(id enode.ID) chan struct{} {
	p.mu.Lock()
	defer p.mu.Unlock()
	c := p.status[id]
	if c == nil {
		c = make(chan struct{})
		p.status[id] = c
	}
	return c
}

type independentBALWire struct {
	gethp2p.MsgReadWriter
	provider *gethBALProvider
}

func (p *gethBALProvider) record(direction string, code uint64, data []byte) {
	p.traceMu.Lock()
	defer p.traceMu.Unlock()
	name := fmt.Sprintf("%03d-%s-%02x.rlp.gz", len(p.traces), direction, code)
	file, err := os.Create(filepath.Join(p.traceDir, name))
	if err != nil {
		panic(err)
	}
	compressed := gzip.NewWriter(file)
	if _, err = compressed.Write(data); err != nil {
		panic(err)
	}
	if err = compressed.Close(); err != nil {
		panic(err)
	}
	if err = file.Close(); err != nil {
		panic(err)
	}
	digest := sha256.Sum256(data)
	row := map[string]any{"direction": direction, "code": code, "size": len(data), "raw_sha256": fmt.Sprintf("%x", digest), "artifact": name}
	if code == gethsnap.GetAccessListsMsg {
		var packet gethsnap.GetAccessListsPacket
		if gethrlp.DecodeBytes(data, &packet) == nil {
			row["id"] = packet.ID
			row["hashes"] = packet.Hashes
			row["soft_bytes"] = packet.Bytes
		}
	}
	if code == gethsnap.AccessListsMsg {
		var packet gethsnap.AccessListsPacket
		if gethrlp.DecodeBytes(data, &packet) == nil {
			items, err := packet.AccessLists.Items()
			if err == nil {
				row["id"] = packet.ID
				var sizes []int
				var unavailable []int
				var roots []gethcommon.Hash
				for i, item := range items {
					sizes = append(sizes, len(item))
					if bytes.Equal(item, gethrlp.EmptyString) {
						unavailable = append(unavailable, i)
						roots = append(roots, gethcommon.Hash{})
					} else {
						roots = append(roots, gethcrypto.Keccak256Hash(item))
					}
				}
				row["item_sizes"] = sizes
				row["unavailable_positions"] = unavailable
				row["item_roots"] = roots
			}
		}
	}
	p.traces = append(p.traces, row)
}
func (rw independentBALWire) ReadMsg() (gethp2p.Msg, error) {
	message, err := rw.MsgReadWriter.ReadMsg()
	if err != nil {
		return message, err
	}
	data, err := io.ReadAll(message.Payload)
	if err != nil {
		return message, err
	}
	rw.provider.record("request", message.Code, data)
	message.Payload = bytes.NewReader(data)
	return message, nil
}
func (rw independentBALWire) WriteMsg(message gethp2p.Msg) error {
	data, err := io.ReadAll(message.Payload)
	if err != nil {
		return err
	}
	rw.provider.record("response", message.Code, data)
	message.Payload = bytes.NewReader(data)
	return rw.MsgReadWriter.WriteMsg(message)
}

type gethBALSnapBackend struct{ *gethBALProvider }

func (b gethBALSnapBackend) Chain() *gethcore.BlockChain                  { return b.chain }
func (b gethBALSnapBackend) PeerInfo(enode.ID) interface{}                { return nil }
func (b gethBALSnapBackend) Handle(*gethsnap.Peer, gethsnap.Packet) error { return nil }
func (b gethBALSnapBackend) RunPeer(peer *gethsnap.Peer, handler gethsnap.Handler) error {
	select {
	case <-b.signal(peer.Node().ID()):
		return handler(peer)
	case <-time.After(5 * time.Second):
		return fmt.Errorf("independent snap peer lacks validated eth status")
	}
}

type gethBALEthBackend struct{ *gethBALProvider }

func (b gethBALEthBackend) Chain() *gethcore.BlockChain                { return b.chain }
func (b gethBALEthBackend) TxPool() getheth.TxPool                     { return nil }
func (b gethBALEthBackend) BlobPool() getheth.BlobPool                 { return nil }
func (b gethBALEthBackend) AcceptTxs() bool                            { return false }
func (b gethBALEthBackend) PeerInfo(enode.ID) interface{}              { return nil }
func (b gethBALEthBackend) Handle(*getheth.Peer, getheth.Packet) error { return nil }
func (b gethBALEthBackend) RunPeer(peer *getheth.Peer, handler getheth.Handler) error {
	head := b.chain.CurrentBlock()
	if err := peer.Handshake(independentBALNetwork, b.chain, getheth.BlockRangeUpdatePacket{EarliestBlock: 0, LatestBlock: head.Number.Uint64(), LatestBlockHash: head.Hash()}); err != nil {
		return err
	}
	close(b.signal(peer.Node().ID()))
	return handler(peer)
}
func startIndependentGethBALProvider(t *testing.T, label string) (*gethp2p.Server, *gethBALProvider, *gethcore.Genesis) {
	t.Helper()
	db := gethrawdb.NewMemoryDatabase()
	cfg := *params.AllDevChainProtocolChanges
	zero := uint64(0)
	cfg.AmsterdamTime = &zero
	cfg.BogotaTime = nil
	genesis := &gethcore.Genesis{Config: &cfg, GasLimit: 30_000_000, BaseFee: big.NewInt(1), Alloc: gethcore.GenesisAlloc{}}
	chain, err := gethcore.NewBlockChain(db, genesis, ethash.NewFaker(), nil)
	if err != nil {
		t.Fatal(err)
	}
	provider := &gethBALProvider{chain: chain, db: db, status: make(map[enode.ID]chan struct{})}
	root := os.Getenv("ETH2030_INTEROP_OUTPUT")
	if root == "" {
		root = t.TempDir()
	}
	provider.traceDir = filepath.Join(root, strings.ReplaceAll(t.Name(), "/", "-"), label)
	if err := os.MkdirAll(provider.traceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	protocols := getheth.MakeProtocols(gethBALEthBackend{provider}, independentBALNetwork, nil)
	snapshotProtocols := gethsnap.MakeProtocols(gethBALSnapBackend{provider}, true)
	for i := range snapshotProtocols {
		run := snapshotProtocols[i].Run
		snapshotProtocols[i].Run = func(peer *gethp2p.Peer, rw gethp2p.MsgReadWriter) error {
			return run(peer, independentBALWire{rw, provider})
		}
	}
	protocols = append(protocols, snapshotProtocols...)
	key, err := gethcrypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	server := &gethp2p.Server{Config: gethp2p.Config{PrivateKey: key, Name: "independent-geth-v1.17.10", MaxPeers: 8, ListenAddr: "127.0.0.1:0", NoDiscovery: true, Protocols: protocols}}
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		server.Stop()
		chain.Stop()
		db.Close()
		provider.traceMu.Lock()
		defer provider.traceMu.Unlock()
		data, err := json.MarshalIndent(map[string]any{"implementation": "Geth v1.17.10 stock eth and snap protocol handlers and database query", "peer": server.Self().URLv4(), "trace": provider.traces}, "", "  ")
		if err != nil {
			t.Error(err)
			return
		}
		if err := os.WriteFile(filepath.Join(provider.traceDir, "exchanges.json"), append(data, '\n'), 0o644); err != nil {
			t.Error(err)
		}
	})
	return server, provider, genesis
}
func independentGethBAL(t *testing.T, marker byte, count int) []byte {
	t.Helper()
	data := make(gethbal.BlockAccessList, count)
	for i := range data {
		var address gethcommon.Address
		address[0] = marker
		binary.BigEndian.PutUint64(address[12:], uint64(i))
		data[i] = gethbal.AccountAccess{Address: address}
	}
	if len(data) > 0 {
		data[0].StorageReads = []*uint256.Int{uint256.NewInt(1), uint256.NewInt(2)}
	}
	if err := data.Validate(1_000_000_000, 0); err != nil {
		t.Fatal(err)
	}
	encoded, err := gethrlp.EncodeToBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}
func TestIndependentGethSnap2PositionalTruncationAndNativeRecovery(t *testing.T) {
	first, a, genesis := startIndependentGethBALProvider(t, "sparse")
	second, b, _ := startIndependentGethBALProvider(t, "alternate")
	var cfg nativeconfig.ChainConfig
	raw, err := json.Marshal(genesis.Config)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatal(err)
	}
	encoded, err := gethrlp.EncodeToBytes(a.chain.Genesis())
	if err != nil {
		t.Fatal(err)
	}
	anchor, err := types.DecodeBlockRLP(encoded)
	if err != nil {
		t.Fatal(err)
	}
	chain, err := nativechain.NewBlockchain(&cfg, anchor, nativestate.NewMemoryStateDB(), nativerawdb.NewMemoryDB())
	if err != nil {
		t.Fatal(err)
	}
	pool := nativetxpool.New(nativetxpool.DefaultConfig(), chain.State())
	eth := nativeeth.NewHandler(chain, pool, independentBALNetwork)
	snap := nativesnap.NewServerHandler(nil)
	protocols := append(eth.Protocols(), snap.Protocols()...)
	key, err := gethcrypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	server := nativep2p.NewServer(nativep2p.Config{ListenAddr: "127.0.0.1:0", EnableRLPx: true, RLPxPrivateKey: key, Protocols: protocols})
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(server.Stop)
	if err := server.AddPeer(first.Self().URLv4()); err != nil {
		t.Fatal(err)
	}
	if err := server.AddPeer(second.Self().URLv4()); err != nil {
		t.Fatal(err)
	}
	node := &Node{snapHandler: snap, ethHandler: eth}
	deadline := time.Now().Add(8 * time.Second)
	for len(node.snap2Clients()) != 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	clients := node.snap2Clients()
	if len(clients) != 2 {
		t.Fatalf("negotiated and eth-validated snap/2 peers=%d; raw snap=%d eth=%d", len(clients), len(snap.Clients()), eth.Peers().Len())
	}
	// The lists are independently encoded/validated by Geth. Each exceeds the
	// requested 2 MiB soft budget, forcing the real production server to truncate.
	headers := make([]*types.Header, 3)
	lists := make([][]byte, 3)
	for i := range lists {
		lists[i] = independentGethBAL(t, byte(0x10+i), 100_000)
		hash := types.Hash(gethcrypto.Keccak256Hash(lists[i]))
		slot := uint64(i + 1)
		headers[i] = &types.Header{Number: big.NewInt(int64(i + 1)), Time: uint64(i + 1), GasLimit: 1_000_000_000, Difficulty: new(big.Int), BlockAccessListHash: &hash, SlotNumber: &slot}
		if i > 0 {
			headers[i].ParentHash = headers[i-1].Hash()
		}
	}
	store := func(provider *gethBALProvider, i int, data []byte) {
		hash := gethcommon.Hash(headers[i].Hash())
		gethrawdb.WriteHeaderNumber(provider.db, hash, uint64(i+1))
		gethrawdb.WriteAccessListRLP(provider.db, hash, uint64(i+1), data)
	}
	store(a, 1, lists[1])
	store(a, 2, lists[2])
	store(b, 0, lists[0])
	entry, err := nativep2p.ParseEnode(first.Self().URLv4())
	if err != nil {
		t.Fatal(err)
	}
	var firstClient *nativesnap.Client
	for _, client := range clients {
		if client.ID() == string(entry.ID) {
			firstClient = client
		}
	}
	if firstClient == nil {
		t.Fatalf("independent peer identity not mapped: %x", first.Self().ID())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	prefix, err := firstClient.RequestBlockAccessLists(ctx, headers, nativesnap.BALSoftResponseLimit)
	if err != nil {
		t.Fatal(err)
	}
	if len(prefix) != 2 || prefix[0] != nil || !bytes.Equal(prefix[1], lists[1]) {
		t.Fatalf("positional/truncated independent response: entries=%d first_missing=%v", len(prefix), len(prefix) > 0 && prefix[0] == nil)
	}
	recovered, err := node.fetchSnapBALPath(ctx, headers)
	if err != nil {
		t.Fatal(err)
	}
	if len(recovered) != 3 {
		t.Fatalf("native recovery steps %d", len(recovered))
	}
	for i, step := range recovered {
		if step.Header.Hash() != headers[i].Hash() || !bytes.Equal(step.Encoded, lists[i]) {
			t.Fatalf("recovery shifted or corrupted slot %d", i)
		}
	}
	// An independently served but differently committed list must not be accepted.
	store(a, 2, lists[0])
	if _, err := firstClient.RequestBlockAccessLists(ctx, headers[2:], nativesnap.BALSoftResponseLimit); err == nil {
		t.Fatal("accepted wrong header commitment")
	}
	store(b, 2, lists[2])
	retry, err := node.fetchSnapBALPath(ctx, headers[2:])
	if err != nil || len(retry) != 1 || !bytes.Equal(retry[0].Encoded, lists[2]) {
		t.Fatalf("native peer failover: steps=%d err=%v", len(retry), err)
	}
	missing := types.NewBlock(headers[2], &types.Body{}).Header()
	missing.Time++
	if missing.Hash() == headers[2].Hash() {
		t.Fatal("unknown-header control reused a cached header hash")
	}
	if _, err := node.fetchSnapBALPath(ctx, []*types.Header{missing}); err == nil {
		t.Fatal("unavailable required BAL falsely recovered")
	}
	t.Logf("INDEPENDENT_SNAP2 recovered=%d positional_holes=true truncated=true wrong_commitment_rejected=true eth_status_validated=true geth=v1.17.10", len(recovered))
}
