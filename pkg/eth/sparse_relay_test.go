package eth

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto/bls"
	"github.com/eth2030/eth2030/p2p"
	"github.com/eth2030/eth2030/txpool"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

func awaitRelay(t *testing.T, label string, fn func() bool) {
	t.Helper()
	until := time.Now().Add(8 * time.Second)
	for time.Now().Before(until) {
		if fn() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("relay timed out: " + label)
}
func TestSparseSamplerTwoProvidersCustodyExpansionAndEagerPromotion(t *testing.T) {
	key, err := gethcrypto.HexToECDSA(strings.Repeat("11", 32))
	if err != nil {
		t.Fatal(err)
	}
	sender := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
	newPool := func() *txpool.TxPool {
		db := state.NewMemoryStateDB()
		db.AddBalance(sender, new(big.Int).Exp(big.NewInt(10), big.NewInt(23), nil))
		cfg := txpool.DefaultConfig()
		cfg.MinGasPrice = new(big.Int)
		pool := txpool.New(cfg, db)
		rules := *config.TestConfig
		rules.ChainID = big.NewInt(1337)
		pool.SetForkRules(&rules, 0)
		return pool
	}
	targetPool, firstPool, secondPool := newPool(), newPool(), newPool()
	target, first, second := NewHandler(newMockBlockchain(), targetPool, 1), NewHandler(newMockBlockchain(), firstPool, 1), NewHandler(newMockBlockchain(), secondPool, 1)
	target.relay.seed = [32]byte{}
	var custody types.CellMask
	custody[0] = 0xff
	target.SetCustodyColumns(custody)
	backend := bls.DefaultKZGBackend()
	blob := make([]byte, 131072)
	blob[31] = 9
	commitment, err := backend.BlobToCommitment(blob)
	if err != nil {
		t.Fatal(err)
	}
	_, proofs, err := backend.ComputeCellsAndProofs(blob)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(commitment[:])
	hash[0] = 1
	var native *types.Transaction
	for nonce := uint64(0); nonce < 64; nonce++ {
		signed, err := gethtypes.SignTx(gethtypes.NewTx(&gethtypes.BlobTx{ChainID: uint256.NewInt(1337), Nonce: nonce, Gas: 100000, GasFeeCap: uint256.NewInt(100), GasTipCap: uint256.NewInt(1), BlobFeeCap: uint256.NewInt(100), To: gethcommon.Address{19: 1}, Value: new(uint256.Int), BlobHashes: []gethcommon.Hash{hash}}), gethtypes.NewCancunSigner(big.NewInt(1337)), key)
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := signed.MarshalBinary()
		native, err = types.DecodeTxRLP(raw)
		if err != nil {
			t.Fatal(err)
		}
		if !target.relay.role(native.Hash()) {
			break
		}
		native = nil
	}
	if native == nil {
		t.Fatal("deterministic sampler fixture unavailable")
	}
	side := &types.BlobSidecarData{Version: 1, Blobs: [][]byte{blob}, Commitments: [][]byte{commitment[:]}}
	for _, p := range proofs {
		side.Proofs = append(side.Proofs, bytes.Clone(p[:]))
	}
	native.SetBlobSidecar(side)
	if err := firstPool.AddRemote(native); err != nil {
		t.Fatal(err)
	}
	if err := secondPool.AddRemote(native); err != nil {
		t.Fatal(err)
	}
	start := func(h *Handler) *p2p.Server {
		s := p2p.NewServer(p2p.Config{ListenAddr: "127.0.0.1:0", EnableRLPx: true, Protocols: h.Protocols()})
		if err := s.Start(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(s.Stop)
		return s
	}
	a, b, c := start(first), start(second), start(target)
	connect := func(source *p2p.Server) {
		_, port, _ := net.SplitHostPort(source.ListenAddr().String())
		if err := c.AddPeer("enode://" + source.LocalID() + "@" + net.JoinHostPort("127.0.0.1", port)); err != nil {
			t.Fatal(err)
		}
	}
	connect(a)
	awaitRelay(t, "signed metadata", func() bool { return target.relay.store.Get(native.Hash()) != nil })
	time.Sleep(3200 * time.Millisecond)
	if got := target.relay.store.Mask(native.Hash()); got.OneCount() != 0 || targetPool.Get(native.Hash()) != nil {
		t.Fatal("sampler fetched before observing two distinct providers")
	}
	connect(b)
	awaitRelay(t, "custody and sampling noise", func() bool { return target.relay.store.Mask(native.Hash()).Intersection(custody) == custody })
	sampled := target.relay.store.Mask(native.Hash())
	if sampled.OneCount() != 9 || targetPool.Get(native.Hash()) != nil {
		t.Fatalf("sampler did not fetch exactly custody plus one extra: %d", sampled.OneCount())
	}
	expanded := custody
	expanded[2] = 0xff
	target.SetCustodyColumns(expanded)
	awaitRelay(t, "custody expansion", func() bool { return target.relay.store.Mask(native.Hash()).Intersection(expanded) == expanded })
	if got := target.relay.store.Mask(native.Hash()); got.OneCount() > 18 || targetPool.Get(native.Hash()) != nil {
		t.Fatal("custody expansion fetched a full blob or entered executable pool")
	}
	cells := target.GetBlobCells([]types.Hash{types.Hash(hash)}, expanded)
	if len(cells) != 1 || len(cells[0].Cells) != 16 {
		t.Fatal("sparse Engine projection omitted custody delta")
	}
	for i, cell := range cells[0].Cells {
		if len(cell) != 2048 || len(cells[0].Proofs[i]) != 48 {
			t.Fatal("authenticated custody cell missing")
		}
	}
	target.PrepareBlobBuilding()
	awaitRelay(t, "eager reconstruction", func() bool { return targetPool.Get(native.Hash()) != nil })
	full := targetPool.Get(native.Hash())
	if !types.HasFullBlobSidecar(full) || !bytes.Equal(full.BlobSidecar().Blobs[0], blob) {
		t.Fatal("eager builder did not reconstruct full signed data")
	}
	if target.relay.store.Get(native.Hash()) != nil {
		t.Fatal("promoted sparse bytes retained twice")
	}
	t.Logf("signed=%s providers=2 sampled=%d expanded=16 eager=%d bytes", native.Hash(), sampled.OneCount(), len(blob))
}
func TestSparseProviderProbability(t *testing.T) {
	r := newSparseRelay(newMockTxPool())
	r.seed = [32]byte{}
	n := 0
	for i := 0; i < 10000; i++ {
		sum := sha256.Sum256([]byte(fmt.Sprint(i)))
		if r.role(types.Hash(sum)) {
			n++
		}
	}
	if n < 1400 || n > 1600 {
		t.Fatalf("provider probability drift: %d/10000", n)
	}
}

func TestSparseCellsRejectUnrequestedAndExpiredResponses(t *testing.T) {
	for _, kind := range []string{"peer", "id", "expired", "mask", "hash", "replay"} {
		t.Run(kind, func(t *testing.T) {
			h := NewHandler(newMockBlockchain(), newMockTxPool(), 1)
			peer := p2p.NewPeer("p", "127.0.0.1:1", nil)
			peer.SetVersion(ETH72)
			left, right := p2p.MsgPipe()
			defer left.Close()
			defer right.Close()
			ep := NewEthPeer(peer, left)
			key := relayKey{peer: "p", id: 7}
			req := relayRequest{code: CellsMsg, hashes: []types.Hash{{1}}, mask: types.CellMask{1}, expires: time.Now().Add(time.Minute)}
			h.relay.requests[key] = req
			reply := CellsPacket{RequestID: 7}
			switch kind {
			case "peer":
				ep.peer = p2p.NewPeer("other", "127.0.0.1:2", nil)
				ep.peer.SetVersion(ETH72)
			case "id":
				reply.RequestID = 8
			case "expired":
				req.expires = time.Now().Add(-time.Second)
				h.relay.requests[key] = req
			case "mask":
				reply.Mask = types.CellMask{2}
			case "hash":
				reply.Hashes = []types.Hash{{2}}
				reply.Mask = types.CellMask{1}
				reply.Cells = [][][]byte{{make([]byte, 2048)}}
			case "replay":
				if _, err := h.takeRelayRequest(ep, 7, CellsMsg); err != nil {
					t.Fatal(err)
				}
			}
			raw, err := gethrlp.EncodeToBytes(reply)
			if err != nil {
				t.Fatal(err)
			}
			err = h.handleCells(ep, p2p.Msg{Code: CellsMsg, Payload: raw, Size: uint32(len(raw))})
			if err == nil || !strings.Contains(err.Error(), "unrequested") && !strings.Contains(err.Error(), "unsolicited") {
				t.Fatalf("%s correlation guard: %v", kind, err)
			}
			if len(h.relay.store.Hashes()) != 0 {
				t.Fatal("unrequested cells published state")
			}
		})
	}
}
