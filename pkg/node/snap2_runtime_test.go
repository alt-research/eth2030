package node

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	coreblock "github.com/eth2030/eth2030/core/block"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
	"github.com/eth2030/eth2030/p2p"
	p2psnap "github.com/eth2030/eth2030/p2p/snap"
	syncbeacon "github.com/eth2030/eth2030/sync/beacon"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

type snap2FixturePool struct{ transactions []*types.Transaction }

func (p snap2FixturePool) Pending() []*types.Transaction { return p.transactions }

type advancingSnapBackend struct {
	*snapStateBackend
	oldRoot         types.Hash
	contract        types.Hash
	once            sync.Once
	advance         func()
	onStorage       bool
	stale           atomic.Bool
	oldCalls        atomic.Uint64
	newCalls        atomic.Uint64
	pointCalls      atomic.Uint64
	restores        atomic.Uint64
	partialStorage  bool
	storageResumes  atomic.Uint64
	storageRestarts atomic.Uint64
	corruptProof    bool
	corruptPoint    atomic.Bool
}

func (b *advancingSnapBackend) AccountProof(root, key types.Hash) ([][]byte, error) {
	proof, err := b.snapStateBackend.AccountProof(root, key)
	if err == nil && root != b.oldRoot && b.corruptPoint.Load() && len(proof) > 0 {
		copyProof := make([][]byte, len(proof))
		for i, node := range proof {
			copyProof[i] = append([]byte(nil), node...)
		}
		copyProof[0][len(copyProof[0])-1] ^= 1
		return copyProof, nil
	}
	return proof, err
}

func (b *advancingSnapBackend) AccountIterator(root, origin types.Hash, fn func(types.Hash, []byte) bool) error {
	// Corrupt only the contract point query. Subsequent bulk proofs stay valid,
	// so their independent range guard cannot mask a bypassed point guard.
	b.corruptPoint.Store(b.corruptProof && root != b.oldRoot && origin == b.contract)
	if root == b.oldRoot {
		b.oldCalls.Add(1)
		if b.stale.Load() {
			return p2psnap.ErrMissingRoot
		}
	} else {
		b.newCalls.Add(1)
		for _, part := range origin[1:] {
			if part != 0 {
				b.pointCalls.Add(1)
				break
			}
		}
	}
	err := b.snapStateBackend.AccountIterator(root, origin, fn)
	if err == nil && root == b.oldRoot && !b.onStorage {
		b.once.Do(func() { b.stale.Store(true); b.advance() })
	}
	return err
}
func (b *advancingSnapBackend) StorageIterator(root, account types.Hash, origin []byte, fn func(types.Hash, []byte) bool) error {
	if root != b.oldRoot && account == b.contract && b.partialStorage {
		if types.BytesToHash(origin) == (types.Hash{}) {
			b.storageRestarts.Add(1)
		} else {
			b.storageResumes.Add(1)
		}
	}
	if root != b.oldRoot && account == b.contract && len(origin) == 32 && types.BytesToHash(origin) == crypto.Keccak256Hash(make([]byte, 32)) {
		b.restores.Add(1)
	}
	callback := fn
	if root == b.oldRoot && account == b.contract && b.partialStorage {
		callback = func(key types.Hash, value []byte) bool { fn(key, value); return false }
	}
	err := b.snapStateBackend.StorageIterator(root, account, origin, callback)
	if err == nil && root == b.oldRoot && account == b.contract && b.onStorage {
		b.once.Do(func() { b.stale.Store(true); b.advance() })
	}
	return err
}

func TestSnap2ActualMovingPivotReorgAndMissingOrphanRecovery(t *testing.T) {
	for _, scenario := range []string{"moving-partial", "moving-partial-storage", "reorg-partial-storage", "missing-orphan", "corrupt-new-proof"} {
		t.Run(scenario, func(t *testing.T) {
			key, err := gethcrypto.HexToECDSA(strings.Repeat("11", 32))
			if err != nil {
				t.Fatal(err)
			}
			sender := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
			contract := types.Address{18: 0xbb, 19: 1}
			var genesis map[string]any
			if err := json.Unmarshal(builderRequestGenesisJSON(1337), &genesis); err != nil {
				t.Fatal(err)
			}
			cfg := genesis["config"].(map[string]any)
			cfg["amsterdamTime"], cfg["osakaTime"], cfg["terminalTotalDifficulty"] = 0, 0, 0
			alloc := genesis["alloc"].(map[string]any)
			alloc[sender.Hex()] = map[string]string{"balance": "100000000000000000000"}
			alloc[contract.Hex()] = map[string]any{"balance": "0", "nonce": "0x1", "code": "0x6020355f355500", "storage": map[string]string{"0x00": "0x05"}}
			if scenario == "moving-partial-storage" {
				alloc[contract.Hex()].(map[string]any)["storage"] = map[string]string{"0x00": "0x05", "0x06": "0x06", "0x07": "0x07"}
			}
			data, err := json.Marshal(genesis)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "genesis.json")
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			sourceCfg, targetCfg := makeTestConfig(t), makeTestConfig(t)
			sourceCfg.GenesisPath, targetCfg.GenesisPath = path, path
			targetCfg.SyncMode = "snap"
			source, target := newTestNode(t, &sourceCfg), newTestNode(t, &targetCfg)
			build := func(parent *types.Block, nonce, slot, value, timestamp uint64) *types.Block {
				t.Helper()
				input := make([]byte, 64)
				input[31], input[63] = byte(slot), byte(value)
				tx, err := gethtypes.SignTx(gethtypes.NewTransaction(nonce, gethcommon.Address(contract), new(big.Int), 100_000, big.NewInt(2_000_000_000), input), gethtypes.LatestSignerForChainID(big.NewInt(1337)), key)
				if err != nil {
					t.Fatal(err)
				}
				raw, err := tx.MarshalBinary()
				if err != nil {
					t.Fatal(err)
				}
				native, err := types.DecodeTxRLP(raw)
				if err != nil {
					t.Fatal(err)
				}
				beacon := types.Hash{1}
				builder := coreblock.NewBlockBuilder(source.blockchain.Config(), source.blockchain, snap2FixturePool{[]*types.Transaction{native}})
				block, _, err := builder.BuildBlock(parent.Header(), &coreblock.BuildBlockAttributes{Timestamp: timestamp, GasLimit: 30_000_000, BeaconRoot: &beacon, SlotNumber: &timestamp})
				if err != nil {
					t.Fatal(err)
				}
				if len(block.Transactions()) != 1 {
					t.Fatal("signed pivot transaction omitted")
				}
				if err := source.blockchain.InsertBlock(block); err != nil {
					t.Fatal(err)
				}
				return block
			}
			old := build(source.blockchain.Genesis(), 0, 0, 10, 12)
			var latest *types.Block
			if scenario == "moving-partial" || scenario == "corrupt-new-proof" {
				latest = build(old, 1, 1, 20, 24)
			} else if scenario == "moving-partial-storage" {
				latest = build(old, 1, 5, 20, 24)
			} else {
				old = build(old, 1, 2, 99, 24)
				latest = build(source.blockchain.Genesis(), 0, 1, 30, 36)
				latest = build(latest, 1, 3, 40, 48)
			}
			if scenario == "missing-orphan" {
				if err := source.db.Delete(append([]byte("BAL/"), old.Hash().Bytes()...)); err != nil {
					t.Fatal(err)
				}
			}
			backend := &advancingSnapBackend{snapStateBackend: newSnapStateBackend(source.blockchain, source.db), oldRoot: old.Root(), contract: crypto.Keccak256Hash(contract[:]), onStorage: scenario != "moving-partial" && scenario != "corrupt-new-proof", partialStorage: scenario == "moving-partial-storage", corruptProof: scenario == "corrupt-new-proof"}
			backend.advance = func() {
				target.executionSync.Request(syncbeacon.ExecutionTarget{Head: latest.Hash(), Finalized: latest.Hash()})
			}
			source.snapHandler = p2psnap.NewServerHandler(backend)
			// Real encrypted negotiation and dispatch, with a serving-policy
			// wrapper that makes the old root stale after authenticated progress.
			source.p2pServer = p2p.NewServer(p2p.Config{ListenAddr: "127.0.0.1:0", MaxPeers: 10, EnableRLPx: true, Protocols: append(source.ethHandler.Protocols(), source.snapHandler.Protocols()...)})
			if err := source.p2pServer.Start(); err != nil {
				t.Fatal(err)
			}
			if err := target.p2pServer.Start(); err != nil {
				t.Fatal(err)
			}
			connectSyncNodes(t, source, target)
			if clients := target.snap2Clients(); len(clients) == 0 || clients[0].Version() != 2 {
				t.Fatal("snap/2 was not negotiated")
			}
			target.executionSync.Request(syncbeacon.ExecutionTarget{Head: old.Hash(), Finalized: old.Hash()})
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			adopted, err := target.syncFinalizedPivot(ctx, old)
			if scenario == "corrupt-new-proof" {
				if err == nil || !strings.Contains(err.Error(), "proof") || adopted || target.blockchain.CurrentBlock().NumberU64() != 0 {
					t.Fatalf("unverified rebase adopted: %t %v", adopted, err)
				}
				if exists, dbErr := target.db.Has([]byte("snap-pivot")); dbErr != nil || exists {
					t.Fatal("failed rebase published a checkpoint")
				}
				return
			}
			if err != nil || !adopted {
				t.Fatalf("snap/2 %s adoption=%v error=%v", scenario, adopted, err)
			}
			if target.blockchain.CurrentBlock().Hash() != latest.Hash() || target.blockchain.CurrentBlock().Root() != latest.Root() {
				t.Fatal("wrong CL-selected pivot adopted")
			}
			want0, want1 := byte(10), byte(20)
			if scenario == "reorg-partial-storage" || scenario == "missing-orphan" {
				want0, want1 = 5, 30
			}
			st := target.blockchain.State()
			wantSlot := types.Hash{31: 1}
			if scenario == "moving-partial-storage" {
				wantSlot = types.Hash{31: 5}
			}
			if st.GetState(contract, types.Hash{}) != (types.Hash{31: want0}) || st.GetState(contract, wantSlot) != (types.Hash{31: want1}) {
				t.Fatalf("old-fork storage leaked: 0=%s 1=%s", st.GetState(contract, types.Hash{}), st.GetState(contract, types.Hash{31: 1}))
			}
			if scenario == "moving-partial-storage" && (backend.storageResumes.Load() == 0 || backend.storageRestarts.Load() != 0 || st.GetState(contract, types.Hash{31: 6}) != (types.Hash{31: 6}) || st.GetState(contract, types.Hash{31: 7}) != (types.Hash{31: 7})) {
				t.Fatalf("partial storage cursor/suffix: resumes=%d restarts=%d slot6=%s slot7=%s", backend.storageResumes.Load(), backend.storageRestarts.Load(), st.GetState(contract, types.Hash{31: 6}), st.GetState(contract, types.Hash{31: 7}))
			}
			if scenario != "moving-partial" && st.GetState(contract, types.Hash{31: 2}) != (types.Hash{}) {
				t.Fatal("orphan-only created storage survived")
			}
			if scenario == "reorg-partial-storage" && (backend.restores.Load() == 0 || backend.pointCalls.Load() == 0) {
				t.Fatal("reorg silently restarted instead of restoring orphan-only entries")
			}
			if scenario == "moving-partial" && backend.pointCalls.Load() == 0 {
				t.Fatal("moving pivot silently restarted instead of rebasing partial progress")
			}
			if backend.oldCalls.Load() == 0 || backend.newCalls.Load() == 0 {
				t.Fatal("fixture did not cross proof roots")
			}
			head := target.blockchain.CurrentBlock().Hash()
			root := target.blockchain.CurrentBlock().Root()
			if err := target.Stop(); err != nil {
				t.Fatal(err)
			}
			reopened := newTestNode(t, &targetCfg)
			if reopened.blockchain.CurrentBlock().Hash() != head || reopened.blockchain.CurrentBlock().Root() != root {
				t.Fatal("snap/2 pivot did not survive restart")
			}
			if stored, err := reopened.db.Get([]byte("snap-pivot")); err != nil || types.BytesToHash(stored) != head {
				t.Fatalf("persisted pivot authority missing: %v", err)
			}
			t.Logf("scenario=%s pivot=%s root=%s oldRequests=%d newRequests=%d signed=%s", scenario, head, root, backend.oldCalls.Load(), backend.newCalls.Load(), hex.EncodeToString(gethcrypto.FromECDSAPub(&key.PublicKey)))
		})
	}
}
