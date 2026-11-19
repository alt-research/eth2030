// Command native-state-tests executes signed EEST state fixtures with ETH2030.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/eth2030/eth2030/core/eftest"
)

type caseResult struct {
	File         string `json:"file"`
	Name         string `json:"name"`
	Fork         string `json:"fork"`
	Index        int    `json:"index"`
	Passed       bool   `json:"passed"`
	ExpectedRoot string `json:"expected_root"`
	GotRoot      string `json:"got_root"`
	Error        string `json:"error,omitempty"`
}
type report struct {
	NativeRevision string       `json:"native_revision"`
	NativeDirty    bool         `json:"native_dirty"`
	Executor       string       `json:"executor"`
	Fork           string       `json:"fork"`
	Passed         int          `json:"passed"`
	Failed         int          `json:"failed"`
	Cases          []caseResult `json:"cases"`
	Errors         []string     `json:"errors"`
}

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("native-state-tests", flag.ContinueOnError)
	flags.SetOutput(stderr)
	fork := flags.String("fork", "Amsterdam", "exact fixture fork to execute")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 1 || !eftest.ForkSupported(*fork) {
		fmt.Fprintln(stderr, "usage: native-state-tests -fork Osaka|Amsterdam FIXTURE_DIRECTORY")
		return 2
	}
	r := report{Executor: "eth2030-native-signed-state-transition", Fork: *fork, Cases: []caseResult{}, Errors: []string{}}
	if b, err := exec.Command("git", "rev-parse", "HEAD").Output(); err == nil {
		r.NativeRevision = strings.TrimSpace(string(b))
	}
	if b, err := exec.Command("git", "status", "--porcelain").Output(); err == nil {
		r.NativeDirty = len(b) > 0
	}
	files, err := eftest.DiscoverFixtures(flags.Arg(0))
	if err != nil {
		r.Errors = append(r.Errors, err.Error())
	}
	for _, file := range files {
		tests, err := eftest.LoadStateTests(file)
		if err != nil {
			r.Errors = append(r.Errors, fmt.Sprintf("%s: %v", file, err))
			continue
		}
		for name, st := range tests {
			for _, sub := range st.Subtests() {
				if sub.Fork != *fork {
					continue
				}
				result := st.RunSigned(sub)
				relative, err := filepath.Rel(flags.Arg(0), file)
				if err != nil {
					relative = file
				}
				c := caseResult{File: relative, Name: name, Fork: sub.Fork, Index: sub.Index, Passed: result.Passed, ExpectedRoot: result.ExpectedRoot.Hex(), GotRoot: result.GotRoot.Hex()}
				if result.Error != nil {
					c.Error = result.Error.Error()
				}
				if c.Passed {
					r.Passed++
				} else {
					r.Failed++
				}
				r.Cases = append(r.Cases, c)
				if len(r.Cases)%1000 == 0 {
					fmt.Fprintf(stderr, "native %s: %d passed, %d failed (%d selected)\n", *fork, r.Passed, r.Failed, len(r.Cases))
				}
			}
		}
	}
	if len(r.Cases) == 0 {
		r.Errors = append(r.Errors, "no selected-fork signed fixtures executed")
	}
	sort.Slice(r.Cases, func(i, j int) bool {
		a, b := r.Cases[i], r.Cases[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		return a.Index < b.Index
	})
	if err := json.NewEncoder(stdout).Encode(r); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if r.Failed > 0 || len(r.Errors) > 0 {
		return 1
	}
	return 0
}
