// Package framepay provides explicit, versioned EIP-8141 paymaster artifacts.
// Local v1 and the pinned upstream PR-12041 proposal have distinct recognition,
// authorization and storage rules. Neither is enabled as a finalized standard.
package framepay

import (
	"bytes"
	_ "embed"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"

	"github.com/eth2030/eth2030/core/types"
)

const Version = "eth2030-frame-paymaster-v1"
const WithdrawalDelay uint64 = 12 * 60 * 60

//go:embed contracts/runtime.hex
var runtimeHex string

//go:embed contracts/init.hex
var initHex string

var runtimeCode = decodeArtifact(runtimeHex)
var initCode = decodeArtifact(initHex)

func decodeArtifact(value string) []byte {
	code, err := hex.DecodeString(strings.TrimSpace(value))
	if err != nil {
		panic("invalid embedded paymaster artifact: " + err.Error())
	}
	return code
}

// RuntimeV1 returns a copy; callers cannot change the recognized code template.
func RuntimeV1() []byte { return bytes.Clone(runtimeCode) }

// IsV1 grants recognition only to the complete, exact compiled runtime.
func IsV1(code []byte) bool { return bytes.Equal(code, runtimeCode) }

func InitV1(owner types.Address) ([]byte, error) {
	if owner.IsZero() {
		return nil, errors.New("paymaster owner cannot be zero")
	}
	argument := make([]byte, 32)
	copy(argument[12:], owner[:])
	return append(bytes.Clone(initCode), argument...), nil
}

// ValidationData selects validation; witness bytes remain in signatures[1].
func ValidationData() []byte        { return []byte{0x63, 0x4b, 0x01, 0xe8} }
func ExecuteWithdrawalData() []byte { return []byte{0xfa, 0x45, 0xaa, 0x00} }

func RequestWithdrawalData(to types.Address, amount *big.Int) ([]byte, error) {
	if to.IsZero() || amount == nil || amount.Sign() < 0 || amount.BitLen() > 256 {
		return nil, errors.New("invalid paymaster withdrawal")
	}
	data := make([]byte, 68)
	copy(data, []byte{0xda, 0x95, 0xeb, 0xf7})
	copy(data[16:36], to[:])
	amount.FillBytes(data[36:])
	return data, nil
}

type StorageReader interface {
	GetState(types.Address, types.Hash) types.Hash
}

// PendingWithdrawal reserves the complete request both before and after maturity
// until execution clears it. The v1 immutable layout matches the EIP reference.
func PendingWithdrawal(db StorageReader, address types.Address) *big.Int {
	if db.GetState(address, types.Hash{31: 3}).IsZero() {
		return new(big.Int)
	}
	amount := db.GetState(address, types.Hash{31: 2})
	return new(big.Int).SetBytes(amount[:])
}
