package execution

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"strings"
	"testing"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/eips"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
)

func builderRequestHeader() *types.Header {
	return &types.Header{Number: big.NewInt(1), Time: 12, GasLimit: 30000000, Difficulty: new(big.Int), BaseFee: big.NewInt(1)}
}
func callBuilderContract(t *testing.T, db state.StateDB, cfg *config.ChainConfig, header *types.Header, caller, address types.Address, input []byte, value *big.Int) ([]byte, error) {
	t.Helper()
	if finalizer, ok := db.(interface{ FinalizePreState() }); ok {
		finalizer.FinalizePreState()
	}
	evm := vm.NewEVMWithState(vm.BlockContext{BlockNumber: header.Number, Time: header.Time, GasLimit: header.GasLimit, BaseFee: header.BaseFee}, vm.TxContext{Origin: caller, GasPrice: new(big.Int)}, vm.Config{}, db)
	if _, err := ConfigureBlockEVM(cfg, header, evm); err != nil {
		t.Fatal(err)
	}
	result, _, err := evm.Call(caller, address, input, 2000000, value)
	return result, err
}
func builderDepositFixture(number byte) *types.BuilderDepositRequest {
	request := &types.BuilderDepositRequest{Amount: 1000000000}
	request.Pubkey[47] = number
	request.WithdrawalCredentials[0] = 0xb0
	request.Signature[95] = number
	return request
}
func TestEIP8282MandatoryContractsAndForkBoundary(t *testing.T) {
	cfg := *config.TestConfig
	activation := uint64(100)
	cfg.AmsterdamTime = &activation
	header := builderRequestHeader()
	header.Time = 99
	db := state.NewMemoryStateDB()
	if _, err := ProcessRequests(&cfg, db, header); err != nil {
		t.Fatalf("pre-fork unexpectedly required builder contracts: %v", err)
	}
	header.Time = 100
	if _, err := ProcessRequests(&cfg, db, header); err == nil || !strings.Contains(err.Error(), "request type 3") {
		t.Fatalf("missing deposit contract did not invalidate activation: %v", err)
	}
	db.SetCode(types.BuilderDepositRequestAddress, eips.BuilderDepositRuntime())
	if _, err := ProcessRequests(&cfg, db, header); err == nil || !strings.Contains(err.Error(), "request type 4") {
		t.Fatalf("missing exit contract did not invalidate activation: %v", err)
	}
}
func TestEIP8282NativeRecordsFeesAndRequestHash(t *testing.T) {
	db := newBuilderRequestTestState()
	header := builderRequestHeader()
	cfg := config.TestConfig
	sender := types.HexToAddress("0x123456")
	db.AddBalance(sender, new(big.Int).Exp(big.NewInt(10), big.NewInt(22), nil))
	request := builderDepositFixture(1)
	request.Amount = 0x0102030405060708
	stake := new(big.Int).Mul(new(big.Int).SetUint64(request.Amount), big.NewInt(1000000000))
	db.AddBalance(sender, new(big.Int).Mul(stake, big.NewInt(2)))
	// The constructor inhibits exits, while deposits are active before the fork.
	if _, err := callBuilderContract(t, db, cfg, header, sender, types.BuilderExitRequestAddress, request.Pubkey[:], big.NewInt(1)); err == nil {
		t.Fatal("exit write bypassed constructor inhibitor")
	}
	for _, data := range [][]byte{request.EncodeCalldata()[:183], append(request.EncodeCalldata(), 0)} {
		if _, err := callBuilderContract(t, db, cfg, header, sender, types.BuilderDepositRequestAddress, data, new(big.Int).Add(stake, big.NewInt(1))); err == nil {
			t.Fatal("invalid deposit size accepted")
		}
	}
	if _, err := callBuilderContract(t, db, cfg, header, sender, types.BuilderDepositRequestAddress, request.EncodeCalldata(), stake); err == nil {
		t.Fatal("stake without request fee accepted")
	}
	if _, err := callBuilderContract(t, db, cfg, header, sender, types.BuilderDepositRequestAddress, request.EncodeCalldata(), new(big.Int).Add(stake, big.NewInt(1))); err != nil {
		t.Fatal(err)
	}
	var anonymous bool
	for _, log := range db.GetLogs(types.Hash{}) {
		if log.Address == types.BuilderDepositRequestAddress && len(log.Topics) == 0 {
			anonymous = bytes.Equal(log.Data, request.EncodeCalldata())
		}
	}
	if !anonymous {
		t.Fatal("deposit log did not preserve the submitted big-endian record")
	}
	requests, err := ProcessRequests(cfg, db, header)
	if err != nil || len(requests) != 1 || requests[0].Type != 3 || !bytes.Equal(requests[0].Data, request.Encode()) {
		t.Fatalf("deposit dequeue/SSZ mismatch: %v requests=%+v", err, requests)
	}
	if binary.LittleEndian.Uint64(requests[0].Data[80:88]) != request.Amount {
		t.Fatal("dequeued amount is not SSZ little-endian")
	}
	if _, err = callBuilderContract(t, db, cfg, header, sender, types.BuilderExitRequestAddress, request.Pubkey[:], big.NewInt(1)); err != nil {
		t.Fatal(err)
	}
	requests, err = ProcessRequests(cfg, db, header)
	expected := (&types.BuilderExitRequest{SourceAddress: sender, Pubkey: request.Pubkey}).Encode()
	if err != nil || len(requests) != 1 || requests[0].Type != 4 || !bytes.Equal(requests[0].Data, expected) {
		t.Fatalf("exit authorization source/record mismatch: %v", err)
	}
	independent := gethtypes.CalcRequestsHash(requests.Encode())
	if types.Hash(independent) != types.ComputeRequestsHash(requests) {
		t.Fatal("independent requests commitment differs")
	}
}
func TestEIP8282QueueDrainCapsAndRollback(t *testing.T) {
	db := newBuilderRequestTestState()
	header := builderRequestHeader()
	cfg := config.TestConfig
	sender := types.HexToAddress("0x123456")
	db.AddBalance(sender, new(big.Int).Exp(big.NewInt(10), big.NewInt(24), nil))
	if _, err := ProcessRequests(cfg, db, header); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 65; i++ {
		request := builderDepositFixture(byte(i + 1))
		feeBytes, err := callBuilderContract(t, db, cfg, header, sender, types.BuilderDepositRequestAddress, nil, new(big.Int))
		if err != nil {
			t.Fatal(err)
		}
		value := new(big.Int).Add(big.NewInt(1000000000000000000), new(big.Int).SetBytes(feeBytes))
		if _, err = callBuilderContract(t, db, cfg, header, sender, types.BuilderDepositRequestAddress, request.EncodeCalldata(), value); err != nil {
			t.Fatalf("deposit %d: %v", i, err)
		}
	}
	for i := 0; i < 17; i++ {
		pubkey := builderDepositFixture(byte(i + 1)).Pubkey
		fee, err := callBuilderContract(t, db, cfg, header, sender, types.BuilderExitRequestAddress, nil, new(big.Int))
		if err != nil {
			t.Fatal(err)
		}
		if _, err = callBuilderContract(t, db, cfg, header, sender, types.BuilderExitRequestAddress, pubkey[:], new(big.Int).SetBytes(fee)); err != nil {
			t.Fatal(err)
		}
	}
	before := db.GetRoot()
	exitCode := db.GetCode(types.BuilderExitRequestAddress)
	db.SetCode(types.BuilderExitRequestAddress, []byte{0xfe})
	failedRoot := db.GetRoot()
	if _, err := ProcessRequests(cfg, db, header); err == nil {
		t.Fatal("failed exit system call accepted")
	}
	if db.GetRoot() != failedRoot {
		t.Fatal("earlier deposit drain survived a later queue failure")
	}
	db.SetCode(types.BuilderExitRequestAddress, exitCode)
	if db.GetRoot() != before {
		t.Fatal("rollback changed pending state")
	}
	requests, err := ProcessRequests(cfg, db, header)
	if err != nil || len(requests) != 2 || len(requests[0].Data) != 64*184 || len(requests[1].Data) != 16*68 {
		t.Fatalf("drain caps: %v requests=%+v", err, requests)
	}
	if requests[0].Type != 3 || requests[1].Type != 4 {
		t.Fatal("builder request order is not canonical")
	}
	requests, err = ProcessRequests(cfg, db, header)
	if err != nil || len(requests) != 2 || len(requests[0].Data) != 184 || len(requests[1].Data) != 68 {
		t.Fatalf("retained queue tail lost: %v", err)
	}
	requests, err = ProcessRequests(cfg, db, header)
	if err != nil || len(requests) != 0 {
		t.Fatal("drained queue did not become empty")
	}
}

func TestEIP8282PreforkQueueAndDynamicFee(t *testing.T) {
	db := newBuilderRequestTestState()
	cfg := *config.TestConfig
	activation := uint64(100)
	cfg.AmsterdamTime = &activation
	header := builderRequestHeader()
	header.Time = 99
	sender := types.HexToAddress("0x123456")
	db.AddBalance(sender, new(big.Int).Exp(big.NewInt(10), big.NewInt(24), nil))
	address := types.BuilderDepositRequestAddress
	// The getter is read-only, including rejection of value-bearing calls.
	before := db.GetRoot()
	if _, err := callBuilderContract(t, db, &cfg, header, sender, address, nil, big.NewInt(1)); err == nil || db.GetRoot() != before {
		t.Fatal("value-bearing fee getter did not revert without state changes")
	}
	request := builderDepositFixture(7)
	tooSmall := *request
	tooSmall.Amount--
	if _, err := callBuilderContract(t, db, &cfg, header, sender, address, tooSmall.EncodeCalldata(), big.NewInt(1000000000000000001)); err == nil {
		t.Fatal("deposit below one ETH was accepted")
	}
	for count := uint64(0); count < 30; count++ {
		// Independent integer-series definition from the selected EIP.
		excess := uint64(0)
		if count > 8 {
			excess = count - 8
		}
		term, total := uint64(17), uint64(0)
		for i := uint64(1); term != 0; i++ {
			total += term
			term = term * excess / (17 * i)
		}
		expectedFee := new(big.Int).SetUint64(total / 17)
		root := db.GetRoot()
		fee, err := callBuilderContract(t, db, &cfg, header, sender, address, nil, new(big.Int))
		if err != nil || new(big.Int).SetBytes(fee).Cmp(expectedFee) != 0 || db.GetRoot() != root {
			t.Fatalf("count %d: fee/getter state differs: %x %v", count, fee, err)
		}
		value := new(big.Int).Add(big.NewInt(1000000000000000000), expectedFee)
		if _, err = callBuilderContract(t, db, &cfg, header, sender, address, request.EncodeCalldata(), value); err != nil {
			t.Fatal(err)
		}
	}
	queuedRoot := db.GetRoot()
	if requests, err := ProcessRequests(&cfg, db, header); err != nil || len(requests) != 0 || db.GetRoot() != queuedRoot {
		t.Fatal("pre-fork finalization drained builder deposits")
	}
	header.Time = activation
	requests, err := ProcessRequests(&cfg, db, header)
	if err != nil || len(requests) != 1 || len(requests[0].Data) != 30*184 {
		t.Fatalf("pre-fork deposits did not survive activation: %v", err)
	}
	if db.GetState(address, types.Hash{}) != types.BytesToHash([]byte{22}) || db.GetState(address, types.BytesToHash([]byte{1})) != (types.Hash{}) || db.GetState(address, types.BytesToHash([]byte{2})) != (types.Hash{}) || db.GetState(address, types.BytesToHash([]byte{3})) != (types.Hash{}) {
		t.Fatal("system call did not update excess/reset count and empty queue indices")
	}
}

// EIP-7685 leaves request_data opaque at the EL boundary. CL record/count
// admission is independent of whether a successful custom system call executes.
func TestEIP8282OpaqueSystemOutput(t *testing.T) {
	for _, size := range []uint64{1, 65 * 184} {
		db := newBuilderRequestTestState()
		code := []byte{0x61, byte(size >> 8), byte(size), 0x5f, 0xf3}
		db.SetCode(types.BuilderDepositRequestAddress, code)
		requests, err := ProcessRequests(config.TestConfig, db, builderRequestHeader())
		if err != nil || len(requests) != 1 || requests[0].Type != types.BuilderDepositRequestType || len(requests[0].Data) != int(size) {
			t.Fatalf("opaque type-3 output %d: requests=%+v err=%v", size, requests, err)
		}
		for _, b := range requests[0].Data {
			if b != 0 {
				t.Fatal("opaque system output changed")
			}
		}
	}
}

func TestEIP8282FailedSystemCallRevertsPostBlockEffects(t *testing.T) {
	db := newBuilderRequestTestState()
	db.SetCode(types.BuilderDepositRequestAddress, []byte{0x60, 0x01, 0x5f, 0x55, 0x5f, 0x5f, 0xfd})
	root := db.GetRoot()
	if _, err := ProcessRequests(config.TestConfig, db, builderRequestHeader()); err == nil || db.GetRoot() != root {
		t.Fatalf("failing system call accepted or retained earlier queue effects: %v", err)
	}
}
