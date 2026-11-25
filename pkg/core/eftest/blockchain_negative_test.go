package eftest

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/eth2030/eth2030/core/types"
	"math/big"
	"strconv"
	"testing"
)

func blockchainFixtureCopy(t *testing.T) *BlockchainFixture {
	t.Helper()
	values, err := LoadBlockchainFixtures("testdata/blockchain/bal.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range values {
		return value
	}
	t.Fatal("missing reference")
	return nil
}
func TestBlockchainReferenceMismatchesCannotPass(t *testing.T) {
	for _, field := range []string{"header", "post_state", "receipt", "bal", "exception"} {
		t.Run(field, func(t *testing.T) {
			fixture := blockchainFixtureCopy(t)
			switch field {
			case "header":
				fixture.Blocks[0].Header["hash"] = json.RawMessage(`"0x0000000000000000000000000000000000000000000000000000000000000000"`)
			case "post_state":
				for a, v := range fixture.Post {
					n, err := fixtureNumber(v.Balance, 256)
					if err != nil {
						t.Fatal(err)
					}
					n.Add(n, big.NewInt(1))
					v.Balance = "0x" + n.Text(16)
					fixture.Post[a] = v
					break
				}
			case "receipt":
				fixture.Blocks[0].Receipts[0].RLP = "0x80"
			case "bal":
				fixture.Blocks[0].BAL = json.RawMessage(`[]`)
			case "exception":
				fixture.Blocks[0].ExpectedException = "UnknownException"
			}
			if result := fixture.RunNative(); result.Passed || result.Error == "" {
				t.Fatalf("altered %s accepted: %+v", field, result)
			}
		})
	}
}
func TestBlockchainUnknownRejectionsNeverMatch(t *testing.T) {
	fixture := blockchainFixtureCopy(t)
	cfg, err := fixture.nativeConfig()
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"UnknownException", "TransactionException.INVALID_SIGNATURE_VRS", "BlockException.INVALID_REQUESTS", "BlockException.INVALID_BLOCK_ACCESS_LIST"} {
		if blockchainExceptionMatches(expected, errors.New("unrelated database failure"), nil, nil, cfg) || blockchainMultiFaultRejection(expected, errors.New("unrelated database failure"), nil, cfg) {
			t.Fatalf("unrelated error satisfies %s", expected)
		}
	}
}

func TestNativeBlockchainRejectsMissingAmsterdamSlot(t *testing.T) {
	fixture := blockchainFixtureCopy(t)
	raw, err := fixtureBytes(fixture.Blocks[0].RLP, -1)
	if err != nil {
		t.Fatal(err)
	}
	original, err := types.DecodeBlockRLP(raw)
	if err != nil {
		t.Fatal(err)
	}
	header := original.Header()
	header.SlotNumber = nil
	altered := types.NewBlock(header, original.Body())
	encoded, err := altered.EncodeRLP()
	if err != nil {
		t.Fatal(err)
	}
	fixture.Blocks[0].RLP = "0x" + fmt.Sprintf("%x", encoded)
	delete(fixture.Blocks[0].Header, "slotNumber")
	fixture.Blocks[0].Header["hash"] = json.RawMessage(strconv.Quote(altered.Hash().Hex()))
	fixture.LastHash = altered.Hash().Hex()
	if result := fixture.RunNative(); result.Passed {
		t.Fatal("accepted Amsterdam header without its required CL slot")
	}
}
