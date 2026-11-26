package payload

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/eth2030/eth2030/core/types"
)

func TestDependency8504PayloadAndBodyJSONPresence(t *testing.T) {
	for _, w := range [][][]byte{nil, {}, {{1, 2, 3}}} {
		p := ExecutionPayloadV3{DependencyWitnesses: w}
		raw, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(raw, []byte(`"dependencyWitnesses"`)) != (w != nil) {
			t.Fatal("payload presence", string(raw))
		}
		if w != nil && len(w) == 0 && !bytes.Contains(raw, []byte(`"dependencyWitnesses":[]`)) {
			t.Fatal("active empty encoded null")
		}
		if len(w) > 0 && !bytes.Contains(raw, []byte(`"0x010203"`)) {
			t.Fatal("witness not DATA hex")
		}
		var decoded ExecutionPayloadV3
		if err := json.Unmarshal(raw, &decoded); err != nil || (decoded.DependencyWitnesses != nil) != (w != nil) || len(decoded.DependencyWitnesses) != len(w) {
			t.Fatal("payload roundtrip", err)
		}
		b := types.NewBlock(&types.Header{}, &types.Body{DependencyWitnesses: w})
		for _, body := range []any{BlockToPayloadBodyV1(b), BlockToPayloadBodyV2(b)} {
			raw, err := json.Marshal(body)
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Contains(raw, []byte(`"dependencyWitnesses"`)) != (w != nil) {
				t.Fatal("retrieval presence", string(raw))
			}
		}
	}
}
