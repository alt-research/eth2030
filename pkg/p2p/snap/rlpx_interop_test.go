package snap

import (
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/p2p"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	gethsnap "github.com/ethereum/go-ethereum/eth/protocols/snap"
	"github.com/ethereum/go-ethereum/p2p/rlpx"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
)

func (b snapBALBackend) TrieNode(types.Hash, []byte) ([]byte, error) { return nil, nil }

func TestSnap2EncryptedExchangeWithIndependentSDKPackets(t *testing.T) {
	key, err := gethcrypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	h := NewServerHandler(snapBALBackend{lists: map[types.Hash][]byte{{1}: {0xc0}}})
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
	clientKey, err := gethcrypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := remote.Handshake(clientKey); err != nil {
		t.Fatal(err)
	}
	type capability struct {
		Name    string
		Version uint
	}
	hello := struct {
		Version uint64
		Name    string
		Caps    []capability
		Port    uint64
		ID      []byte
	}{5, "independent-snap2", []capability{{"snap", 2}, {"snap", 1}}, 0, gethcrypto.FromECDSAPub(&clientKey.PublicKey)[1:]}
	encoded, err := gethrlp.EncodeToBytes(hello)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := remote.Write(0, encoded); err != nil {
		t.Fatal(err)
	}
	code, _, _, err := remote.Read()
	if err != nil || code != 0 {
		t.Fatalf("hello: %d %v", code, err)
	}
	remote.SetSnappy(true)
	request := gethsnap.GetAccessListsPacket{ID: 17, Hashes: []gethcommon.Hash{{9}, {1}}, Bytes: BALSoftResponseLimit}
	encoded, err = gethrlp.EncodeToBytes(request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := remote.Write(0x10+gethsnap.GetAccessListsMsg, encoded); err != nil {
		t.Fatal(err)
	}
	code, encoded, _, err = remote.Read()
	if err != nil || code != 0x10+gethsnap.AccessListsMsg {
		t.Fatalf("BAL response code %d %v", code, err)
	}
	var reply gethsnap.AccessListsPacket
	if err := gethrlp.DecodeBytes(encoded, &reply); err != nil {
		t.Fatal(err)
	}
	items, err := reply.AccessLists.Items()
	if err != nil || reply.ID != 17 || len(items) != 2 || !bytes.Equal(items[0], []byte{0x80}) || !bytes.Equal(items[1], []byte{0xc0}) {
		t.Fatalf("SDK BAL packet: %x %v", encoded, err)
	}
	removed := gethsnap.GetTrieNodesPacket{ID: 18, Bytes: 1024}
	encoded, err = gethrlp.EncodeToBytes(removed)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := remote.Write(0x10+gethsnap.GetTrieNodesMsg, encoded); err != nil {
		t.Fatal(err)
	}
	code, _, _, err = remote.Read()
	if err == nil && code != 1 {
		t.Fatalf("removed trie request was served as %d", code)
	}
}
