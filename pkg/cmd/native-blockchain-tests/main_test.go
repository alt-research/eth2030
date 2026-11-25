package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInvalidInvocationCannotReportConformance(t *testing.T) {
	for _, args := range [][]string{nil, {"-fork", "Unsupported", "missing"}, {"missing", "extra"}} {
		var out, diagnostic bytes.Buffer
		if code := run(args, &out, &diagnostic); code != 2 || out.Len() != 0 {
			t.Fatalf("args=%v code=%d output=%s", args, code, out.String())
		}
	}
}

func TestActualBinarySourceAndCompleteCase(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "native-blockchain-tests")
	revision, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	stamp := "-X main.sourceRevision=" + strings.TrimSpace(string(revision)) + " -X main.sourceModified=true"
	if output, err := exec.Command("go", "build", "-buildvcs=true", "-ldflags", stamp, "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v %s", err, output)
	}
	fixture, err := filepath.Abs("../../core/eftest/testdata/blockchain/bal.json")
	if err != nil {
		t.Fatal(err)
	}
	// Running outside the source checkout must retain the compiled revision.
	command := exec.Command(binary, "-fork", "Amsterdam", fixture)
	command.Dir = t.TempDir()
	raw, err := command.Output()
	if err != nil {
		t.Fatalf("native fixture: %v output=%s", err, raw)
	}
	var result report
	if err = json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	if len(result.NativeRevision) != 40 || result.Passed != 1 || result.Failed != 0 || len(result.Errors) != 0 || len(result.Cases) != 1 || len(result.Cases[0].Blocks) != 1 {
		t.Fatalf("incorrect/unbound report %+v", result)
	}
	for _, input := range []string{"{}", "not JSON"} {
		file := filepath.Join(t.TempDir(), "invalid.json")
		if err = os.WriteFile(file, []byte(input), 0600); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(binary, "-fork", "Amsterdam", file)
		raw, err = cmd.Output()
		if err == nil {
			t.Fatal("invalid fixture input reported success")
		}
		if json.Unmarshal(raw, &result) != nil || len(result.Errors) == 0 || result.Passed != 0 {
			t.Fatalf("missing fixture failure: %s", raw)
		}
	}
}
