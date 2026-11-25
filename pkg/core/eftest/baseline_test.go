package eftest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLockedNativeReferenceBaseline(t *testing.T) {
	r, err := RunBaseline("testdata/hegota/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if r.Executed != 12 || r.Passed != 12 || r.Skipped != 0 || r.Failed != 0 {
		t.Fatalf("incomplete execution: %+v", r)
	}
}
func TestBaselineRejectsInvalidOrUnsupportedEvidence(t *testing.T) {
	original, err := os.ReadFile("testdata/hegota/cancun-add11.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"missing fixture", "hash mismatch", "zero root", "missing logs", "negative index", "exception wildcard", "unsupported tx fields", "empty corpus", "case count"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			var raw map[string]map[string]any
			if err := json.Unmarshal(original, &raw); err != nil {
				t.Fatal(err)
			}
			j := raw["add11"]
			post := j["post"].(map[string]any)["Berlin"].([]any)[0].(map[string]any)
			switch mode {
			case "zero root":
				post["hash"] = "0x0000000000000000000000000000000000000000000000000000000000000000"
			case "missing logs":
				delete(post, "logs")
			case "negative index":
				post["indexes"].(map[string]any)["data"] = -1
			case "exception wildcard":
				post["expectException"] = "anything"
			case "unsupported tx fields":
				j["transaction"].(map[string]any)["accessLists"] = []any{}
			case "empty corpus":
				raw = map[string]map[string]any{}
			}
			data, _ := json.Marshal(raw)
			hash := sha256.Sum256(data)
			digest := hex.EncodeToString(hash[:])
			if mode == "hash mismatch" {
				digest = "bad"
			}
			if mode != "missing fixture" {
				if err := os.WriteFile(filepath.Join(dir, "fixture.json"), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			m := BaselineManifest{Version: 1, ExpectedCases: 5, Fixtures: []BaselineFixture{{Path: "fixture.json", SHA256: digest, Cases: 5, Source: "test"}}}
			if mode == "case count" {
				m.ExpectedCases = 6
			}
			b, _ := json.Marshal(m)
			path := filepath.Join(dir, "manifest.json")
			if err := os.WriteFile(path, b, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := RunBaseline(path); err == nil {
				t.Fatal("invalid evidence reported as a pass")
			}
		})
	}
}
