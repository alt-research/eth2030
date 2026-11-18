// This generator encodes the independently calculated four-account post-state
// using go-ethereum StateAccount RLP and StackTrie. It does not execute ETH2030.
package main

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"
	"sort"
)

func main() {
	factory := common.Address{19: 0x90}
	sender := crypto.CreateAddress(factory, 0)
	accounts := map[common.Address]struct {
		nonce, balance uint64
		code           []byte
	}{
		common.HexToAddress("0x7e5f4552091a69125d5dfcb7b8c2659029395bdf"): {balance: 1000000000},
		factory:                      {nonce: 1, code: common.FromHex("0x365f5f37365f5ff05000")},
		sender:                       {nonce: 2, balance: 999892111, code: common.FromHex("0x60035f5faa")},
		common.HexToAddress("0xfee"): {balance: 71926},
	}
	type leaf struct{ key, value []byte }
	var leaves []leaf
	for address, account := range accounts {
		value, err := rlp.EncodeToBytes(&types.StateAccount{Nonce: account.nonce, Balance: uint256.NewInt(account.balance), Root: types.EmptyRootHash, CodeHash: crypto.Keccak256(account.code)})
		if err != nil {
			panic(err)
		}
		leaves = append(leaves, leaf{crypto.Keccak256(address[:]), value})
	}
	sort.Slice(leaves, func(i, j int) bool { return bytes.Compare(leaves[i].key, leaves[j].key) < 0 })
	tree := trie.NewStackTrie(nil)
	for _, leaf := range leaves {
		if err := tree.Update(leaf.key, leaf.value); err != nil {
			panic(err)
		}
	}
	fmt.Println(tree.Hash().Hex())
}
