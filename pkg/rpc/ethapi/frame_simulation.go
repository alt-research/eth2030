package ethapi

import (
	"encoding/json"
	"errors"
	"github.com/eth2030/eth2030/core/types"
	rpctypes "github.com/eth2030/eth2030/rpc/types"
)

func (api *EthAPI) simulateFrame(req *Request) *Response {
	if len(req.Params) < 1 || len(req.Params) > 2 {
		return errorResponse(req.ID, ErrCodeInvalidParams, "expected Frame options and optional block selector")
	}
	var options rpctypes.FrameSimulationOptions
	if err := json.Unmarshal(req.Params[0], &options); err != nil {
		return errorResponse(req.ID, ErrCodeInvalidParams, err.Error())
	}
	var selector json.RawMessage
	if len(req.Params) > 1 {
		selector = req.Params[1]
	}
	header, selectErr := resolveRawBlockHeader(api.backend, selector)
	if selectErr != nil {
		return &Response{JSONRPC: "2.0", ID: req.ID, Error: selectErr}
	}
	backend, ok := api.backend.(interface {
		SimulateFrame(rpctypes.FrameSimulationOptions, *types.Header) (*rpctypes.FrameSimulationResult, error)
	})
	if !ok {
		return errorResponse(req.ID, ErrCodeMethodNotFound, "Frame simulation unavailable")
	}
	result, err := backend.SimulateFrame(options, header)
	if err != nil {
		var rpcErr *rpctypes.RPCError
		if errors.As(err, &rpcErr) {
			return &Response{JSONRPC: "2.0", ID: req.ID, Error: rpcErr}
		}
		return errorResponse(req.ID, ErrCodeInternal, err.Error())
	}
	return successResponse(req.ID, result)
}
