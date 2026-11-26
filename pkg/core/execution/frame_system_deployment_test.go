package execution

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/types"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func TestFrameSystemContractsOrdinaryKeylessDeployment(t *testing.T) {
	raw, err := os.ReadFile("../../../evidence/hegota-b/deployments.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors []struct {
		EIP         int             `json:"eip"`
		Transaction json.RawMessage `json:"transaction"`
	}
	if err = json.Unmarshal(raw, &vectors); err != nil {
		t.Fatal(err)
	}
	for _, vector := range vectors {
		t.Run(fmt.Sprintf("EIP_%d", vector.EIP), func(t *testing.T) {
			var upstream gethtypes.Transaction
			if err := json.Unmarshal(vector.Transaction, &upstream); err != nil {
				t.Fatal(err)
			}
			signer, err := gethtypes.Sender(gethtypes.HomesteadSigner{}, &upstream)
			if err != nil {
				t.Fatal(err)
			}
			address := types.Address(gethcrypto.CreateAddress(signer, 0))
			want, code := types.NonceManager8250, types.NonceManagerCode8250
			if vector.EIP == 8272 {
				want, code = types.RecentRootAddress, types.RecentRootCode
			}
			if address != want {
				t.Fatalf("keyless address %s want %s", address, want)
			}
			_, db, cfg, header := frameExecutionFixture(t)
			// Funding is explicit; activation itself neither deploys nor checks accounts.
			db.AddBalance(types.Address(signer), new(big.Int).Exp(big.NewInt(10), big.NewInt(21), nil))
			wire, err := upstream.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			transaction, err := types.DecodeTxRLP(wire)
			if err != nil {
				t.Fatal(err)
			}
			receipt, _, err := ApplyTransaction(cfg, db, header, transaction, new(gaspool.GasPool).AddGas(header.GasLimit))
			if err != nil || receipt.Status != 1 {
				t.Fatalf("ordinary deployment failed: %+v %v", receipt, err)
			}
			if db.GetNonce(address) != 1 || !bytes.Equal(db.GetCode(address), code) {
				t.Fatalf("deployed runtime/nonce mismatch: %x", db.GetCode(address))
			}
		})
	}
}
