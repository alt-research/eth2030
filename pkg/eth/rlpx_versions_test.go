package eth

import (
	"bytes"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/p2p"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/rlpx"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
)

func TestNegotiatedEthVersionsAgainstIndependentCodec(t *testing.T) {
	for _, version := range []uint{68, 69, 70, 71} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			key, err := gethcrypto.GenerateKey()
			if err != nil {
				t.Fatal(err)
			}
			chain := newMockBlockchain()
			block := makeTestBlock(1, chain.Genesis().Hash(), []*types.Transaction{makeTestTx(0)})
			chain.addBlock(block)
			h := NewHandler(chain, newMockTxPool(), 1)
			sample := &types.Receipt{Status: 1, CumulativeGasUsed: 21000, Logs: []*types.Log{{Address: types.Address{19: 9}, Topics: []types.Hash{{31: 7}}, Data: []byte{1}}}}
			sample.Bloom = types.LogsBloom(sample.Logs)
			h.SetReceiptProvider(fixedReceiptProvider{block.Hash(), []*types.Receipt{sample}})
			server := p2p.NewServer(p2p.Config{ListenAddr: "127.0.0.1:0", EnableRLPx: true, RLPxPrivateKey: key, Protocols: h.Protocols()})
			if err := server.Start(); err != nil {
				t.Fatal(err)
			}
			defer server.Stop()
			conn, err := net.Dial("tcp", server.ListenAddr().String())
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			remote := rlpx.NewConn(conn, &key.PublicKey)
			_ = remote.SetDeadline(time.Now().Add(5 * time.Second))
			clientKey, _ := gethcrypto.GenerateKey()
			if _, err := remote.Handshake(clientKey); err != nil {
				t.Fatal(err)
			}
			type cap struct {
				Name    string
				Version uint
			}
			caps := []cap{{"eth", version}}
			if version > 68 {
				caps = append(caps, cap{"eth", 68})
			}
			hello := struct {
				Version uint64
				Name    string
				Caps    []cap
				Port    uint64
				ID      []byte
			}{5, "independent-eth", caps, 0, gethcrypto.FromECDSAPub(&clientKey.PublicKey)[1:]}
			raw, err := gethrlp.EncodeToBytes(hello)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := remote.Write(0, raw); err != nil {
				t.Fatal(err)
			}
			code, _, _, err := remote.Read()
			if err != nil || code != 0 {
				t.Fatalf("hello: %x %v", code, err)
			}
			remote.SetSnappy(true)
			code, raw, _, err = remote.Read()
			if err != nil || code != 0x10 {
				t.Fatalf("status: %x %v", code, err)
			}
			if version == 68 {
				var status p2p.StatusData
				if err := gethrlp.DecodeBytes(raw, &status); err != nil || status.ProtocolVersion != 68 {
					t.Fatalf("legacy status: %+v %v", status, err)
				}
				raw, _ = gethrlp.EncodeToBytes(status)
			} else {
				var status status69
				if err := gethrlp.DecodeBytes(raw, &status); err != nil || status.Version != uint32(version) || status.Earliest > status.Latest {
					t.Fatalf("range status: %+v %v", status, err)
				}
				raw, _ = gethrlp.EncodeToBytes(status)
			}
			if _, err := remote.Write(0x10, raw); err != nil {
				t.Fatal(err)
			}
			if version == 71 {
				request := struct {
					ID     uint64
					Hashes []types.Hash
				}{77, []types.Hash{h.chain.Genesis().Hash()}}
				raw, _ = gethrlp.EncodeToBytes(request)
				if _, err := remote.Write(0x10+p2p.GetBlockAccessListsMsg, raw); err != nil {
					t.Fatal(err)
				}
				code, raw, _, err = remote.Read()
				if err != nil || code != 0x10+p2p.BlockAccessListsMsg {
					t.Fatalf("BAL reply: %x %v", code, err)
				}
				var reply struct {
					ID   uint64
					BALs []gethrlp.RawValue
				}
				if err := gethrlp.DecodeBytes(raw, &reply); err != nil || reply.ID != 77 || len(reply.BALs) != 1 || !bytes.Equal(reply.BALs[0], []byte{0xc0}) {
					t.Fatalf("noncanonical BAL wire: %x %v", raw, err)
				}
			}
			{
				// A legacy receipt request works on 68/69; 70 uses its offset field.
				if version < 70 {
					raw, _ = gethrlp.EncodeToBytes(struct {
						ID     uint64
						Hashes []types.Hash
					}{5, []types.Hash{block.Hash()}})
				} else {
					raw, _ = gethrlp.EncodeToBytes(struct {
						ID, Offset uint64
						Hashes     []types.Hash
					}{5, 0, []types.Hash{block.Hash()}})
				}
				if _, err := remote.Write(0x10+p2p.GetReceiptsMsg, raw); err != nil {
					t.Fatal(err)
				}
				code, raw, _, err = remote.Read()
				if err != nil || code != 0x10+p2p.ReceiptsMsg {
					t.Fatalf("receipt reply: %x %v", code, err)
				}
				var fields []gethrlp.RawValue
				if err := gethrlp.DecodeBytes(raw, &fields); err != nil {
					t.Fatal(err)
				}
				want := 2
				if version >= 70 {
					want = 3
				}
				if len(fields) != want {
					t.Fatalf("receipt schema fields=%d want %d", len(fields), want)
				}
				var pages [][]gethrlp.RawValue
				if err := gethrlp.DecodeBytes(fields[want-1], &pages); err != nil || len(pages) != 1 || len(pages[0]) != 1 {
					t.Fatalf("receipt pages: %x %v", raw, err)
				}
				var receipt []gethrlp.RawValue
				if err := gethrlp.DecodeBytes(pages[0][0], &receipt); err != nil || len(receipt) != 4 {
					t.Fatalf("receipt fields: %x %v", pages[0][0], err)
				}
				if version >= 69 {
					if !bytes.Equal(receipt[0], []byte{0x80}) {
						t.Fatal("modern receipt omitted explicit transaction type")
					}
				} else {
					var bloom []byte
					if err := gethrlp.DecodeBytes(receipt[2], &bloom); err != nil || len(bloom) != 256 {
						t.Fatal("legacy bloom encoding lost")
					}
				}

			}
		})
	}
}

type fixedReceiptProvider struct {
	hash     types.Hash
	receipts []*types.Receipt
}

func (p fixedReceiptProvider) GetReceipts(hash types.Hash) []*types.Receipt {
	if hash == p.hash {
		return p.receipts
	}
	return nil
}
