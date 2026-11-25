// Command glamsterdam-ledger independently verifies captured execution payments.
// It reads artifacts only and imports no native execution/state implementation.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

type capture struct {
	Hashes    []common.Hash    `json:"execution_hashes"`
	Addresses []common.Address `json:"addresses"`
	RPC       []struct {
		Port   int               `json:"endpoint_port"`
		Method string            `json:"method"`
		Params []json.RawMessage `json:"params"`
		Reply  struct {
			Result json.RawMessage `json:"result"`
			Error  json.RawMessage `json:"error"`
		} `json:"response"`
	} `json:"rpc"`
}
type blockCapture struct {
	Hash         common.Hash         `json:"hash"`
	Transactions []json.RawMessage   `json:"transactions"`
	Withdrawals  []*types.Withdrawal `json:"withdrawals"`
	Header       *types.Header       `json:"-"`
}
type accountProof struct {
	Address      common.Address  `json:"address"`
	Balance      *hexutil.Big    `json:"balance"`
	Nonce        hexutil.Uint64  `json:"nonce"`
	CodeHash     common.Hash     `json:"codeHash"`
	StorageHash  common.Hash     `json:"storageHash"`
	AccountProof []hexutil.Bytes `json:"accountProof"`
}
type observation struct {
	Port    int               `json:"endpoint_port"`
	Before  map[string]string `json:"before_wei"`
	After   map[string]string `json:"after_wei"`
	PaidWei string            `json:"paid_wei"`
}

func verifyAccount(root common.Hash, address common.Address, proof accountProof) (*big.Int, error) {
	if proof.Address != address || proof.Balance == nil {
		return nil, fmt.Errorf("proof does not bind the requested account")
	}
	db := memorydb.New()
	for _, node := range proof.AccountProof {
		if err := db.Put(crypto.Keccak256(node), node); err != nil {
			return nil, err
		}
	}
	encoded, err := trie.VerifyProof(root, crypto.Keccak256(address[:]), db)
	if err != nil {
		return nil, fmt.Errorf("independent account proof: %w", err)
	}
	var account types.StateAccount
	if err := rlp.DecodeBytes(encoded, &account); err != nil {
		return nil, err
	}
	balance := account.Balance.ToBig()
	if balance.Cmp((*big.Int)(proof.Balance)) != 0 || account.Nonce != uint64(proof.Nonce) || account.Root != proof.StorageHash || common.BytesToHash(account.CodeHash) != proof.CodeHash {
		return nil, fmt.Errorf("account proof contradicts claimed fields")
	}
	return balance, nil
}

func verifyPressureProofs(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	scan := bufio.NewScanner(file)
	scan.Buffer(make([]byte, 64<<10), 16<<20)
	count := 0
	for scan.Scan() {
		// The stream also contains headers (whose nonce is an 8-byte value),
		// EVM results and disconnected transports. Decode proof fields only
		// after selecting an actual proof record, keeping strict quantity
		// validation for its account nonce unchanged.
		var kind struct{ Kind string }
		if err := json.Unmarshal(scan.Bytes(), &kind); err != nil {
			return count, err
		}
		if kind.Kind != "proof" && kind.Kind != "historical-proof" {
			continue
		}
		var record struct {
			Kind    string          `json:"kind"`
			Hash    common.Hash     `json:"expected_hash"`
			Header  json.RawMessage `json:"header"`
			Request struct {
				Method string
				Params []json.RawMessage
			} `json:"request"`
			Response struct {
				Result accountProof
				Error  json.RawMessage
			} `json:"response"`
		}
		if err := json.Unmarshal(scan.Bytes(), &record); err != nil {
			return count, err
		}
		if record.Request.Method != "eth_getProof" || len(record.Request.Params) != 3 || (len(record.Response.Error) > 0 && string(record.Response.Error) != "null") {
			return count, fmt.Errorf("invalid captured proof RPC")
		}
		var address common.Address
		var selector struct {
			Hash      common.Hash `json:"blockHash"`
			Canonical bool        `json:"requireCanonical"`
		}
		var header types.Header
		if err := json.Unmarshal(record.Request.Params[0], &address); err != nil {
			return count, err
		}
		if err := json.Unmarshal(record.Request.Params[2], &selector); err != nil {
			return count, err
		}
		if err := json.Unmarshal(record.Header, &header); err != nil {
			return count, err
		}
		if selector.Canonical != (record.Kind == "proof") || selector.Hash != record.Hash || header.Hash() != record.Hash {
			return count, fmt.Errorf("proof is not bound to its requested header and canonicality")
		}
		if _, err := verifyAccount(header.Root, address, record.Response.Result); err != nil {
			return count, err
		}
		count++
	}
	if err := scan.Err(); err != nil {
		return count, err
	}
	if count == 0 {
		return 0, fmt.Errorf("no actual proof controls captured")
	}
	return count, nil
}

func verify(input capture) ([]observation, error) {
	if len(input.Hashes) != 2 || input.Hashes[0] == input.Hashes[1] || len(input.Addresses) == 0 {
		return nil, fmt.Errorf("two distinct original execution hashes and payment recipients are required")
	}
	blocks := make(map[int]map[common.Hash]blockCapture)
	proofs := make(map[int]map[common.Hash]map[common.Address]accountProof)
	for _, record := range input.RPC {
		if len(record.Reply.Error) > 0 && string(record.Reply.Error) != "null" {
			return nil, fmt.Errorf("captured RPC error at endpoint %d", record.Port)
		}
		switch record.Method {
		case "eth_getBlockByHash":
			var block blockCapture
			if err := json.Unmarshal(record.Reply.Result, &block); err != nil {
				return nil, err
			}
			if block.Hash != input.Hashes[0] && block.Hash != input.Hashes[1] {
				continue
			}
			var requested common.Hash
			if len(record.Params) != 2 {
				return nil, fmt.Errorf("invalid captured block parameters")
			}
			if err := json.Unmarshal(record.Params[0], &requested); err != nil || requested != block.Hash {
				return nil, fmt.Errorf("captured block request mismatch")
			}
			block.Header = new(types.Header)
			if err := json.Unmarshal(record.Reply.Result, block.Header); err != nil {
				return nil, err
			}
			if block.Header.Hash() != block.Hash {
				return nil, fmt.Errorf("independent complete header hash mismatch")
			}
			if block.Header.WithdrawalsHash == nil || types.DeriveSha(types.Withdrawals(block.Withdrawals), trie.NewStackTrie(nil)) != *block.Header.WithdrawalsHash {
				return nil, fmt.Errorf("withdrawals do not match original header commitment")
			}
			if blocks[record.Port] == nil {
				blocks[record.Port] = make(map[common.Hash]blockCapture)
			}
			blocks[record.Port][block.Hash] = block
		case "eth_getProof":
			if len(record.Params) != 3 {
				return nil, fmt.Errorf("invalid captured proof parameters")
			}
			var selector struct {
				Hash      common.Hash `json:"blockHash"`
				Canonical bool        `json:"requireCanonical"`
			}
			var requested common.Address
			if err := json.Unmarshal(record.Params[0], &requested); err != nil {
				return nil, err
			}
			if err := json.Unmarshal(record.Params[2], &selector); err != nil {
				return nil, err
			}
			if !selector.Canonical || (selector.Hash != input.Hashes[0] && selector.Hash != input.Hashes[1]) {
				return nil, fmt.Errorf("proof did not use original canonical hash")
			}
			var proof accountProof
			if err := json.Unmarshal(record.Reply.Result, &proof); err != nil {
				return nil, err
			}
			if proof.Address != requested {
				return nil, fmt.Errorf("proof address differs from request")
			}
			if proofs[record.Port] == nil {
				proofs[record.Port] = make(map[common.Hash]map[common.Address]accountProof)
			}
			if proofs[record.Port][selector.Hash] == nil {
				proofs[record.Port][selector.Hash] = make(map[common.Address]accountProof)
			}
			proofs[record.Port][selector.Hash][proof.Address] = proof
		}
	}
	var results []observation
	if len(blocks) != 3 {
		return nil, fmt.Errorf("three captured execution endpoints are required")
	}
	ports := make([]int, 0, len(blocks))
	for port := range blocks {
		ports = append(ports, port)
	}
	sort.Ints(ports)
	for _, port := range ports {
		before, beforeOK := blocks[port][input.Hashes[0]]
		after, afterOK := blocks[port][input.Hashes[1]]
		if !beforeOK || !afterOK || after.Header.ParentHash != before.Hash || after.Header.Number.Uint64() != before.Header.Number.Uint64()+1 {
			return nil, fmt.Errorf("endpoint %d lacks the original consecutive execution pair", port)
		}
		if len(after.Transactions) != 0 || after.Header.TxHash != types.EmptyTxsHash {
			return nil, fmt.Errorf("selected credit control must have an empty transaction payload")
		}
		paid := make(map[common.Address]*big.Int)
		total := new(big.Int)
		for i, withdrawal := range after.Withdrawals {
			if withdrawal.Validator&uint64(1<<40) == 0 || withdrawal.Amount == 0 {
				return nil, fmt.Errorf("selected control includes non-builder or zero withdrawal")
			}
			if i > 0 && withdrawal.Index <= after.Withdrawals[i-1].Index {
				return nil, fmt.Errorf("duplicate or unordered withdrawal index")
			}
			if len(before.Withdrawals) > 0 && withdrawal.Index <= before.Withdrawals[len(before.Withdrawals)-1].Index {
				return nil, fmt.Errorf("withdrawal paid again across execution blocks")
			}
			if paid[withdrawal.Address] == nil {
				paid[withdrawal.Address] = new(big.Int)
			}
			amount := new(big.Int).Mul(new(big.Int).SetUint64(withdrawal.Amount), big.NewInt(1_000_000_000))
			paid[withdrawal.Address].Add(paid[withdrawal.Address], amount)
			total.Add(total, amount)
		}
		if total.Sign() == 0 || len(paid) != len(input.Addresses) {
			return nil, fmt.Errorf("captured recipient/payment set incomplete")
		}
		result := observation{Port: port, Before: make(map[string]string), After: make(map[string]string), PaidWei: total.String()}
		for _, address := range input.Addresses {
			var balances []*big.Int
			for _, block := range []blockCapture{before, after} {
				proof, ok := proofs[port][block.Hash][address]
				if !ok || proof.Balance == nil {
					return nil, fmt.Errorf("missing original account proof")
				}
				balance, err := verifyAccount(block.Header.Root, address, proof)
				if err != nil {
					return nil, err
				}
				balances = append(balances, balance)
			}
			if paid[address] == nil || new(big.Int).Sub(balances[1], balances[0]).Cmp(paid[address]) != 0 {
				return nil, fmt.Errorf("actual execution credit differs from exactly one committed payment")
			}
			result.Before[address.Hex()], result.After[address.Hex()] = balances[0].String(), balances[1].String()
		}
		results = append(results, result)
	}
	return results, nil
}

func main() {
	if len(os.Args) == 3 && os.Args[1] == "proof-stream" {
		count, err := verifyPressureProofs(os.Args[2])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"passed": true, "independent_proofs": count})
		return
	}
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: glamsterdam-ledger captured-ledger.json")
		os.Exit(2)
	}
	raw, err := os.ReadFile(os.Args[1])
	var input capture
	if err == nil {
		err = json.Unmarshal(raw, &input)
	}
	var results []observation
	if err == nil {
		results, err = verify(input)
	}
	if err == nil {
		err = json.NewEncoder(os.Stdout).Encode(map[string]any{"passed": true, "execution_hashes": input.Hashes, "observations": results})
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
