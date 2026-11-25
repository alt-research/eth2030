package node

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine/payload"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
	gethtrie "github.com/ethereum/go-ethereum/trie"
)

// This test-binary argument is not a node option. The input must name an owned,
// disposable restored database; all replay writes and marker resets stay there.
var nativeReplayManifest = flag.String("native-import-replay", "", "private mature-state replay manifest (disposable clone only)")

type nativeReplayRow struct {
	Hash types.Hash    `json:"hash"`
	RLP  hexutil.Bytes `json:"rlp"`
}

type nativeReplayInput struct {
	DataDir         string `json:"datadir"`
	Genesis         string `json:"genesis"`
	Rows            string `json:"rows"`
	RowsSHA256      string `json:"rows_sha256"`
	DisposableClone bool   `json:"disposable_clone"`
}

func nativeReplayDecode(row nativeReplayRow) (*types.Block, error) {
	var independent gethtypes.Block
	if err := gethrlp.DecodeBytes(row.RLP, &independent); err != nil {
		return nil, err
	}
	b, err := types.DecodeBlockRLP(row.RLP)
	if err != nil {
		return nil, err
	}
	if row.Hash == (types.Hash{}) || b.Hash() != row.Hash || types.Hash(independent.Hash()) != row.Hash {
		return nil, fmt.Errorf("native/SDK RLP identity mismatch")
	}
	if b.Header().SlotNumber == nil || b.Header().ParentBeaconRoot == nil || b.Header().BlockAccessListHash == nil {
		return nil, fmt.Errorf("replay requires a complete Amsterdam header")
	}
	return b, nil
}

func nativeReplayBounded(t *testing.T, label string, run func() error) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- run() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
	case <-time.After(8 * time.Second):
		var stack bytes.Buffer
		_ = pprof.Lookup("goroutine").WriteTo(&stack, 2)
		t.Logf("WAIT BEFORE CLEANUP %s\n%s", label, stack.String())
		t.Fatalf("%s exceeded the unchanged Engine8s budget", label)
	}
}

func nativeReplayProof(rpc track4RPC, anchor *types.Block) error {
	address := types.Address{19: 0x71}
	var proof struct{ AccountProof []hexutil.Bytes }
	if err := rpc.invoke("eth_getProof", &proof, address.Hex(), []string{}, map[string]any{"blockHash": anchor.Hash(), "requireCanonical": true}); err != nil {
		return err
	}
	db := memorydb.New()
	for _, n := range proof.AccountProof {
		if err := db.Put(gethcrypto.Keccak256(n), n); err != nil {
			return err
		}
	}
	_, err := gethtrie.VerifyProof(gethcommon.Hash(anchor.Root()), gethcrypto.Keccak256(address[:]), db)
	return err
}

func nativeReplayRun(t *testing.T, n *Node, rows []nativeReplayRow) {
	t.Helper()
	if len(rows) == 0 {
		t.Fatal("empty replay is not a qualification")
	}
	first, err := nativeReplayDecode(rows[0])
	if err != nil {
		t.Fatal(err)
	}
	anchor := n.blockchain.GetBlock(first.ParentHash())
	if anchor == nil || !n.blockchain.HasValidatedBlock(anchor.Hash()) {
		t.Fatal("first parent is not locally execution validated")
	}
	final := types.Hash{}
	if b := n.blockchain.CurrentFinalBlock(); b != nil {
		final = b.Hash()
	}
	// Rewind only the disposable clone to the authenticated parent. No finality
	// regression is allowed. Do not let stale-FCU optimization skip this setup.
	if err := n.blockchain.ApplyForkchoice(anchor.Hash(), final, final); err != nil {
		t.Fatal(err)
	}
	rpc := track4NodeRPC(t, n, n.config.EngineAuthToken)
	stop, reads, ready := make(chan struct{}), make(chan error, 1), make(chan struct{})
	go func() {
		if err := nativeReplayProof(rpc, anchor); err != nil {
			close(ready)
			reads <- err
			return
		}
		close(ready)
		for {
			select {
			case <-stop:
				reads <- nil
				return
			default:
			}
			if err := nativeReplayProof(rpc, anchor); err != nil {
				reads <- err
				return
			}
		}
	}()
	<-ready
	defer func() {
		close(stop)
		if err := <-reads; err != nil {
			t.Errorf("concurrent independent historical proof: %v", err)
		}
	}()
	previous := anchor
	for _, row := range rows {
		b, err := nativeReplayDecode(row)
		if err != nil {
			t.Fatal(err)
		}
		if b.ParentHash() != previous.Hash() || b.NumberU64() != previous.NumberU64()+1 {
			t.Fatal("noncontiguous replay input")
		}
		// Reset only this clone's marker so both passes below execute. Cached
		// VALID responses and an existing root index are not execution evidence.
		marker := append([]byte("validated-block/"), b.Hash().Bytes()...)
		if err := n.db.Delete(marker); err != nil {
			t.Fatal(err)
		}
		var result *execution.ProcessResult
		nativeReplayBounded(t, fmt.Sprintf("native block%d", b.NumberU64()), func() error {
			var err error
			result, err = n.blockchain.InsertSyncedBlockWithResult(b)
			return err
		})
		if result == nil || result.BlockAccessList == nil {
			t.Fatal("replay did not execute the native processor/BAL collector")
		}
		if b.Header().RequestsHash == nil || types.ComputeRequestsHash(result.Requests) != *b.Header().RequestsHash {
			t.Fatal("execution requests differ from committed reference")
		}
		p := payload.BlockToPayloadV5(b, b.Header().MixDigest, payload.WithdrawalsToEngine(b.Withdrawals()), result.BlockAccessList, *b.Header().SlotNumber)
		reqs := make([]string, len(result.Requests))
		for i, req := range result.Requests {
			reqs[i] = hexutil.Encode(req.Encode())
		}
		hashes := []types.Hash{}
		for _, tx := range b.Transactions() {
			hashes = append(hashes, tx.BlobHashes()...)
		}
		if err := n.db.Delete(marker); err != nil {
			t.Fatal(err)
		}
		nativeReplayBounded(t, fmt.Sprintf("Engine block%d", b.NumberU64()), func() error {
			for attempt := 0; attempt < 2; attempt++ {
				var status payload.PayloadStatusV1
				if err := rpc.invoke("engine_newPayloadV5", &status, p, hashes, *b.Header().ParentBeaconRoot, reqs); err != nil {
					return err
				}
				if status.Status != "VALID" || status.LatestValidHash == nil || *status.LatestValidHash != b.Hash() {
					return fmt.Errorf("payload: %+v", status)
				}
			}
			var fc payload.ForkchoiceUpdatedResult
			if err := rpc.invoke("engine_forkchoiceUpdatedV4", &fc, payload.ForkchoiceStateV1{HeadBlockHash: b.Hash(), SafeBlockHash: final, FinalizedBlockHash: final}, nil); err != nil {
				return err
			}
			if fc.PayloadStatus.Status != "VALID" {
				return fmt.Errorf("forkchoice: %+v", fc)
			}
			return nil
		})
		if n.blockchain.CurrentBlock().Hash() != b.Hash() || n.blockchain.State().GetRoot() != b.Root() {
			t.Fatal("canonical native hash/state differ from independent RLP")
		}
		previous = b
	}
	t.Logf("REPLAY blocks=%d first=%d last=%d hash=%s; native execution + repeated Engine + FCU + SDK proofs; actual48h_passed=false", len(rows), first.NumberU64(), previous.NumberU64(), previous.Hash())
}

func TestNativeMatureImportReplay(t *testing.T) {
	if *nativeReplayManifest == "" {
		t.Skip("external mature snapshot not supplied; this is not a pass for the incident replay")
	}
	data, err := os.ReadFile(*nativeReplayManifest)
	if err != nil {
		t.Fatal(err)
	}
	var input nativeReplayInput
	if err := json.Unmarshal(data, &input); err != nil {
		t.Fatal(err)
	}
	if !input.DisposableClone || !filepath.IsAbs(input.DataDir) || !filepath.IsAbs(input.Genesis) || !filepath.IsAbs(input.Rows) {
		t.Fatal("explicit disposable clone and absolute inputs required")
	}
	data, err = os.ReadFile(input.Rows)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != input.RowsSHA256 {
		t.Fatal("replay input digest differs")
	}
	var rows []nativeReplayRow
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 65536), 32<<20)
	for scanner.Scan() {
		var row nativeReplayRow
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			t.Fatal(err)
		}
		rows = append(rows, row)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	c := startupConfig(t)
	c.DataDir, c.GenesisPath = input.DataDir, input.Genesis
	c.EngineAuthToken = "isolated-native-replay"
	c.Bootnodes, c.MaxPeers = "", 0
	n, err := New(&c)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := n.Stop(); err != nil {
			t.Error(err)
		}
	})
	if err := n.Start(); err != nil {
		t.Fatal(err)
	}
	nativeReplayRun(t, n, rows)
	expected, root := n.blockchain.CurrentBlock().Hash(), n.blockchain.State().GetRoot()
	if err := n.Stop(); err != nil {
		t.Fatal(err)
	}
	reopened, err := New(&c)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := reopened.Stop(); err != nil {
			t.Error(err)
		}
	}()
	if reopened.blockchain.CurrentBlock().Hash() != expected || reopened.blockchain.State().GetRoot() != root {
		t.Fatal("durable replay restart lost the authenticated head/state")
	}
	t.Log("durable native restart passed; wall-clock soak and CL/BLS are separate")
}

func TestNativeFastReplayWithRealEmptyBlocks(t *testing.T) {
	var genesis map[string]any
	if err := json.Unmarshal(builderRequestGenesisJSON(1337), &genesis); err != nil {
		t.Fatal(err)
	}
	genesis["config"].(map[string]any)["terminalTotalDifficulty"] = 0
	genesis["config"].(map[string]any)["amsterdamTime"] = 0
	wire, err := json.Marshal(genesis)
	if err != nil {
		t.Fatal(err)
	}
	newCfg := func() *Config {
		c := startupConfig(t)
		c.EngineAuthToken = "fast-native-replay"
		c.GenesisPath = filepath.Join(c.DataDir, "genesis.json")
		if err := os.WriteFile(c.GenesisPath, wire, 0600); err != nil {
			t.Fatal(err)
		}
		return &c
	}
	producer := newTestNode(t, newCfg())
	rpc := track4NodeRPC(t, producer, producer.config.EngineAuthToken)
	var rows []nativeReplayRow
	parent := producer.blockchain.Genesis().Hash()
	for slot := uint64(1); slot <= 16; slot++ {
		attrs := map[string]any{"timestamp": hexutil.EncodeUint64(slot * 12), "slotNumber": hexutil.EncodeUint64(slot), "withdrawals": []any{}, "prevRandao": types.Hash{}, "suggestedFeeRecipient": types.Address{}, "parentBeaconBlockRoot": types.Hash{31: byte(slot)}, "targetGasLimit": "0x1c9c380"}
		var fc payload.ForkchoiceUpdatedResult
		rpc.call(t, "engine_forkchoiceUpdatedV4", &fc, payload.ForkchoiceStateV1{HeadBlockHash: parent}, attrs)
		if fc.PayloadID == nil {
			t.Fatal("empty-block build did not start")
		}
		var built struct {
			ExecutionPayload  payload.ExecutionPayloadV5
			ExecutionRequests []string
		}
		rpc.call(t, "engine_getPayloadV6", &built, fc.PayloadID)
		var status payload.PayloadStatusV1
		rpc.call(t, "engine_newPayloadV5", &status, built.ExecutionPayload, []types.Hash{}, types.Hash{31: byte(slot)}, built.ExecutionRequests)
		if status.Status != "VALID" {
			t.Fatal(status)
		}
		parent = built.ExecutionPayload.BlockHash
		rpc.call(t, "engine_forkchoiceUpdatedV4", &fc, payload.ForkchoiceStateV1{HeadBlockHash: parent}, nil)
		b := producer.blockchain.GetBlock(parent)
		if len(b.Transactions()) != 0 {
			t.Fatal("expected real empty blocks")
		}
		raw, err := b.EncodeRLP()
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, nativeReplayRow{parent, raw})
	}
	receiver := newTestNode(t, newCfg())
	nativeReplayRun(t, receiver, rows)
	broken := rows[0]
	broken.Hash[0] ^= 1
	if _, err := nativeReplayDecode(broken); err == nil {
		t.Fatal("changed independent block identity accepted")
	}
}
