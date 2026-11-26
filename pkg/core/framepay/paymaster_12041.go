package framepay

import (
	"bytes"
	_ "embed"
	"errors"
	"math/big"

	"github.com/eth2030/eth2030/core/types"
)

// PR12041Revision pins the proposed implementation, not a mainnet activation.
const PR12041Revision = "4a98fcefa5c6f02a25289bf55b9d92902d02b4c6"
const PR12041CodeHash = "0xda42f0d11838c4c0c3129b8b8e93e9718127ad6b315e517e1088125707c4d45c"
const WithdrawalDelay12041 uint64 = 86400
const MaxPayExecution12041 uint64 = 15000

//go:embed contracts/pr12041.hex
var runtime12041Hex string

var runtime12041 = decodeArtifact(runtime12041Hex)

func Runtime12041() []byte     { return bytes.Clone(runtime12041) }
func Is12041(code []byte) bool { return bytes.Equal(code, runtime12041) }

// Init12041 uses the proposal's constructor-writes-storage pattern. A zero
// signer is deliberately deployable but inert, as specified in the proposal.
// The initialization program is local; recognition covers only runtime bytes.
func Init12041(signer types.Address) []byte {
	code := append([]byte{0x73}, signer[:]...) // PUSH20 signer
	code = append(code, 0x5f, 0x55, 0x61, 0x01, 0x63, 0x61, 0x00, 0x24, 0x5f, 0x39, 0x61, 0x01, 0x63, 0x5f, 0xf3)
	return append(code, Runtime12041()...)
}

func RequestWithdrawal12041(amount *big.Int) ([]byte, error) {
	if amount == nil || amount.Sign() <= 0 || amount.BitLen() > 256 {
		return nil, errors.New("paymaster withdrawal amount must be a nonzero uint256")
	}
	data := make([]byte, 33)
	data[0] = 1
	amount.FillBytes(data[1:])
	return data, nil
}

func RequestRotation12041(signer types.Address) ([]byte, error) {
	if signer.IsZero() {
		return nil, errors.New("paymaster new signer cannot be zero")
	}
	data := make([]byte, 33)
	data[0] = 2
	copy(data[13:], signer[:])
	return data, nil
}

func Cancel12041() []byte   { return []byte{3} }
func Finalize12041() []byte { return []byte{4} }

// Profile is local public-pool recognition policy. It never changes consensus
// execution, deploys a contract at activation, or recognizes a delegation stub.
type Profile uint8

const (
	NoPaymaster Profile = iota
	LocalV1
	PR12041
)

func SelectProfile(localV1, proposal12041 bool) (Profile, error) {
	if localV1 && proposal12041 {
		return NoPaymaster, errors.New("select exactly one experimental Frame paymaster profile")
	}
	if localV1 {
		return LocalV1, nil
	}
	if proposal12041 {
		return PR12041, nil
	}
	return NoPaymaster, nil
}

func (p Profile) Valid() bool { return p <= PR12041 }
func (p Profile) Matches(code []byte) bool {
	switch p {
	case LocalV1:
		return IsV1(code)
	case PR12041:
		return Is12041(code)
	default:
		return false
	}
}

func (p Profile) PendingWithdrawal(db StorageReader, payer types.Address) *big.Int {
	if p == LocalV1 {
		return PendingWithdrawal(db, payer)
	}
	if p == PR12041 {
		amount := db.GetState(payer, types.Hash{31: 1})
		return new(big.Int).SetBytes(amount[:])
	}
	return new(big.Int)
}

// ValidatePayUsage applies the proposal's empty-data interface and its optional
// local 15,000 execution-gas bound, without imposing either on consensus.
func (p Profile) ValidatePayUsage(frame types.Frame8141) error {
	if p == PR12041 {
		if len(frame.Data) != 0 {
			return errors.New("PR-12041 PAY requires empty data")
		}
		if frame.Limits.Execution > MaxPayExecution12041 {
			return errors.New("PR-12041 PAY exceeds local execution limit")
		}
	}
	return nil
}
