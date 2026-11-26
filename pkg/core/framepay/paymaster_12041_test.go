package framepay

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// Assemble the upstream annotated listing independently of the embedded bytes.
func TestPR12041ReproducesUpstreamAssembly(t *testing.T) {
	source, err := os.ReadFile("contracts/pr12041.asm")
	if err != nil {
		t.Fatal(err)
	}
	ops := map[string]byte{"callvalue": 0x34, "calldatasize": 0x36, "jumpi": 0x57, "iszero": 0x15, "sigparam": 0xb4, "push0": 0x5f, "sload": 0x54, "eq": 0x14, "approve": 0xaa, "jumpdest": 0x5b, "stop": 0, "calldataload": 0x35, "shr": 0x1c, "dup1": 0x80, "revert": 0xfd, "pop": 0x50, "caller": 0x33, "sstore": 0x55, "timestamp": 0x42, "add": 1, "lt": 0x10, "dup5": 0x84, "gas": 0x5a, "call": 0xf1}
	var lines [][]string
	labels := make(map[string]int)
	position := 0
	for _, line := range strings.Split(string(source), "\n") {
		fields := strings.Fields(strings.SplitN(line, ";", 2)[0])
		if len(fields) == 0 {
			continue
		}
		if strings.HasSuffix(fields[0], ":") {
			labels[strings.TrimSuffix(fields[0], ":")] = position
			continue
		}
		lines = append(lines, fields)
		position++
		if strings.HasPrefix(fields[0], "push") && fields[0] != "push0" {
			n, e := strconv.Atoi(fields[0][4:])
			if e != nil {
				t.Fatal(e)
			}
			position += n
		}
	}
	var assembled []byte
	for _, fields := range lines {
		if op, ok := ops[fields[0]]; ok {
			assembled = append(assembled, op)
			continue
		}
		if len(fields) != 2 || !strings.HasPrefix(fields[0], "push") {
			t.Fatalf("unknown instruction %v", fields)
		}
		n, err := strconv.Atoi(fields[0][4:])
		if err != nil || n < 1 || n > 32 {
			t.Fatal("invalid push width")
		}
		var value []byte
		if strings.HasPrefix(fields[1], "0x") {
			value, err = hex.DecodeString(fields[1][2:])
			if err != nil {
				t.Fatal(err)
			}
		} else {
			offset, ok := labels[fields[1]]
			if !ok {
				t.Fatal("unknown label")
			}
			value = big.NewInt(int64(offset)).Bytes()
		}
		if len(value) > n {
			t.Fatal("push truncates value")
		}
		assembled = append(assembled, byte(0x5f+n))
		assembled = append(assembled, make([]byte, n-len(value))...)
		assembled = append(assembled, value...)
	}
	const expected = "0xda42f0d11838c4c0c3129b8b8e93e9718127ad6b315e517e1088125707c4d45c"
	if len(assembled) != 355 || !bytes.Equal(assembled, Runtime12041()) || gethcrypto.Keccak256Hash(assembled).Hex() != expected || PR12041CodeHash != expected {
		t.Fatal("upstream listing, frozen runtime or code hash disagrees")
	}
	mutated := Runtime12041()
	mutated[0] ^= 1
	if Is12041(mutated) || Is12041(append(Runtime12041(), 0)) || Is12041(RuntimeV1()) || !Is12041(Runtime12041()) {
		t.Fatal("inexact recognition or mutable artifact")
	}
}

func TestPR12041LayoutAndProfileIsolation(t *testing.T) {
	db := state.NewMemoryStateDB()
	payer := types.Address{19: 1}
	db.SetState(payer, types.Hash{31: 1}, types.Hash{31: 9})
	for _, clock := range []byte{0, 1, 255} {
		db.SetState(payer, types.Hash{31: 2}, types.Hash{31: clock})
		if PR12041.PendingWithdrawal(db, payer).Int64() != 9 {
			t.Fatal("proposal must reserve raw slot 1 until cleared")
		}
	}
	db.SetState(payer, types.Hash{31: 2}, types.Hash{31: 42})
	db.SetState(payer, types.Hash{31: 3}, types.Hash{31: 1})
	if LocalV1.PendingWithdrawal(db, payer).Int64() != 42 {
		t.Fatal("old layout was confused with proposal layout")
	}
	if _, err := SelectProfile(true, true); err == nil {
		t.Fatal("ambiguous version selection")
	}
	for _, p := range []Profile{NoPaymaster, LocalV1} {
		if p.Matches(Runtime12041()) {
			t.Fatal("proposal automatically recognized")
		}
	}
	if !PR12041.Matches(Runtime12041()) || PR12041.Matches(RuntimeV1()) || Profile(255).Valid() {
		t.Fatal("profile mismatch")
	}
	for _, amount := range []*big.Int{nil, new(big.Int), big.NewInt(-1), new(big.Int).Lsh(big.NewInt(1), 256)} {
		if _, err := RequestWithdrawal12041(amount); err == nil {
			t.Fatal("invalid withdrawal helper")
		}
	}
	amount := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	data, err := RequestWithdrawal12041(amount)
	if err != nil || len(data) != 33 || data[0] != 1 || new(big.Int).SetBytes(data[1:]).Cmp(amount) != 0 {
		t.Fatal("uint256 withdrawal ABI")
	}
	signer := types.Address{19: 7}
	data, err = RequestRotation12041(signer)
	if err != nil || len(data) != 33 || data[0] != 2 || !bytes.Equal(data[13:], signer[:]) {
		t.Fatal("rotation ABI")
	}
	if _, err := RequestRotation12041(types.Address{}); err == nil {
		t.Fatal("zero rotation helper")
	}
	if !bytes.Equal(Cancel12041(), []byte{3}) || !bytes.Equal(Finalize12041(), []byte{4}) {
		t.Fatal("admin dispatch")
	}
}
