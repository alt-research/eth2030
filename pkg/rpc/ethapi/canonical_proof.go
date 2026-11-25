package ethapi

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/eth2030/eth2030/zkvm/canonicalproof"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type canonicalProofProvider interface {
	CanonicalProofService() *canonicalproof.Service
}

type canonicalExecutionRequest struct {
	Program hexutil.Bytes          `json:"program"`
	Input   hexutil.Bytes          `json:"input"`
	Config  *canonicalproof.Config `json:"config,omitempty"`
	Proof   *canonicalproof.Proof  `json:"proof,omitempty"`
}

func (api *EthAPI) canonicalProofRequest(req *Request) *Response {
	provider, ok := api.backend.(canonicalProofProvider)
	if !ok || provider.CanonicalProofService() == nil {
		return errorResponse(req.ID, ErrCodeMethodNotFound, "canonical proof service unavailable")
	}
	service := provider.CanonicalProofService()
	if err := req.Context().Err(); err != nil {
		return errorResponse(req.ID, -32000, err.Error())
	}
	if len(req.Params) != 1 {
		return errorResponse(req.ID, ErrCodeInvalidParams, "expected exactly one parameter")
	}
	if req.Method == "zkvm_getProof" || req.Method == "zkvm_cancelProof" {
		var id string
		if json.Unmarshal(req.Params[0], &id) != nil || len(id) != 32 {
			return errorResponse(req.ID, ErrCodeInvalidParams, "invalid proof job ID")
		}
		if _, err := hex.DecodeString(id); err != nil {
			return errorResponse(req.ID, ErrCodeInvalidParams, "invalid proof job ID")
		}
		if req.Method == "zkvm_cancelProof" {
			if err := service.Cancel(id); err != nil {
				return canonicalProofError(req, err)
			}
		}
		job, err := service.Job(id)
		if err != nil {
			return canonicalProofError(req, err)
		}
		return successResponse(req.ID, job)
	}
	var input canonicalExecutionRequest
	decoder := json.NewDecoder(bytes.NewReader(req.Params[0]))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return errorResponse(req.ID, ErrCodeInvalidParams, err.Error())
	}
	cfg := canonicalproof.DefaultConfig()
	if input.Config != nil {
		cfg = *input.Config
	}
	if cfg.Validate() != nil || len(input.Program) == 0 || len(input.Program) > canonicalproof.MaxProgram || len(input.Input) > canonicalproof.MaxInput {
		return errorResponse(req.ID, ErrCodeInvalidParams, "invalid program/input/resource bounds")
	}
	if req.Method == "zkvm_proveExecution" {
		if input.Proof != nil {
			return errorResponse(req.ID, ErrCodeInvalidParams, "proving does not accept an existing proof")
		}
		job, err := service.Submit(input.Program, input.Input, cfg)
		if err != nil {
			return canonicalProofError(req, err)
		}
		return successResponse(req.ID, job)
	}
	if input.Proof == nil {
		return errorResponse(req.ID, ErrCodeInvalidParams, "missing execution receipt")
	}
	err := service.Verify(req.Context(), input.Proof, canonicalproof.Expected{Program: input.Program, Input: input.Input, Config: cfg})
	if err != nil {
		if errors.Is(err, canonicalproof.ErrUnavailable) || errors.Is(err, canonicalproof.ErrBusy) || errors.Is(err, canonicalproof.ErrClosed) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return canonicalProofError(req, err)
		}
		return successResponse(req.ID, map[string]any{"valid": false})
	}
	return successResponse(req.ID, map[string]any{"valid": true, "statement": input.Proof.Statement})
}

func canonicalProofError(req *Request, err error) *Response {
	code := -32000
	if errors.Is(err, canonicalproof.ErrBounds) {
		code = ErrCodeInvalidParams
	}
	if errors.Is(err, canonicalproof.ErrBusy) {
		code = -32005
	}
	if errors.Is(err, canonicalproof.ErrUnavailable) {
		code = -32001
	}
	return errorResponse(req.ID, code, err.Error())
}
