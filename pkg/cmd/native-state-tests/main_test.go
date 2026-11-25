package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNativeFixtureCommandRejectsEmptyMalformedAndUnsupportedInputs(t *testing.T) {
	for _, kind := range []string{"empty", "malformed", "unsupported"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			fork := "Osaka"
			if kind == "malformed" {
				if err := os.WriteFile(filepath.Join(dir, "bad.json"), []byte("{bad"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "unsupported" {
				fork = "FutureFork"
			}
			var output, stderr bytes.Buffer
			if code := run([]string{"-fork", fork, dir}, &output, &stderr); code == 0 {
				t.Fatalf("invalid input succeeded: %s", output.String())
			}
		})
	}
}
func TestNativeFixtureCommandConsumesAllFiveOfficialEnvelopes(t *testing.T) {
	var output, stderr bytes.Buffer
	dir := filepath.Join("..", "..", "core", "eftest", "testdata", "eest-v5.4.0")
	if code := run([]string{"-fork", "Osaka", dir}, &output, &stderr); code != 0 {
		t.Fatalf("official input exit=%d stderr=%s output=%s", code, stderr.String(), output.String())
	}
	var result report
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Passed != 5 || result.Failed != 0 || len(result.Errors) != 0 || len(result.Cases) != 5 {
		t.Fatalf("incomplete official replay: %+v", result)
	}
	for _, c := range result.Cases {
		if !c.Passed || c.ExpectedRoot != c.GotRoot {
			t.Fatalf("invalid conformance receipt: %+v", c)
		}
	}
}
