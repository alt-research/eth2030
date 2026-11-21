package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// Verbatim EIP-7997 at EIPs 3db38a3725105cb67b8804ac3e147967113bc3ca.
// No native execution, state or system-contract helper supplies these expectations.
var canonicalFactory = common.HexToAddress("0x4e59b44847b379578588920cA78FbF26c0B4956C")

const canonicalFactoryCode = "0x7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe03601600081602082378035828234f58015156039578182fd5b8082525050506014600cf3"

func factoryGenesis(ctx context.Context, clients []client) error {
	for _, c := range clients {
		code, err := c.call(ctx, "eth_getCode", canonicalFactory, "0x0")
		if err != nil {
			return err
		}
		var value string
		if err := json.Unmarshal(code, &value); err != nil || value != canonicalFactoryCode {
			return fmt.Errorf("%s lacks the exact selected factory genesis runtime", c.url)
		}
		nonce, err := c.call(ctx, "eth_getTransactionCount", canonicalFactory, "0x0")
		if err != nil {
			return err
		}
		var count hexutil.Uint64
		if err := json.Unmarshal(nonce, &count); err != nil || count != 1 {
			return fmt.Errorf("%s factory genesis nonce is not one", c.url)
		}
	}
	return nil
}

func runFactory(ctx context.Context, urls []string, genesis common.Hash, writer io.Writer) error {
	clients, err := campaignClients(ctx, urls, genesis, writer)
	if err != nil {
		return err
	}
	if err := factoryGenesis(ctx, clients); err != nil {
		return err
	}
	key, err := crypto.HexToECDSA(fmt.Sprintf("%064x", 1))
	if err != nil {
		return err
	}
	owner := crypto.PubkeyToAddress(key.PublicKey)
	var nonce hexutil.Uint64
	for index, c := range clients {
		raw, err := c.call(ctx, "eth_getTransactionCount", owner, "pending")
		if err != nil {
			return err
		}
		var value hexutil.Uint64
		if err := json.Unmarshal(raw, &value); err != nil {
			return err
		}
		if index > 0 && value != nonce {
			return fmt.Errorf("factory sender nonce has not converged")
		}
		nonce = value
	}
	// The deterministic child returns uint256(42); all seven wei must reach it.
	initcode := common.FromHex("0x600a600c600039600a6000f3602a60005260206000f3")
	runtime := "0x602a60005260206000f3"
	var salt [32]byte
	salt[31] = 0x79
	child := crypto.CreateAddress2(canonicalFactory, salt, crypto.Keccak256(initcode))
	input := append(append([]byte(nil), salt[:]...), initcode...)
	for _, c := range clients {
		raw, err := c.call(ctx, "eth_call", map[string]any{"from": owner, "to": canonicalFactory,
			"data": hexutil.Encode(input), "gas": "0x7a120", "value": "0x7"}, "latest")
		if err != nil {
			return err
		}
		var result hexutil.Bytes
		if err := json.Unmarshal(raw, &result); err != nil || !bytes.Equal(result, child[:]) {
			return fmt.Errorf("factory did not return the exact unpadded CREATE2 address")
		}
		code, err := c.call(ctx, "eth_getCode", child, "latest")
		if err != nil || string(code) != `"0x"` {
			return fmt.Errorf("simulation published child state or salt was previously used: %v", err)
		}
	}
	var last common.Hash
	var transactions []common.Hash
	for index, scenario := range []struct {
		input  []byte
		value  int64
		status uint64
	}{{input, 7, 1}, {input, 11, 0}, {[]byte{1}, 13, 0}} {
		tx, err := types.SignTx(types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(1337),
			Nonce: uint64(nonce) + uint64(index), Gas: 500000, GasFeeCap: big.NewInt(8000000000),
			GasTipCap: big.NewInt(1000000000), To: &canonicalFactory, Value: big.NewInt(scenario.value), Data: scenario.input}),
			types.LatestSignerForChainID(big.NewInt(1337)), key)
		if err != nil {
			return err
		}
		receipt, err := submit(ctx, clients, tx, scenario.status)
		if err != nil {
			return fmt.Errorf("factory scenario %d: %w", index, err)
		}
		last = receipt.BlockHash
		transactions = append(transactions, tx.Hash())
		selector := map[string]any{"blockHash": last, "requireCanonical": true}
		for _, c := range clients {
			for _, check := range []struct {
				method   string
				address  common.Address
				expected string
			}{{"eth_getCode", child, runtime}, {"eth_getBalance", child, "0x7"},
				{"eth_getBalance", canonicalFactory, "0x0"}, {"eth_getCode", canonicalFactory, canonicalFactoryCode},
				{"eth_getTransactionCount", canonicalFactory, "0x2"}, {"eth_getTransactionCount", owner, hexutil.EncodeUint64(uint64(nonce) + uint64(index) + 1)}} {
				raw, err := c.call(ctx, check.method, check.address, selector)
				if err != nil {
					return err
				}
				var actual string
				if err := json.Unmarshal(raw, &actual); err != nil || actual != check.expected {
					return fmt.Errorf("factory scenario %d %s/%s: got %s, expected %s", index, check.method, check.address, raw, check.expected)
				}
			}
		}
		if err := query(ctx, clients, step{Name: "factory child returns 42", To: child.Hex(), Input: "0x", Expected: "0x" + fmt.Sprintf("%064x", 42)}, last); err != nil {
			return err
		}
	}
	for _, c := range clients {
		for _, address := range []common.Address{canonicalFactory, child} {
			if _, err := c.call(ctx, "eth_getProof", address, []string{}, map[string]any{"blockHash": last, "requireCanonical": true}); err != nil {
				return err
			}
		}
	}
	return json.NewEncoder(writer).Encode(map[string]any{"kind": "completed_factory_campaign", "spec": "3db38a3725105cb67b8804ac3e147967113bc3ca", "factory": canonicalFactory, "child": child, "transactions": transactions, "statuses": []uint64{1, 0, 0}, "last_block": last, "deployment_route": "exact explicit genesis allocation; no fork injection", "independent_mpt_proofs_evaluated": false})
}
