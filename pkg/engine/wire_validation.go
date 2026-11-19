package engine

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
)

func isNullWire(raw json.RawMessage) bool { return bytes.Equal(bytes.TrimSpace(raw), []byte("null")) }

// Engine wire validation must run before permissive internal decoders can pad
// DATA, truncate quantities or replace absent fields with zero values.
func wireObject(raw json.RawMessage, fields ...string) (map[string]json.RawMessage, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return nil, fmt.Errorf("expected object")
	}
	for _, field := range fields {
		value, ok := object[field]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return nil, fmt.Errorf("missing %s", field)
		}
	}
	return object, nil
}

func wireData(raw json.RawMessage, size int) error {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil || !strings.HasPrefix(value, "0x") {
		return fmt.Errorf("expected hex DATA")
	}
	decoded, err := hex.DecodeString(value[2:])
	if err != nil || (size >= 0 && len(decoded) != size) {
		return fmt.Errorf("invalid DATA width or encoding")
	}
	return nil
}

func wireQuantity(raw json.RawMessage, bits int) error {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil || !strings.HasPrefix(value, "0x") || len(value) < 3 || (len(value) > 3 && value[2] == '0') {
		return fmt.Errorf("expected canonical hex QUANTITY")
	}
	for _, digit := range value[2:] {
		if !strings.ContainsRune("0123456789abcdefABCDEF", digit) {
			return fmt.Errorf("invalid QUANTITY digit")
		}
	}
	n, ok := new(big.Int).SetString(value[2:], 16)
	if !ok || n.BitLen() > bits {
		return fmt.Errorf("QUANTITY exceeds %d bits", bits)
	}
	return nil
}

func wireArray(raw json.RawMessage) ([]json.RawMessage, error) {
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil || items == nil {
		return nil, fmt.Errorf("expected array, including [] for empty")
	}
	return items, nil
}

func wireWithdrawals(raw json.RawMessage, validatorIndex string) error {
	items, err := wireArray(raw)
	if err != nil {
		return err
	}
	for _, item := range items {
		w, err := wireObject(item, "index", validatorIndex, "address", "amount")
		if err != nil {
			return err
		}
		for _, field := range []string{"index", validatorIndex, "amount"} {
			if err := wireQuantity(w[field], 64); err != nil {
				return fmt.Errorf("withdrawal %s: %w", field, err)
			}
		}
		if err := wireData(w["address"], 20); err != nil {
			return err
		}
	}
	return nil
}

func validateNewPayloadWire(params []json.RawMessage, amsterdam bool) error {
	fields := []string{"parentHash", "feeRecipient", "stateRoot", "receiptsRoot", "logsBloom", "prevRandao", "blockNumber", "gasLimit", "gasUsed", "timestamp", "extraData", "baseFeePerGas", "blockHash", "transactions", "withdrawals", "blobGasUsed", "excessBlobGas"}
	if amsterdam {
		fields = append(fields, "blockAccessList", "slotNumber")
	}
	p, err := wireObject(params[0], fields...)
	if err != nil {
		return err
	}
	for _, field := range []string{"parentHash", "stateRoot", "receiptsRoot", "prevRandao", "blockHash"} {
		if err := wireData(p[field], 32); err != nil {
			return fmt.Errorf("%s: %w", field, err)
		}
	}
	for field, size := range map[string]int{"feeRecipient": 20, "logsBloom": 256, "extraData": -1} {
		if err := wireData(p[field], size); err != nil {
			return fmt.Errorf("%s: %w", field, err)
		}
	}
	for _, field := range []string{"blockNumber", "gasLimit", "gasUsed", "timestamp", "blobGasUsed", "excessBlobGas"} {
		if err := wireQuantity(p[field], 64); err != nil {
			return fmt.Errorf("%s: %w", field, err)
		}
	}
	if err := wireQuantity(p["baseFeePerGas"], 256); err != nil {
		return err
	}
	if amsterdam {
		if err := wireQuantity(p["slotNumber"], 64); err != nil {
			return err
		}
		if err := wireData(p["blockAccessList"], -1); err != nil {
			return err
		}
	}
	transactions, err := wireArray(p["transactions"])
	if err != nil {
		return err
	}
	for _, tx := range transactions {
		if err := wireData(tx, -1); err != nil {
			return err
		}
	}
	if err := wireWithdrawals(p["withdrawals"], "validatorIndex"); err != nil {
		return err
	}
	hashes, err := wireArray(params[1])
	if err != nil {
		return err
	}
	for _, hash := range hashes {
		if err := wireData(hash, 32); err != nil {
			return err
		}
	}
	return wireData(params[2], 32)
}

func validateForkchoiceWire(raw json.RawMessage) error {
	state, err := wireObject(raw, "headBlockHash", "safeBlockHash", "finalizedBlockHash")
	if err != nil {
		return err
	}
	for _, field := range []string{"headBlockHash", "safeBlockHash", "finalizedBlockHash"} {
		if err := wireData(state[field], 32); err != nil {
			return err
		}
	}
	return nil
}

func validateAttributesWire(raw json.RawMessage, amsterdam bool) error {
	object, err := wireObject(raw)
	if err != nil {
		return err
	}
	randao, recipient, root, slot, target, index := "prevRandao", "suggestedFeeRecipient", "parentBeaconBlockRoot", "slotNumber", "targetGasLimit", "validatorIndex"
	if _, snake := object["prev_randao"]; amsterdam && snake {
		randao, recipient, root, slot, target, index = "prev_randao", "suggested_fee_recipient", "parent_beacon_block_root", "slot_number", "target_gas_limit", "validator_index"
	}
	fields := []string{"timestamp", randao, recipient, root, "withdrawals"}
	if amsterdam {
		fields = append(fields, slot, target)
	}
	if _, err := wireObject(raw, fields...); err != nil {
		return err
	}
	if err := wireQuantity(object["timestamp"], 64); err != nil {
		return err
	}
	if amsterdam {
		if err := wireQuantity(object[slot], 64); err != nil {
			return err
		}
		if err := wireQuantity(object[target], 64); err != nil {
			return err
		}
	}
	for field, size := range map[string]int{randao: 32, recipient: 20, root: 32} {
		if err := wireData(object[field], size); err != nil {
			return err
		}
	}
	return wireWithdrawals(object["withdrawals"], index)
}

func (api *EngineAPI) rejectAttributes(state ForkchoiceStateV1, amsterdam bool) (any, *jsonrpcError) {
	var result ForkchoiceUpdatedResult
	var err error
	if amsterdam {
		result, err = api.backend.ForkchoiceUpdatedV4(state, nil)
	} else {
		result, err = api.backend.ForkchoiceUpdated(state, nil)
	}
	result, err = forkchoiceAttributeError(result, err, ErrInvalidPayloadAttributes)
	if err != nil {
		return nil, engineErrorToRPC(err)
	}
	return result, nil
}
