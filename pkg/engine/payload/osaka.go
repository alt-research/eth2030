package payload

import (
	"encoding/json"
	"math/big"
)

// GetPayloadV5Response is the canonical Osaka Engine response. Amsterdam uses V6.
type GetPayloadV5Response struct {
	ExecutionPayload  *ExecutionPayloadV3
	BlockValue        *big.Int
	BlobsBundle       *BlobsBundleV2
	Override          bool
	ExecutionRequests [][]byte
}

type osakaResponseJSON struct {
	ExecutionPayload  *ExecutionPayloadV3 `json:"executionPayload"`
	BlockValue        *hexBig             `json:"blockValue"`
	BlobsBundle       *BlobsBundleV2      `json:"blobsBundle"`
	Override          bool                `json:"shouldOverrideBuilder"`
	ExecutionRequests []hexBytes          `json:"executionRequests"`
}

func (r GetPayloadV5Response) MarshalJSON() ([]byte, error) {
	bundle := r.BlobsBundle
	if bundle == nil {
		bundle = &BlobsBundleV2{}
	}
	return json.Marshal(osakaResponseJSON{
		ExecutionPayload:  r.ExecutionPayload,
		BlockValue:        newHexBig(r.BlockValue),
		BlobsBundle:       bundle,
		Override:          r.Override,
		ExecutionRequests: toHexBytesSlice(r.ExecutionRequests),
	})
}

func (r *GetPayloadV5Response) UnmarshalJSON(data []byte) error {
	var j osakaResponseJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	r.ExecutionPayload = j.ExecutionPayload
	r.BlockValue = toBigInt(j.BlockValue)
	r.BlobsBundle = j.BlobsBundle
	r.Override = j.Override
	r.ExecutionRequests = fromHexBytesSlice(j.ExecutionRequests)
	return nil
}
