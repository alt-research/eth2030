package types

import (
	"encoding/binary"
	"fmt"
)

const (
	BuilderDepositRequestType           byte = 3
	BuilderExitRequestType              byte = 4
	BuilderDepositRequestSSZSize             = 184
	BuilderExitRequestSSZSize                = 68
	MaxBuilderDepositRequestsPerPayload      = 64
	MaxBuilderExitRequestsPerPayload         = 16
)

var BuilderDepositRequestAddress = HexToAddress("0x0000bFF46984e3725691FA540a8C7589300D8282")
var BuilderExitRequestAddress = HexToAddress("0x000064D678505ad48F8cCb093BC65613800E8282")

type BuilderDepositRequest struct {
	Pubkey                [48]byte
	WithdrawalCredentials [32]byte
	Amount                uint64
	Signature             [96]byte
}

func (r *BuilderDepositRequest) Encode() []byte {
	data := make([]byte, BuilderDepositRequestSSZSize)
	copy(data, r.Pubkey[:])
	copy(data[48:], r.WithdrawalCredentials[:])
	binary.LittleEndian.PutUint64(data[80:88], r.Amount)
	copy(data[88:], r.Signature[:])
	return data
}

// EncodeCalldata differs from SSZ only in the big-endian deposit amount.
func (r *BuilderDepositRequest) EncodeCalldata() []byte {
	data := r.Encode()
	binary.BigEndian.PutUint64(data[80:88], r.Amount)
	return data
}
func DecodeBuilderDepositRequest(data []byte) (*BuilderDepositRequest, error) {
	if len(data) != BuilderDepositRequestSSZSize {
		return nil, fmt.Errorf("invalid builder deposit record size %d", len(data))
	}
	value := new(BuilderDepositRequest)
	copy(value.Pubkey[:], data)
	copy(value.WithdrawalCredentials[:], data[48:80])
	value.Amount = binary.LittleEndian.Uint64(data[80:88])
	copy(value.Signature[:], data[88:])
	return value, nil
}

type BuilderExitRequest struct {
	SourceAddress Address
	Pubkey        [48]byte
}

func (r *BuilderExitRequest) Encode() []byte {
	data := make([]byte, BuilderExitRequestSSZSize)
	copy(data, r.SourceAddress[:])
	copy(data[20:], r.Pubkey[:])
	return data
}
func DecodeBuilderExitRequest(data []byte) (*BuilderExitRequest, error) {
	if len(data) != BuilderExitRequestSSZSize {
		return nil, fmt.Errorf("invalid builder exit record size %d", len(data))
	}
	value := new(BuilderExitRequest)
	copy(value.SourceAddress[:], data[:20])
	copy(value.Pubkey[:], data[20:])
	return value, nil
}
