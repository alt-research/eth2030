package rpctypes

import (
	"encoding/json"
	"fmt"
	"strings"
)

type SimulateOptions struct {
	BlockStateCalls        []SimulateBlock `json:"blockStateCalls"`
	Validation             bool            `json:"validation"`
	TraceTransfers         bool            `json:"traceTransfers"`
	ReturnFullTransactions bool            `json:"returnFullTransactions"`
}
type SimulateBlock struct {
	BlockOverrides *SimulateBlockOverrides `json:"blockOverrides,omitempty"`
	StateOverrides StateOverride           `json:"stateOverrides,omitempty"`
	Calls          []CallArgs              `json:"calls,omitempty"`
}
type SimulateBlockOverrides struct {
	Number        *string           `json:"number,omitempty"`
	Time          *string           `json:"time,omitempty"`
	GasLimit      *string           `json:"gasLimit,omitempty"`
	FeeRecipient  *string           `json:"feeRecipient,omitempty"`
	PrevRandao    *string           `json:"prevRandao,omitempty"`
	BaseFeePerGas *string           `json:"baseFeePerGas,omitempty"`
	BlobBaseFee   *string           `json:"blobBaseFee,omitempty"`
	BeaconRoot    *string           `json:"beaconRoot,omitempty"`
	Withdrawals   *[]*RPCWithdrawal `json:"withdrawals,omitempty"`
}

func knownOverrideFields(data []byte, fields ...string) error {
	var values map[string]json.RawMessage
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}
	allowed := make(map[string]bool, len(fields))
	for _, field := range fields {
		allowed[strings.ToLower(field)] = true
	}
	seen := make(map[string]bool, len(values))
	for name := range values {
		key := strings.ToLower(name)
		if !allowed[key] {
			return fmt.Errorf("unsupported override field %q", name)
		}
		if seen[key] {
			return fmt.Errorf("duplicate override field %q", name)
		}
		seen[key] = true
	}
	return nil
}
func (o *SimulateBlockOverrides) UnmarshalJSON(data []byte) error {
	if err := knownOverrideFields(data, "number", "time", "gasLimit", "feeRecipient", "prevRandao", "baseFeePerGas", "blobBaseFee", "beaconRoot", "withdrawals"); err != nil {
		return err
	}
	type plain SimulateBlockOverrides
	return json.Unmarshal(data, (*plain)(o))
}
func (o *AccountOverride) UnmarshalJSON(data []byte) error {
	if err := knownOverrideFields(data, "nonce", "balance", "code", "state", "stateDiff", "movePrecompileToAddress"); err != nil {
		return err
	}
	type plain AccountOverride
	return json.Unmarshal(data, (*plain)(o))
}

type SimulateCallResult struct {
	ReturnData string    `json:"returnData"`
	GasUsed    string    `json:"gasUsed"`
	MaxUsedGas string    `json:"maxUsedGas,omitempty"`
	Status     string    `json:"status"`
	Logs       []*RPCLog `json:"logs"`
	Error      *RPCError `json:"error,omitempty"`
}
type SimulateBlockResult struct {
	Block any
	Calls []SimulateCallResult
}

func (r SimulateBlockResult) MarshalJSON() ([]byte, error) {
	raw, err := json.Marshal(r.Block)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	fields["calls"], err = json.Marshal(r.Calls)
	if err != nil {
		return nil, err
	}
	return json.Marshal(fields)
}
