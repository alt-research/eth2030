package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto/bls"
	"os"
)

func h(a, b []byte) [32]byte { return sha256.Sum256(append(append([]byte{}, a...), b...)) }
func main() {
	ikm := sha256.Sum256([]byte("ETH2030 EIP-8282 fixed public test vector 2026-11-18"))
	pub, secret, err := bls.BlstKeyGen(ikm[:])
	if err != nil {
		panic(err)
	}
	r := &types.BuilderDepositRequest{Amount: 1000000000}
	copy(r.Pubkey[:], pub)
	r.WithdrawalCredentials[0] = 0xb0
	address := types.HexToAddress("0x19e7e376e7c213b7e7e7e46cc70a5dd086daff2a")
	copy(r.WithdrawalCredentials[12:], address[:])
	var pubchunks [64]byte
	copy(pubchunks[:], pub)
	pubroot := sha256.Sum256(pubchunks[:])
	var amount [32]byte
	binary.LittleEndian.PutUint64(amount[:8], r.Amount)
	zero := [32]byte{}
	left := h(pubroot[:], r.WithdrawalCredentials[:])
	right := h(amount[:], zero[:])
	message := h(left[:], right[:])
	version := [32]byte{0, 0, 0, 1}
	fork := h(version[:], zero[:])
	domain := [32]byte{14}
	copy(domain[4:], fork[:28])
	signing := h(message[:], domain[:])
	sig, err := bls.BlstSign(secret, signing[:])
	if err != nil {
		panic(err)
	}
	copy(r.Signature[:], sig)
	ex := (&types.BuilderExitRequest{SourceAddress: address, Pubkey: r.Pubkey}).Encode()
	hexdata := func(v []byte) string { return "0x" + hex.EncodeToString(v) }
	wrongdomain := domain
	wrongdomain[0] = 3
	wrongroot := h(message[:], wrongdomain[:])
	badSig, err := bls.BlstSign(secret, wrongroot[:])
	if err != nil {
		panic(err)
	}
	wrong := *r
	copy(wrong.Signature[:], badSig)
	result := map[string]any{"description": "Synthetic real-BLS BuilderDepositRequest signed under DOMAIN_BUILDER_DEPOSIT; public IKM documented in generator", "eips_revision": "3db38a3725105cb67b8804ac3e147967113bc3ca", "consensus_revision": "2ea9d96a420927eb1a35d7c9399c8b3691c69f6a", "genesis_fork_version": "0x00000001", "execution_address": address.Hex(), "pubkey": hexdata(pub), "domain": hexdata(domain[:]), "signing_root": hexdata(signing[:]), "deposit_calldata": hexdata(r.EncodeCalldata()), "deposit_ssz": hexdata(r.Encode()), "validator_domain_deposit_ssz": hexdata(wrong.Encode()), "exit_ssz": hexdata(ex), "execution_requests": []string{hexdata(append([]byte{3}, r.Encode()...)), hexdata(append([]byte{4}, ex...))}}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err = encoder.Encode(result); err != nil {
		panic(err)
	}
	fmt.Fprint(os.Stderr, "fixture generated\n")
}
