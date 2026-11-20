package snap

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"sort"
	"testing"
	"time"

	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/p2p/wire"
	"github.com/eth2030/eth2030/sync/statesync"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethsnap "github.com/ethereum/go-ethereum/eth/protocols/snap"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
)

func independentAccountPacket(t *testing.T, count int) (*state.MemoryStateDB, *state.StateSnapshot, *gethsnap.AccountRangePacket) {
	t.Helper()
	source := state.NewMemoryStateDB()
	for i := 1; i <= count; i++ {
		source.AddBalance(types.BytesToAddress(big.NewInt(int64(i)).Bytes()), big.NewInt(int64(i)*1000))
	}
	source.SetNonce(types.Address{19: 3}, 1)
	source.SetCode(types.Address{19: 3}, []byte{0x60, 1, 0x50, 0})
	source.SetState(types.Address{19: 3}, types.Hash{31: 1}, types.Hash{31: 9})
	snapshot, err := state.ExportStateSnapshot(source)
	if err != nil {
		t.Fatal(err)
	}
	var keys []types.Hash
	for key := range snapshot.Accounts {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return bytes.Compare(keys[i][:], keys[j][:]) < 0 })
	packet := &gethsnap.AccountRangePacket{ID: 1}
	for _, key := range keys {
		var account gethtypes.StateAccount
		if err := gethrlp.DecodeBytes(snapshot.Accounts[key], &account); err != nil {
			t.Fatal(err)
		}
		packet.Accounts = append(packet.Accounts, &gethsnap.AccountData{Hash: gethcommon.Hash(key), Body: gethrlp.RawValue(gethtypes.SlimAccountRLP(account))})
	}
	return source, snapshot, packet
}

func TestIndependentSDKRawAccountListReachesNativeRangeProofConsumer(t *testing.T) {
	for _, test := range []struct {
		version uint
		count   int
	}{{1, 3}, {2, 3}, {2, MaxAccountRangeCount + 1}} {
		t.Run(fmt.Sprintf("v%d/accounts=%d", test.version, test.count), func(t *testing.T) {
			source, _, packet := independentAccountPacket(t, test.count)
			left, right := wire.MsgPipe()
			defer left.Close()
			defer right.Close()
			client := &Client{id: "independent-sdk", version: test.version, transport: left, pending: make(map[uint64]pendingResponse)}
			readerDone := make(chan struct{})
			go func() {
				defer close(readerDone)
				for {
					msg, err := left.ReadMsg()
					if err != nil {
						return
					}
					if client.deliver(msg) != nil {
						return
					}
				}
			}()
			defer func() { left.Close(); <-readerDone }()
			sent := make(chan error, 1)
			go func() {
				msg, err := right.ReadMsg()
				if err != nil {
					sent <- err
					return
				}
				var request gethsnap.GetAccountRangePacket
				if err := gethrlp.DecodeBytes(msg.Payload, &request); err != nil {
					sent <- err
					return
				}
				packet.ID = request.ID
				encoded, err := gethrlp.EncodeToBytes(packet)
				if err != nil {
					sent <- err
					return
				}
				sent <- right.WriteMsg(wire.Msg{Code: AccountRangeMsg, Size: uint32(len(encoded)), Payload: encoded})
			}()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			peer := VerifiedPeers(ctx, []*Client{client})[0]
			var limit types.Hash
			for i := range limit {
				limit[i] = 0xff
			}
			response, err := peer.RequestAccountRange(statesync.AccountRangeRequest{Root: source.GetRoot(), Limit: limit, Bytes: SoftResponseLimit})
			if err != nil {
				t.Fatalf("actual independent raw account list failed native cryptographic range consumer: %v", err)
			}
			if len(response.Accounts) != test.count || response.More {
				t.Fatalf("complete original range changed: %+v", response)
			}
			if err := <-sent; err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestNativeServedAccountPacketUsesIndependentCanonicalRawLists(t *testing.T) {
	_, snapshot, original := independentAccountPacket(t, 3)
	packet := AccountRangePacket{ID: original.ID}
	for _, account := range original.Accounts {
		packet.Accounts = append(packet.Accounts, AccountData{Hash: types.Hash(account.Hash), Body: bytes.Clone(account.Body)})
	}
	left, right := wire.MsgPipe()
	defer left.Close()
	defer right.Close()
	done := make(chan error, 1)
	go func() { done <- sendSnapMsg(left, AccountRangeMsg, &packet) }()
	msg, err := right.ReadMsg()
	if err != nil {
		t.Fatal(err)
	}
	var decoded gethsnap.AccountRangePacket
	if err := gethrlp.DecodeBytes(msg.Payload, &decoded); err != nil {
		t.Fatal(err)
	}
	keys, accounts, err := decoded.Unpack()
	if err != nil {
		t.Fatalf("native server double-encoded the account list for independent consumer: %v", err)
	}
	if len(accounts) != 3 {
		t.Fatal("native server lost original accounts")
	}
	for i, key := range keys {
		if !bytes.Equal(accounts[i], snapshot.Accounts[types.Hash(key)]) {
			t.Fatal("independent account expansion differs from original consensus bytes")
		}
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
