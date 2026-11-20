package ethapi

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBlockSelectionWireFormsAndInvalidHashes(t *testing.T) {
	hash := "0x" + strings.Repeat("ab", 32)
	for _, raw := range []string{`"` + hash + `"`, `{"blockHash":"` + hash + `"}`, `{"blockHash":"` + hash + `","requireCanonical":false}`} {
		selection, err := parseBlockSelection(json.RawMessage(raw))
		if err != nil || selection.hash == nil || selection.hash.Hex() != hash || selection.requireCanonical {
			t.Fatalf("hash selector %s: %+v %v", raw, selection, err)
		}
	}
	for _, raw := range []string{`"0x2"`, `{"blockNumber":"0x2"}`, `2`} {
		selection, err := parseBlockSelection(json.RawMessage(raw))
		if err != nil || selection.hash != nil || selection.number != 2 {
			t.Fatalf("number selector %s: %+v %v", raw, selection, err)
		}
	}
	for _, raw := range []string{`null`, `{}`, `{"blockHash":null}`, `{"blockHash":"0x01"}`, `{"blockHash":"0x` + strings.Repeat("ab", 33) + `"}`, `{"blockHash":"0x` + strings.Repeat("gg", 32) + `"}`, `{"blockHash":"` + hash + `","blockNumber":"0x2"}`, `{"blockHash":"` + hash + `","requireCanonical":"true"}`} {
		if _, err := parseBlockSelection(json.RawMessage(raw)); err == nil || err.Code != ErrCodeInvalidParams {
			t.Fatalf("accepted malformed selector %s: %v", raw, err)
		}
	}
}
