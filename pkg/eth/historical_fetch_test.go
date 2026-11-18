package eth

import (
	"bytes"
	"context"
	"errors"
	"github.com/eth2030/eth2030/crypto"
	"math/big"
	"strings"
	"testing"
	"time"

	coreblock "github.com/eth2030/eth2030/core/block"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/p2p"
)

type receiptRestoringChain struct {
	*mockBlockchain
	stored []*types.Receipt
}

func (c *receiptRestoringChain) StoreReceivedReceipts(_ types.Hash, receipts []*types.Receipt) error {
	c.stored = receipts
	return nil
}

func TestReceiptConsumerPaginationAndIntegrity(t *testing.T) {
	for _, mode := range []string{"partial", "bad_root", "no_progress", "too_large", "extra_page"} {
		t.Run(mode, func(t *testing.T) {
			chain := &receiptRestoringChain{mockBlockchain: newMockBlockchain()}
			receipts := []*types.Receipt{{Status: 1, CumulativeGasUsed: 21000}, {Status: 1, CumulativeGasUsed: 42000}}
			candidate := makeTestBlock(1, chain.Genesis().Hash(), []*types.Transaction{makeTestTx(0), makeTestTx(1)})
			header := candidate.Header()
			header.ReceiptHash = coreblock.ComputeReceiptsRoot(receipts)
			block := types.NewBlock(header, candidate.Body())
			chain.addBlock(block)
			h := NewHandler(chain, newMockTxPool(), 1)
			a, b := p2p.MsgPipe()
			ep := NewEthPeer(p2p.NewPeer("source", "127.0.0.1:1", nil), a)
			ep.Peer().SetVersion(ETH71)
			h.ethPeers[ep.ID()] = ep
			loopDone := make(chan struct{})
			go func() { defer close(loopDone); defer close(ep.done); _ = h.handleMessages(ep) }()
			t.Cleanup(func() {
				a.Close()
				b.Close()
				select {
				case <-loopDone:
				case <-time.After(time.Second):
					t.Error("receipt reader leaked")
				}
			})
			scriptDone := make(chan error, 1)
			go func() {
				for page := 0; page < 2; page++ {
					msg, err := b.ReadMsg()
					if err != nil {
						scriptDone <- err
						return
					}
					var request p2p.GetReceiptsPacketV70
					if err := decodeMsg(msg, &request); err != nil {
						scriptDone <- err
						return
					}
					if len(request.Hashes) != 1 || request.Hashes[0] != block.Hash() || request.FirstBlockReceiptIndex != uint64(page) {
						scriptDone <- errors.New("pagination request lost hash/offset")
						return
					}
					response := p2p.ReceiptsPacketV70{RequestID: request.RequestID, LastBlockIncomplete: page == 0, Receipts: [][]*types.Receipt{{receipts[page]}}}
					switch mode {
					case "bad_root":
						response.LastBlockIncomplete = false
						response.Receipts = [][]*types.Receipt{{receipts[0], {Status: 0, CumulativeGasUsed: 42000}}}
					case "no_progress":
						response.Receipts = [][]*types.Receipt{{}}
					case "too_large":
						response.Receipts = [][]*types.Receipt{{{Status: 1, Logs: []*types.Log{{Data: make([]byte, 10000)}}}}}
					case "extra_page":
						response.Receipts = [][]*types.Receipt{{receipts[0]}, {receipts[1]}}
					}
					encoded, err := p2p.EncodeMessage(p2p.ReceiptsMsg, &response)
					if err != nil {
						scriptDone <- err
						return
					}
					if err := b.WriteMsg(p2p.Msg{Code: encoded.Code, Size: encoded.Size, Payload: encoded.Payload}); err != nil {
						scriptDone <- err
						return
					}
					if mode != "partial" {
						break
					}
				}
				scriptDone <- nil
			}()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			got, err := h.FetchReceipts(ctx, block.Hash())
			if mode == "partial" {
				if err != nil || len(got) != 2 || len(chain.stored) != 2 {
					t.Fatalf("validated pagination failed: %v", err)
				}
			} else {
				if err == nil || len(chain.stored) != 0 {
					t.Fatalf("invalid receipts reached storage: %v", err)
				}
				if mode == "bad_root" && !strings.Contains(err.Error(), "receipt root mismatch") {
					t.Fatal(err)
				}
				if mode == "too_large" && !errors.Is(err, ErrReceiptsTooLarge) {
					t.Fatal(err)
				}
			}
			if err := <-scriptDone; err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestHistoricalBALFetchUsesCorrelatedStorageConsumer(t *testing.T) {
	source := newMockBlockchain()
	data := []byte{0xc1, 1}
	commit := crypto.Keccak256Hash(data)
	block := types.NewBlock(&types.Header{Number: big.NewInt(1), BlockAccessListHash: &commit}, &types.Body{})
	source.addBlock(block)
	target := &restoringChain{newMockBlockchain(), make(map[types.Hash][]byte)}
	target.addBlock(block)
	server := NewHandler(source, newMockTxPool(), 1)
	provider := newMockAccessListProvider()
	provider.accessLists[block.Hash()] = data
	server.SetAccessListProvider(provider)
	client := NewHandler(target, newMockTxPool(), 1)
	a, b := p2p.MsgPipe()
	ep := NewEthPeer(p2p.NewPeer("source", "127.0.0.1:1", nil), a)
	ep.Peer().SetVersion(ETH71)
	other := NewEthPeer(p2p.NewPeer("client", "127.0.0.1:2", nil), b)
	other.Peer().SetVersion(ETH71)
	client.ethPeers[ep.ID()] = ep
	done := make(chan struct{})
	serverDone := make(chan struct{})
	go func() { defer close(done); defer close(ep.done); _ = client.handleMessages(ep) }()
	go func() { defer close(serverDone); _ = server.handleMessages(other) }()
	defer func() { a.Close(); b.Close(); <-done; <-serverDone }()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	received, err := client.FetchBlockAccessList(ctx, block.Hash())
	if err != nil || !bytes.Equal(received, data) || !bytes.Equal(target.restored[block.Hash()], data) {
		t.Fatalf("historical BAL consumer failed: %x %v", received, err)
	}
}
