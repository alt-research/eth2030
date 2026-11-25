package types

import (
	"encoding/binary"
	"encoding/hex"
	"errors"

	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

const (
	RecentRootLength        = 8192
	RecentRootUsableWindow  = RecentRootLength - 1
	MaxRecentRootReferences = 16
	RecentRootTupleBytes    = 72
)

var RecentRootAddress = HexToAddress("0x8272D9679689Ea2f307140CdF9002D27dC00Ffff")

// RecentRootCode is the verbatim EIP-8272 runtime at the locked EIP revision.
// It is an ordinary deployed contract; fork activation must not install it.
var RecentRootCode = func() []byte {
	code, err := hex.DecodeString("346100ba57366040146100c05736604836066100ba5780156100ba5761048081116100ba574b60005b602081013560c01c828110156100ba5780830361200011156100ba577f8f42481679c8e6fefa040974b3c905e0ce3f2e464ba93acdb074a41181617efc60005260488260203760686000207fbdc897da2177d260ff5f4be5d4b2aad43f89c3347a305b584fa5a2546d053daa60005290611fff1660c01b60405260486000205414156100ba5760480182811061002857005b60006000fd5b33600052602060006020376034600c20807f8f42481679c8e6fefa040974b3c905e0ce3f2e464ba93acdb074a41181617efc6040524b606852606052602060206088376068604020817fbdc897da2177d260ff5f4be5d4b2aad43f89c3347a305b584fa5a2546d053daa60a852611fff4b1660d05260c852604860a8205500")
	if err != nil {
		panic(err)
	}
	return code
}()

var recentRootEntryDomain = gethcrypto.Keccak256([]byte("RECENT_ROOT_ENTRY"))
var recentRootStorageDomain = gethcrypto.Keccak256([]byte("RECENT_ROOT_STORAGE"))

type RecentRootReference struct {
	Source Hash
	Slot   uint64
	Root   Hash
}

func RecentRootSource(writer Address, salt Hash) Hash {
	return Hash(gethcrypto.Keccak256Hash(writer[:], salt[:]))
}

func (ref RecentRootReference) EntryHash() Hash {
	var slot [8]byte
	binary.BigEndian.PutUint64(slot[:], ref.Slot)
	return Hash(gethcrypto.Keccak256Hash(recentRootEntryDomain, ref.Source[:], slot[:], ref.Root[:]))
}

func (ref RecentRootReference) StorageKey() Hash {
	var index [8]byte
	binary.BigEndian.PutUint64(index[:], ref.Slot%RecentRootLength)
	return Hash(gethcrypto.Keccak256Hash(recentRootStorageDomain, ref.Source[:], index[:]))
}

func (ref RecentRootReference) IsRecent(currentSlot uint64) bool {
	return ref.Slot < currentSlot && currentSlot-ref.Slot <= RecentRootUsableWindow
}

// DecodeRecentRootReferences parses the selector-free, packed calldata. This is
// frame classification, not a substitute for executing the deployed contract.
func DecodeRecentRootReferences(data []byte) ([]RecentRootReference, error) {
	if len(data) == 0 || len(data)%RecentRootTupleBytes != 0 || len(data) > MaxRecentRootReferences*RecentRootTupleBytes {
		return nil, errors.New("EIP-8272 invalid recent root calldata length")
	}
	refs := make([]RecentRootReference, len(data)/RecentRootTupleBytes)
	for i := range refs {
		tuple := data[i*RecentRootTupleBytes:]
		copy(refs[i].Source[:], tuple[:32])
		refs[i].Slot = binary.BigEndian.Uint64(tuple[32:40])
		copy(refs[i].Root[:], tuple[40:72])
	}
	return refs, nil
}

func (ref RecentRootReference) AppendTo(data []byte) []byte {
	var tuple [RecentRootTupleBytes]byte
	copy(tuple[:32], ref.Source[:])
	binary.BigEndian.PutUint64(tuple[32:40], ref.Slot)
	copy(tuple[40:], ref.Root[:])
	return append(data, tuple[:]...)
}

func (frame Frame8141) IsRecentRootVerifier() bool {
	_, err := DecodeRecentRootReferences(frame.Data)
	return frame.Mode == ModeVerify && frame.Target != nil && *frame.Target == RecentRootAddress &&
		frame.Flags == 0 && bigOrZero(frame.Value).Sign() == 0 && frame.Limits.State == 0 && err == nil
}
