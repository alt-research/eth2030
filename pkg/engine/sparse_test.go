package engine

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/eth2030/eth2030/core/types"
)

type custodyBackend struct {
	handlerMockBackend
	mask           types.CellMask
	changes, calls int
}

func (b *custodyBackend) SetCustodyColumns(mask types.CellMask) {
	if mask != b.mask {
		b.mask = mask
		b.changes++
	}
}
func (b *custodyBackend) ForkchoiceUpdatedV4(state ForkchoiceStateV1, attrs *PayloadAttributesV4) (ForkchoiceUpdatedResult, error) {
	b.calls++
	return b.handlerMockBackend.ForkchoiceUpdatedV4(state, attrs)
}
func TestSparseCustodyUpdatesReachConsumerWithoutSwallowingForkchoice(t *testing.T) {
	backend := new(custodyBackend)
	api := NewEngineAPI(backend)
	state := json.RawMessage(fmt.Sprintf(`{"headBlockHash":"0x%s","safeBlockHash":"0x%s","finalizedBlockHash":"0x%s"}`, strings.Repeat("00", 32), strings.Repeat("00", 32), strings.Repeat("00", 32)))
	mask := json.RawMessage(`"0xff000000000000000000000000000000"`)
	for _, value := range []json.RawMessage{mask, mask, json.RawMessage(`null`)} {
		if _, err := api.handleForkchoiceUpdatedV4([]json.RawMessage{state, json.RawMessage(`null`), value}); err != nil {
			t.Fatal(err)
		}
	}
	if backend.changes != 1 || backend.calls != 3 || backend.mask[0] != 255 {
		t.Fatalf("custody update/rest FCU lost: changes=%d calls=%d", backend.changes, backend.calls)
	}
	for _, value := range []string{`"0xff"`, `"ff000000000000000000000000000000"`, `"0xgg000000000000000000000000000000"`, `7`} {
		if _, err := api.handleForkchoiceUpdatedV4([]json.RawMessage{state, json.RawMessage(`null`), json.RawMessage(value)}); err == nil {
			t.Fatalf("malformed custody accepted: %s", value)
		}
	}
	if backend.changes != 1 || backend.calls != 3 {
		t.Fatal("invalid custody mutated consumer or applied forkchoice")
	}
}
func TestSparseGetBlobsValidatesBoundedCanonicalInput(t *testing.T) {
	api := newHandlerTestAPI()
	hash := `["0x01` + strings.Repeat("00", 31) + `"]`
	mask := `"0x01000000000000000000000000000000"`
	for _, params := range [][]json.RawMessage{{json.RawMessage(hash)}, {json.RawMessage(`null`), json.RawMessage(mask)}, {json.RawMessage(`["0x01"]`), json.RawMessage(mask)}, {json.RawMessage(`["0x02` + strings.Repeat("00", 31) + `"]`), json.RawMessage(mask)}, {json.RawMessage(hash), json.RawMessage(`null`)}, {json.RawMessage(hash), json.RawMessage(`"0x01"`)}} {
		if _, err := api.handleGetBlobsV4(params); err == nil {
			t.Fatal("malformed cell request accepted")
		}
	}
	hashes := make([]string, 129)
	for i := range hashes {
		hashes[i] = "0x01" + strings.Repeat("00", 31)
	}
	raw, _ := json.Marshal(hashes)
	if _, err := api.handleGetBlobsV4([]json.RawMessage{raw, json.RawMessage(mask)}); err == nil || err.Code != TooLargeRequestCode {
		t.Fatal("unbounded cell request accepted")
	}
}
