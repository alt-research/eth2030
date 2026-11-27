package node

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
	"github.com/eth2030/eth2030/engine/payload"
	"github.com/eth2030/eth2030/focil/heze"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

type nativeHezeView struct {
	clock heze.Clock
	state heze.DependentState
}

func (v *nativeHezeView) Clock() (heze.Clock, error) { return v.clock, nil }
func (v *nativeHezeView) DependentState(uint64, types.Hash) (heze.DependentState, error) {
	return v.state, nil
}

func TestGroupEHezeGossipNativeEngineForkchoiceAndRestart(t *testing.T) {
	const token = "heze-native-integration"
	key, _ := gethcrypto.HexToECDSA("0000000000000000000000000000000000000000000000000000000000000001")
	to := types.Address{19: 0x88}
	var genesis map[string]any
	if err := json.Unmarshal(builderRequestGenesisJSON(1337), &genesis); err != nil {
		t.Fatal(err)
	}
	c := genesis["config"].(map[string]any)
	c["terminalTotalDifficulty"], c["amsterdamTime"], c["bogotaTime"] = 0, 0, 0
	alloc := genesis["alloc"].(map[string]any)
	alloc[gethcrypto.PubkeyToAddress(key.PublicKey).Hex()] = map[string]string{"balance": "1000000000000000000"}
	alloc[to.Hex()] = map[string]string{"balance": "1"}
	wire, _ := json.Marshal(genesis)
	newConfig := func() *Config {
		cfg := makeTestConfig(t)
		cfg.EngineAuthToken, cfg.GenesisPath = token, filepath.Join(cfg.DataDir, "genesis.json")
		if err := os.WriteFile(cfg.GenesisPath, wire, 0600); err != nil {
			t.Fatal(err)
		}
		return &cfg
	}
	producer := newTestNode(t, newConfig())
	receiverConfig := newConfig()
	receiver := newTestNode(t, receiverConfig)
	source, destination := track4NodeRPC(t, producer, token), track4NodeRPC(t, receiver, token)
	v := &nativeHezeView{clock: heze.Clock{Slot: 1, IntoSlotMS: 8999, SlotDurationMS: 12000}, state: heze.DependentState{KnownBlock: true, StateAvailable: true, PossibleDependent: true, ForkVersion: [4]byte{8}, ValidatorPubkeys: map[uint64][48]byte{}}}
	for i := range v.state.Committee {
		v.state.Committee[i] = uint64(i)
		v.state.ValidatorPubkeys[uint64(i)] = crypto.BLSPubkeyFromSecret(big.NewInt(int64(i + 1)))
	}
	branch := types.Hash{31: 0xab}
	listKey := heze.Key{Slot: 1, DependentRoot: branch}
	raws := make([][]byte, 2)
	for i := range raws {
		target := gethcrypto.PubkeyToAddress(key.PublicKey)
		copy(target[:], to[:])
		tx, err := gethtypes.SignTx(gethtypes.NewTx(&gethtypes.DynamicFeeTx{ChainID: big.NewInt(1337), Nonce: uint64(i), Gas: 50000, GasFeeCap: big.NewInt(2_000_000_000), GasTipCap: big.NewInt(1), To: &target, Value: big.NewInt(1)}), gethtypes.LatestSignerForChainID(big.NewInt(1337)), key)
		if err != nil {
			t.Fatal(err)
		}
		raws[i], err = tx.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
	}
	signed := func(index uint64, txs [][]byte) []byte {
		s := heze.SignedInclusionList{Message: heze.InclusionList{Slot: 1, ValidatorIndex: index, DependentRoot: branch, Transactions: txs}}
		root := s.Message.SigningRoot(v.state.ForkVersion, v.state.GenesisValidatorsRoot)
		s.Signature = crypto.BLSSign(big.NewInt(int64(index+1)), root[:])
		b, err := s.MarshalSSZ()
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	rawIL := signed(7, [][]byte{raws[0], {0xff}})
	if err := receiver.HandleHezeInclusionList(rawIL); err == nil {
		t.Fatal("unauthenticated CL context accepted")
	}
	for _, n := range []*Node{producer, receiver} {
		if err := n.ConfigureHezeConsensus(v); err != nil {
			t.Fatal(err)
		}
		if err := n.HandleHezeInclusionList(rawIL); err != nil {
			t.Fatal(err)
		}
	}
	v.clock.IntoSlotMS = 9000
	for _, n := range []*Node{producer, receiver} {
		if err := n.HandleHezeInclusionList(signed(8, [][]byte{raws[1]})); err != nil {
			t.Fatal(err)
		}
	}
	timely, err := receiver.HezeInclusionListTransactions(listKey, true)
	if err != nil || len(timely) != 2 {
		t.Fatal(timely, err)
	}
	all, err := producer.HezeInclusionListTransactions(listKey, false)
	if err != nil || len(all) != 3 {
		t.Fatal(all, err)
	}
	hexList := func(txs [][]byte) []string {
		out := make([]string, len(txs))
		for i, b := range txs {
			out[i] = "0x" + hex.EncodeToString(b)
		}
		return out
	}
	beacon := types.Hash{31: 1}
	build := func(timestamp string, list [][]byte) bogotaIntegrationPayload {
		var fc payload.ForkchoiceUpdatedResult
		source.call(t, "engine_forkchoiceUpdatedV5", &fc, payload.ForkchoiceStateV1{HeadBlockHash: producer.Blockchain().Genesis().Hash()}, map[string]any{"timestamp": timestamp, "slotNumber": "0x2", "withdrawals": []any{}, "prevRandao": types.Hash{}, "suggestedFeeRecipient": types.Address{}, "parentBeaconBlockRoot": beacon, "targetGasLimit": "0x1c9c380", "inclusionListTransactions": hexList(list)})
		if fc.PayloadID == nil {
			t.Fatal(fc)
		}
		var envelope bogotaIntegrationPayload
		source.call(t, "engine_getPayloadV6", &envelope, fc.PayloadID)
		return envelope
	}
	empty := build("0x18", nil)
	var status bogotaIntegrationStatus
	destination.call(t, "engine_newPayloadV6", &status, empty.ExecutionPayload, []types.Hash{}, beacon, empty.ExecutionRequests, hexList(timely))
	if status.Status != "VALID" || status.InclusionListSatisfied == nil || *status.InclusionListSatisfied || !receiver.Blockchain().HasValidatedBlock(empty.ExecutionPayload.BlockHash) {
		t.Fatalf("omission invalidated payload or escaped: %+v", status)
	}
	decisions := heze.NewPayloadDecisions(receiver.Blockchain().Genesis().Hash())
	if err := decisions.Record(empty.ExecutionPayload.BlockHash, heze.Valid, status.InclusionListSatisfied, true); err != nil {
		t.Fatal(err)
	}
	if decisions.ShouldExtend(empty.ExecutionPayload.BlockHash, heze.ExtensionContext{PayloadSlot: 2, CurrentSlot: 3, PayloadTimely: true, PayloadDataAvailable: true}) {
		t.Fatal("omitted IL head extended")
	}
	built := build("0x19", all)
	if len(built.ExecutionPayload.Transactions) != 2 {
		t.Fatalf("private IL did not build both transactions: %v", built.ExecutionPayload.Transactions)
	}
	for _, rpc := range []track4RPC{source, destination} {
		rpc.call(t, "engine_newPayloadV6", &status, built.ExecutionPayload, []types.Hash{}, beacon, built.ExecutionRequests, hexList(all))
		if status.Status != "VALID" || status.InclusionListSatisfied == nil || !*status.InclusionListSatisfied {
			t.Fatal(status)
		}
		var fc struct{ PayloadStatus bogotaIntegrationStatus }
		rpc.call(t, "engine_forkchoiceUpdatedV5", &fc, payload.ForkchoiceStateV1{HeadBlockHash: built.ExecutionPayload.BlockHash}, nil)
		if fc.PayloadStatus.Status != "VALID" || fc.PayloadStatus.InclusionListSatisfied == nil || !*fc.PayloadStatus.InclusionListSatisfied {
			t.Fatal(fc)
		}
	}
	request := heze.Request{Slot: 1, DependentRoot: branch}
	request.Indices.Set(7)
	requestBytes := request.MarshalSSZ()
	chunks, err := receiver.HezeInclusionListsByIndices(requestBytes, 0)
	if err != nil || len(chunks) != 1 || !reflect.DeepEqual(chunks[0], rawIL) {
		t.Fatal("request response changed", err)
	}
	if err := receiver.Stop(); err != nil {
		t.Fatal(err)
	}
	reopened := newTestNode(t, receiverConfig)
	if err := reopened.ConfigureHezeConsensus(v); err != nil {
		t.Fatal(err)
	}
	restored, err := reopened.HezeInclusionListTransactions(listKey, false)
	if err != nil || !reflect.DeepEqual(all, restored) {
		t.Fatal("authenticated IL view lost on restart", err)
	}
	var fc struct{ PayloadStatus bogotaIntegrationStatus }
	track4NodeRPC(t, reopened, token).call(t, "engine_forkchoiceUpdatedV5", &fc, payload.ForkchoiceStateV1{HeadBlockHash: built.ExecutionPayload.BlockHash}, nil)
	if fc.PayloadStatus.Status != "VALID" || fc.PayloadStatus.InclusionListSatisfied == nil || !*fc.PayloadStatus.InclusionListSatisfied {
		t.Fatal("payload decision lost on restart", fc)
	}
}
