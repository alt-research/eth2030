package ethapi

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/eth2030/eth2030/core/types"
)

type blockSelection struct {
	number           BlockNumber
	hash             *types.Hash
	requireCanonical bool
}

func parseBlockSelection(raw json.RawMessage) (blockSelection, *RPCError) {
	selection := blockSelection{number: LatestBlockNumber}
	if len(raw) == 0 {
		return selection, nil
	}
	invalid := func(err error) (blockSelection, *RPCError) {
		return selection, &RPCError{Code: ErrCodeInvalidParams, Message: "invalid block selector: " + err.Error()}
	}
	decodeHash := func(text string) error {
		if len(text) != 66 || text[:2] != "0x" {
			return fmt.Errorf("blockHash must contain exactly 32 bytes")
		}
		value, err := hex.DecodeString(text[2:])
		if err != nil {
			return err
		}
		hash := types.BytesToHash(value)
		selection.hash = &hash
		return nil
	}
	raw = bytes.TrimSpace(raw)
	if bytes.Equal(raw, []byte("null")) {
		return invalid(fmt.Errorf("null block selector"))
	}
	if len(raw) > 0 && raw[0] == '{' {
		var object struct {
			BlockHash        *string      `json:"blockHash"`
			BlockNumber      *BlockNumber `json:"blockNumber"`
			RequireCanonical bool         `json:"requireCanonical"`
		}
		if err := json.Unmarshal(raw, &object); err != nil {
			return invalid(err)
		}
		if (object.BlockHash == nil) == (object.BlockNumber == nil) {
			return invalid(fmt.Errorf("specify exactly one of blockHash or blockNumber"))
		}
		if object.BlockHash != nil {
			if err := decodeHash(*object.BlockHash); err != nil {
				return invalid(err)
			}
			selection.requireCanonical = object.RequireCanonical
		} else {
			selection.number = *object.BlockNumber
		}
		return selection, nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		if err := json.Unmarshal(raw, &selection.number); err != nil {
			return invalid(err)
		}
		return selection, nil
	}
	if len(text) == 66 {
		if err := decodeHash(text); err != nil {
			return invalid(err)
		}
	} else if err := json.Unmarshal(raw, &selection.number); err != nil {
		return invalid(err)
	}
	return selection, nil
}
func resolveBlockSelection(backend Backend, selection blockSelection) (*types.Header, *RPCError) {
	var header *types.Header
	if selection.hash != nil {
		header = backend.HeaderByHash(*selection.hash)
	} else {
		header = backend.HeaderByNumber(selection.number)
	}
	if header == nil {
		return nil, &RPCError{Code: -32001, Message: "block not found"}
	}
	if selection.hash != nil && selection.requireCanonical {
		canonical := backend.HeaderByNumber(BlockNumber(header.Number.Int64()))
		if canonical == nil || canonical.Hash() != *selection.hash {
			return nil, &RPCError{Code: -32000, Message: "requested block is not canonical"}
		}
	}
	return header, nil
}
func resolveRawBlockHeader(backend Backend, raw json.RawMessage) (*types.Header, *RPCError) {
	selection, err := parseBlockSelection(raw)
	if err != nil {
		return nil, err
	}
	return resolveBlockSelection(backend, selection)
}
