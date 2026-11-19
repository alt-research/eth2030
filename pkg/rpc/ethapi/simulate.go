package ethapi

import (
	"encoding/json"
	"errors"
	"github.com/eth2030/eth2030/core/types"
	rpcbackend "github.com/eth2030/eth2030/rpc/backend"
	rpctypes "github.com/eth2030/eth2030/rpc/types"
	"strings"
)

func (api *EthAPI) simulateV1(req *Request) *Response {
	if len(req.Params) < 1 || len(req.Params) > 2 {
		return errorResponse(req.ID, ErrCodeInvalidParams, "expected simulation options and optional base block")
	}
	var options rpctypes.SimulateOptions
	if err := json.Unmarshal(req.Params[0], &options); err != nil {
		return errorResponse(req.ID, ErrCodeInvalidParams, err.Error())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(req.Params[0], &raw); err != nil || raw["blockStateCalls"] == nil || string(raw["blockStateCalls"]) == "null" {
		return errorResponse(req.ID, ErrCodeInvalidParams, "blockStateCalls is required")
	}
	header := api.backend.CurrentHeader()
	if len(req.Params) > 1 {
		var value string
		if err := json.Unmarshal(req.Params[1], &value); err != nil {
			return errorResponse(req.ID, ErrCodeInvalidParams, err.Error())
		}
		if strings.HasPrefix(value, "0x") && len(value) == 66 {
			header = api.backend.HeaderByHash(types.HexToHash(value))
		} else {
			var number BlockNumber
			if err := json.Unmarshal(req.Params[1], &number); err != nil {
				return errorResponse(req.ID, ErrCodeInvalidParams, err.Error())
			}
			header = api.backend.HeaderByNumber(number)
		}
	}
	if header == nil {
		return errorResponse(req.ID, ErrCodeInternal, "base block not found")
	}
	backend, ok := api.backend.(rpcbackend.SimulateBackend)
	if !ok {
		return errorResponse(req.ID, ErrCodeMethodNotFound, "stateful simulation unavailable")
	}
	result, err := backend.Simulate(options, header)
	if err != nil {
		var rpcError *rpctypes.RPCError
		if errors.As(err, &rpcError) {
			return &Response{JSONRPC: "2.0", ID: req.ID, Error: rpcError}
		}
		return errorResponse(req.ID, ErrCodeInternal, err.Error())
	}
	return successResponse(req.ID, result)
}
