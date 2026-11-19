package node

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eth2030/eth2030/engine/payload"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
)

func TestOriginalRecoveringCLWireBindsIndependentCanonicalHeaders(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "docs", "reviews", "pr-104", "ORIGINAL_WIRE.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Observed struct {
			Exchange struct {
				RequestBody  string `json:"request_body"`
				ResponseBody string `json:"response_body"`
			}
			Request struct {
				Method string
				Params []json.RawMessage
			}
			Response struct {
				Error struct {
					Code    int
					Message string
				}
			}
			PriorSuccess struct {
				Request  struct{ Params []json.RawMessage }
				Response struct {
					Result struct{ PayloadStatus struct{ Status string } }
				}
			} `json:"prior_success"`
		}
		Headers map[string]map[string]struct {
			ByHash        json.RawMessage `json:"by_hash"`
			CanonicalHash string          `json:"current_canonical_hash_at_number"`
		} `json:"historical_headers"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	observed := fixture.Observed
	if observed.Response.Error.Code != -38002 || !strings.Contains(observed.Response.Error.Message, "finalized block regresses") || observed.PriorSuccess.Response.Result.PayloadStatus.Status != "VALID" {
		t.Fatal("original counterexample classification changed")
	}
	request, err := base64.StdEncoding.DecodeString(observed.Exchange.RequestBody)
	if err != nil {
		t.Fatal(err)
	}
	var requestWire struct {
		Method string
		Params []json.RawMessage
	}
	if err := json.Unmarshal(request, &requestWire); err != nil {
		t.Fatal(err)
	}
	var originalRequest, decodedRequest bytes.Buffer
	if err := json.Compact(&originalRequest, observed.Request.Params[0]); err != nil {
		t.Fatal(err)
	}
	if err := json.Compact(&decodedRequest, requestWire.Params[0]); err != nil {
		t.Fatal(err)
	}
	if requestWire.Method != observed.Request.Method || len(requestWire.Params) != 2 || !bytes.Equal(decodedRequest.Bytes(), originalRequest.Bytes()) {
		t.Fatal("decoded request differs from exact captured bytes")
	}
	response, err := base64.StdEncoding.DecodeString(observed.Exchange.ResponseBody)
	if err != nil {
		t.Fatal(err)
	}
	var responseWire struct {
		Error struct {
			Code    int
			Message string
		}
	}
	if err := json.Unmarshal(response, &responseWire); err != nil {
		t.Fatal(err)
	}
	if responseWire.Error.Code != observed.Response.Error.Code || responseWire.Error.Message != observed.Response.Error.Message {
		t.Fatal("decoded error differs from exact captured bytes")
	}
	var incoming, prior payload.ForkchoiceStateV1
	if err := json.Unmarshal(observed.Request.Params[0], &incoming); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(observed.PriorSuccess.Request.Params[0], &prior); err != nil {
		t.Fatal(err)
	}
	decoded := make(map[string]*gethtypes.Header)
	for hash, observations := range fixture.Headers {
		var independentEncoding []byte
		for endpoint, observation := range observations {
			if string(observation.ByHash) == "null" {
				continue
			}
			var header gethtypes.Header
			if err := json.Unmarshal(observation.ByHash, &header); err != nil {
				t.Fatal(err)
			}
			if header.Hash().Hex() != hash {
				t.Fatalf("%s original complete header fails independent hash", endpoint)
			}
			encoded, err := gethrlp.EncodeToBytes(&header)
			if err != nil {
				t.Fatal(err)
			}
			if independentEncoding == nil {
				independentEncoding = encoded
			} else if !bytes.Equal(independentEncoding, encoded) {
				t.Fatal("native/independent full header encodings differ")
			}
			decoded[hash] = &header
		}
	}
	for _, hash := range []string{incoming.HeadBlockHash.Hex(), incoming.SafeBlockHash.Hex(), incoming.FinalizedBlockHash.Hex()} {
		for _, endpoint := range []string{"32827", "32858"} {
			observation := fixture.Headers[hash][endpoint]
			if string(observation.ByHash) == "null" || observation.CanonicalHash != hash {
				t.Fatal("recovering notification is not a retained independently canonical header")
			}
		}
	}
	if decoded[incoming.HeadBlockHash.Hex()].Number.Cmp(decoded[prior.FinalizedBlockHash.Hex()].Number) >= 0 || decoded[incoming.FinalizedBlockHash.Hex()].Number.Cmp(decoded[prior.FinalizedBlockHash.Hex()].Number) >= 0 {
		t.Fatal("fixture no longer describes a stale notification below durable finality")
	}
}
