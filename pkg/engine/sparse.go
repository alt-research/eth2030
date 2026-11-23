package engine

import (
	"encoding/json"
	"fmt"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine/payload"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func parseCellMask(raw []byte) (*types.CellMask, error) {
	if isNullWire(raw) {
		return nil, nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return nil, err
	}
	var mask types.CellMask
	if err := mask.UnmarshalText([]byte(text)); err != nil {
		return nil, err
	}
	return &mask, nil
}
func (api *EngineAPI) handleGetBlobsV4(params []json.RawMessage) (any, *jsonrpcError) {
	if len(params) != 2 {
		return nil, &jsonrpcError{Code: InvalidParamsCode, Message: "expected versioned hashes and 16-byte cell mask"}
	}
	var encoded []string
	if err := json.Unmarshal(params[0], &encoded); err != nil || isNullWire(params[0]) {
		return nil, &jsonrpcError{Code: InvalidParamsCode, Message: "invalid versioned hashes"}
	}
	if len(encoded) > 128 {
		return nil, &jsonrpcError{Code: TooLargeRequestCode, Message: "too many versioned hashes"}
	}
	hashes := make([]types.Hash, len(encoded))
	for i, text := range encoded {
		raw, err := hexutil.Decode(text)
		if err != nil || len(raw) != 32 || raw[0] != 1 {
			return nil, &jsonrpcError{Code: InvalidParamsCode, Message: fmt.Sprintf("invalid versioned hash %d", i)}
		}
		copy(hashes[i][:], raw)
	}
	mask, err := parseCellMask(params[1])
	if err != nil || mask == nil {
		return nil, &jsonrpcError{Code: InvalidParamsCode, Message: "invalid 16-byte cell mask"}
	}
	if !api.backend.IsOsaka(api.backend.GetHeadTimestamp()) {
		return nil, nil
	}
	if backend, ok := api.backend.(interface {
		GetBlobsV4ByVersionedHashes([]types.Hash, types.CellMask) []*payload.BlobCellsAndProofsV1
	}); ok {
		return backend.GetBlobsV4ByVersionedHashes(hashes, *mask), nil
	}
	return nil, nil
}
