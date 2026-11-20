package node

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
)

// These are supplied, independently authenticated official fixture pre-states.
// The actual HTTP/RPC/native EVM runs below; this is not a claim of native PoW
// chain import, and no production ingress bypass is introduced.
func officialFixtureNode(t *testing.T) (*Node, track4RPC) {
	t.Helper()
	directory := filepath.Join("testdata", "execution-api-a9e9095c")
	cfg := makeTestConfig(t)
	cfg.GenesisPath = filepath.Join(directory, "genesis.json")
	cfg.EngineAuthToken = "official-rpc-fixture"
	n := newTestNode(t, &cfg)
	file, err := os.Open(filepath.Join(directory, "chain.rlp"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	stream := gethrlp.NewStream(file, 0)
	var blocks []*types.Block
	batch := n.db.NewBatch()
	for {
		var raw gethrlp.RawValue
		if err := stream.Decode(&raw); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		block, err := types.DecodeBlockRLP(raw)
		if err != nil {
			t.Fatal(err)
		}
		var fields []gethrlp.RawValue
		if err := gethrlp.DecodeBytes(raw, &fields); err != nil {
			t.Fatal(err)
		}
		body, err := gethrlp.EncodeToBytes(fields[1:])
		if err != nil {
			t.Fatal(err)
		}
		if err := rawdb.WriteHeader(batch, block.NumberU64(), block.Hash(), fields[0]); err != nil {
			t.Fatal(err)
		}
		if err := rawdb.WriteBody(batch, block.NumberU64(), block.Hash(), body); err != nil {
			t.Fatal(err)
		}
		if err := batch.Put(append([]byte("validated-block/"), block.Hash().Bytes()...), []byte{1}); err != nil {
			t.Fatal(err)
		}
		blocks = append(blocks, block)
	}
	if len(blocks) != 45 {
		t.Fatalf("official block count=%d", len(blocks))
	}
	if err := batch.Write(); err != nil {
		t.Fatal(err)
	}
	for _, number := range []uint64{1, 44, 45} {
		file, err := os.Open(filepath.Join(directory, fmt.Sprintf("state-%d.json.gz", number)))
		if err != nil {
			t.Fatal(err)
		}
		reader, err := gzip.NewReader(file)
		if err != nil {
			t.Fatal(err)
		}
		var wire struct {
			Root     string
			Accounts map[string][]byte
			Storage  map[string]map[string][]byte
			Codes    map[string][]byte
		}
		if err := json.NewDecoder(reader).Decode(&wire); err != nil {
			t.Fatal(err)
		}
		reader.Close()
		file.Close()
		snapshot := state.NewStateSnapshot()
		for key, value := range wire.Accounts {
			snapshot.Accounts[types.HexToHash(key)] = value
		}
		for address, slots := range wire.Storage {
			a := types.HexToHash(address)
			snapshot.Storage[a] = make(map[types.Hash][]byte)
			for key, value := range slots {
				snapshot.Storage[a][types.HexToHash(key)] = value
			}
		}
		for key, value := range wire.Codes {
			snapshot.Codes[types.HexToHash(key)] = value
		}
		root := blocks[number-1].Root()
		if types.HexToHash(wire.Root) != root {
			t.Fatal("fixture state/header commitment mismatch")
		}
		if err := snapshot.Validate(root); err != nil {
			t.Fatal(err)
		}
		if err := state.StoreStateSnapshot(n.db, root, snapshot); err != nil {
			t.Fatal(err)
		}
	}
	head := blocks[44]
	if err := n.blockchain.ApplyForkchoice(head.Hash(), head.Hash(), head.Hash()); err != nil {
		t.Fatal(err)
	}
	return n, track4NodeRPC(t, n, cfg.EngineAuthToken)
}

func TestOfficialExecutionAPICallEstimateAccessAndSimulateFixtures(t *testing.T) {
	_, rpc := officialFixtureNode(t)
	directory := filepath.Join("testdata", "execution-api-a9e9095c")
	files := 0
	requests := 0
	for _, method := range []string{"eth_call", "eth_estimateGas", "eth_createAccessList", "eth_simulateV1"} {
		paths, err := filepath.Glob(filepath.Join(directory, method, "*.io"))
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range paths {
			files++
			t.Run(method+"/"+filepath.Base(path), func(t *testing.T) {
				file, err := os.Open(path)
				if err != nil {
					t.Fatal(err)
				}
				defer file.Close()
				scanner := bufio.NewScanner(file)
				scanner.Buffer(make([]byte, 4096), 4<<20)
				var request []byte
				for scanner.Scan() {
					line := scanner.Text()
					if strings.HasPrefix(line, ">> ") {
						request = []byte(strings.TrimPrefix(line, ">> "))
					}
					if !strings.HasPrefix(line, "<< ") {
						continue
					}
					if request == nil {
						t.Fatal("expected response without source request")
					}
					requests++
					req, err := http.NewRequest(http.MethodPost, rpc.url, bytes.NewReader(request))
					if err != nil {
						t.Fatal(err)
					}
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("Authorization", "Bearer "+rpc.token)
					response, err := http.DefaultClient.Do(req)
					if err != nil {
						t.Fatal(err)
					}
					raw, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
					response.Body.Close()
					if err != nil {
						t.Fatal(err)
					}
					var actual, expected any
					if err := json.Unmarshal(raw, &actual); err != nil {
						t.Fatal(err)
					}
					if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "<< ")), &expected); err != nil {
						t.Fatal(err)
					}
					record, err := json.Marshal(map[string]any{"fixture": filepath.Base(path), "method": method, "request": json.RawMessage(request), "response": json.RawMessage(raw)})
					if err != nil {
						t.Fatal(err)
					}
					t.Logf("OFFICIAL_RPC %s", record)
					if !reflect.DeepEqual(actual, expected) {
						t.Errorf("original request: %s\nactual: %s\nexpected: %s", request, raw, strings.TrimPrefix(line, "<< "))
					}
					request = nil
				}
				if err := scanner.Err(); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
	if files != 107 || requests != 108 {
		t.Fatalf("official input count changed: files=%d requests=%d", files, requests)
	}
}
