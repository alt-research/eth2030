package rpctypes

import "encoding/json"

// FrameSimulationOptions is the ETH2030 v1 client extension, not a consensus
// envelope or an unsigned transaction submitted through a public pool.
type FrameSimulationOptions struct {
	RawTransaction string                  `json:"rawTransaction"`
	SlotNumber     *string                 `json:"slotNumber,omitempty"`
	BlockOverrides *SimulateBlockOverrides `json:"blockOverrides,omitempty"`
	StateOverrides StateOverride           `json:"stateOverrides,omitempty"`
	Trace          bool                    `json:"trace,omitempty"`
	TraceLimit     *string                 `json:"traceLimit,omitempty"`
}

func (o *FrameSimulationOptions) UnmarshalJSON(data []byte) error {
	if err := knownOverrideFields(data, "rawTransaction", "slotNumber", "blockOverrides", "stateOverrides", "trace", "traceLimit"); err != nil {
		return err
	}
	type plain FrameSimulationOptions
	return json.Unmarshal(data, (*plain)(o))
}

type FrameSimulationFrame struct {
	Index            string    `json:"index"`
	Status           string    `json:"status"`
	ExecutionGasUsed string    `json:"executionGasUsed"`
	StateGasUsed     string    `json:"stateGasUsed"`
	ReturnData       string    `json:"returnData"`
	Logs             []*RPCLog `json:"logs"`
	Error            string    `json:"error,omitempty"`
	StructLogs       any       `json:"structLogs,omitempty"`
}
type FrameSimulationResult struct {
	Valid                bool                   `json:"valid"`
	Hash                 string                 `json:"hash"`
	SigningHash          string                 `json:"signingHash"`
	Payer                *string                `json:"payer,omitempty"`
	IntrinsicGas         string                 `json:"intrinsicGas"`
	CalldataFloor        string                 `json:"calldataFloor"`
	ExecutionReservation string                 `json:"executionReservation"`
	StateReservation     string                 `json:"stateReservation"`
	GasUsed              string                 `json:"gasUsed"`
	BlobGasUsed          string                 `json:"blobGasUsed"`
	BlobCost             string                 `json:"blobCost"`
	Escrow               string                 `json:"escrow"`
	Refund               string                 `json:"refund"`
	Tip                  string                 `json:"tip"`
	Frames               []FrameSimulationFrame `json:"frames"`
	Error                string                 `json:"error,omitempty"`
}
