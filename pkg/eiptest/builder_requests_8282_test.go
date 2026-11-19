package eiptest_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eth2030/eth2030/core/eips"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/eiptest"
	"github.com/eth2030/eth2030/engine"
)

func TestEIP8282BuilderRequestsThroughActualEngine(t *testing.T) {
	var fixture struct {
		Address  string   `json:"execution_address"`
		Calldata string   `json:"deposit_calldata"`
		Pubkey   string   `json:"pubkey"`
		Requests []string `json:"execution_requests"`
	}
	raw, err := os.ReadFile("testdata/eip8282/builder-requests.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	decode := func(value string) []byte {
		t.Helper()
		out, err := hex.DecodeString(strings.TrimPrefix(value, "0x"))
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	s := eiptest.New(t, eiptest.ForkAt(eiptest.ForkAmsterdam, 0))
	probe := s.Engine()
	if s.Alice().Address != types.HexToAddress(fixture.Address) {
		t.Fatal("fixture withdrawal/exit address differs from signer")
	}
	// The first empty system call enables the constructor-inhibited exit queue.
	if _, status, err := probe.BuildAndProcess(); err != nil || status.Status != engine.StatusValid {
		t.Fatalf("queue activation: %v status=%+v", err, status)
	}
	depositAddress, exitAddress := types.BuilderDepositRequestAddress, types.BuilderExitRequestAddress
	deposit := eiptest.DynamicFee(s.Config().ChainID, s.Alice(), 0, &depositAddress, new(big.Int).Add(eiptest.Ether, big.NewInt(1)), decode(fixture.Calldata), 1500000, big.NewInt(10000000000), big.NewInt(2000000000))
	exit := eiptest.DynamicFee(s.Config().ChainID, s.Alice(), 1, &exitAddress, big.NewInt(1), decode(fixture.Pubkey), 500000, big.NewInt(10000000000), big.NewInt(1000000000))
	probe.Submit(deposit, exit)
	attributes := probe.Attributes()
	id, err := probe.FCUWith(attributes)
	if err != nil {
		t.Fatal(err)
	}
	built, err := probe.GetPayload(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(built.ExecutionPayload.Transactions) != 2 || len(built.ExecutionRequests) != 2 {
		t.Fatalf("builder did not include both real transactions/queues: tx=%d requests=%d", len(built.ExecutionPayload.Transactions), len(built.ExecutionRequests))
	}
	for i, expected := range fixture.Requests {
		if !bytes.Equal(built.ExecutionRequests[i], decode(expected)) {
			t.Fatalf("Engine request %d differs from frozen CL wire bytes", i)
		}
	}
	parent := probe.HeadHash()
	bad := make([][]byte, len(built.ExecutionRequests))
	for i, item := range built.ExecutionRequests {
		bad[i] = bytes.Clone(item)
	}
	bad[0][81] ^= 1
	rejected, err := probe.ProcessWith(built.ExecutionPayload, attributes.ParentBeaconBlockRoot, bad)
	if err != nil || rejected.Status != engine.StatusInvalid || probe.HeadHash() != parent {
		t.Fatalf("altered builder requests were accepted: %v %+v", err, rejected)
	}
	accepted, err := probe.Process(built.ExecutionPayload)
	if err != nil {
		t.Fatal(err)
	}
	probe.ExpectStatusValid(t, accepted)
}

func TestEIP8282OwnedGloasGenesisAllocations(t *testing.T) {
	files, err := filepath.Glob("../devnet/kurtosis/configs/*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, filename := range files {
		data, err := os.ReadFile(filename)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(data, []byte("gloas_fork_epoch:")) {
			continue
		}
		checked++
		var encoded string
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "additional_preloaded_contracts:") {
				encoded = strings.TrimSpace(strings.TrimPrefix(line, "additional_preloaded_contracts:"))
			}
		}
		var alloc map[string]struct {
			Code    string            `json:"code"`
			Nonce   int               `json:"nonce"`
			Storage map[string]string `json:"storage"`
		}
		if err = json.Unmarshal([]byte(encoded), &alloc); err != nil {
			t.Fatalf("%s: explicit canonical pre-root allocations absent: %v", filename, err)
		}
		for _, item := range []struct {
			address types.Address
			code    []byte
		}{{types.BuilderDepositRequestAddress, eips.BuilderDepositRuntime()}, {types.BuilderExitRequestAddress, eips.BuilderExitRuntime()}} {
			account, ok := alloc[item.address.Hex()]
			if !ok || account.Nonce != 1 || account.Code != "0x"+hex.EncodeToString(item.code) {
				t.Fatalf("%s: wrong canonical allocation at %s", filename, item.address)
			}
		}
		exit := alloc[types.BuilderExitRequestAddress.Hex()]
		if exit.Storage[(types.Hash{}).Hex()] != "0x"+strings.Repeat("ff", 32) {
			t.Fatalf("%s: exit constructor inhibitor missing", filename)
		}
	}
	if checked == 0 {
		t.Fatal("no owned Gloas genesis profiles qualified")
	}
}
