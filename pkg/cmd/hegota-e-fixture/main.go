// hegota-e-fixture signs public-key-1 transactions for the isolated qualification
// chain and exports canonical PQ bytes solely for transport measurements.
package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"math/big"
	"os"

	"github.com/eth2030/eth2030/internal/testutil/proof8504"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

const fixtureChain = 3151908

func ordinary(nonce, value uint64) (map[string]any, error) {
	key, err := gethcrypto.HexToECDSA("0000000000000000000000000000000000000000000000000000000000000001")
	if err != nil {
		return nil, err
	}
	to := common.HexToAddress("0x0000000000000000000000000000000000000801")
	// EIP-8037 charges a fresh recipient's account creation at runtime. The
	// historical 21,000-gas fixture cannot cover that state cost.
	tx, err := gethtypes.SignTx(gethtypes.NewTx(&gethtypes.LegacyTx{Nonce: nonce, To: &to, Value: new(big.Int).SetUint64(value), Gas: 250000, GasPrice: big.NewInt(10000000000)}), gethtypes.LatestSignerForChainID(big.NewInt(fixtureChain)), key)
	if err != nil {
		return nil, err
	}
	raw, err := tx.MarshalBinary()
	return map[string]any{"raw": "0x" + hex.EncodeToString(raw), "transaction": tx.Hash().Hex(), "sender": gethcrypto.PubkeyToAddress(key.PublicKey).Hex(), "chain_id": fixtureChain}, err
}

func sample() (map[string]any, error) {
	tx, _ := proof8504.CanonicalTransaction(false)
	raw, err := tx.EncodeRLP()
	return map[string]any{"raw": "0x" + hex.EncodeToString(raw), "transaction_bytes": len(raw), "scope": "canonical direct-8504 transport bytes only; not live-chain transaction validity or omitted-witness composition"}, err
}

func main() {
	kind := flag.String("kind", "ordinary", "ordinary or pq-transport")
	nonce := flag.Uint64("nonce", 0, "public fixture account nonce")
	value := flag.Uint64("value", 1, "fixture transfer value in wei")
	flag.Parse()
	var result map[string]any
	var err error
	switch *kind {
	case "ordinary":
		result, err = ordinary(*nonce, *value)
	case "pq-transport":
		result, err = sample()
	default:
		err = fmt.Errorf("unknown fixture kind %q", *kind)
	}
	if err == nil {
		err = json.NewEncoder(os.Stdout).Encode(result)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
