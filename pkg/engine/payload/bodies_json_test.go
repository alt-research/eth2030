package payload

import (
	"encoding/json"
	"testing"

	"github.com/eth2030/eth2030/core/types"
)

func TestPayloadBodyNullAndEmptyWire(t *testing.T) {
	pre := BlockToPayloadBodyV2(types.NewBlock(&types.Header{}, &types.Body{}))
	raw, err := json.Marshal(pre)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"transactions":[],"withdrawals":null,"blockAccessList":null}` {
		t.Fatalf("pre-fork wire: %s", raw)
	}
	post := &ExecutionPayloadBodyV2{Transactions: [][]byte{{6, 1, 2}}, Withdrawals: []*Withdrawal{}, BlockAccessList: []byte{0xc0}}
	raw, err = json.Marshal(post)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"transactions":["0x060102"],"withdrawals":[],"blockAccessList":"0xc0"}` {
		t.Fatalf("post-fork wire: %s", raw)
	}
	var decoded ExecutionPayloadBodyV2
	if err := json.Unmarshal(raw, &decoded); err != nil || len(decoded.BlockAccessList) != 1 || decoded.BlockAccessList[0] != 0xc0 || len(decoded.Transactions) != 1 {
		t.Fatalf("historical body decode: %+v %v", decoded, err)
	}
	if err := json.Unmarshal([]byte(`{"transactions":["BgEC"],"withdrawals":[],"blockAccessList":"0xc0"}`), &decoded); err == nil {
		t.Fatal("accepted Go base64 in an Engine DATA field")
	}
}
