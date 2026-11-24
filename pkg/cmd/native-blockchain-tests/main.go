// Command native-blockchain-tests imports official encoded blockchain fixtures.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"

	"github.com/eth2030/eth2030/core/eftest"
)

type caseResult struct {
	File string `json:"file"`
	Name string `json:"name"`
	Fork string `json:"fork"`
	eftest.BlockchainResult
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

// Explicit release stamps also support toolchains/build modes that omit vcs settings.
var sourceRevision string
var sourceModified string

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }
func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("native-blockchain-tests", flag.ContinueOnError)
	flags.SetOutput(stderr)
	fork := flags.String("fork", "Amsterdam", "exact fixture network (Osaka or Amsterdam)")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 1 || (*fork != "Osaka" && *fork != "Amsterdam") {
		fmt.Fprintln(stderr, "usage: native-blockchain-tests -fork Osaka|Amsterdam FIXTURE_FILE_OR_DIRECTORY")
		return 2
	}
	r := report{Executor: "eth2030-native-blockchain-import", Fork: *fork, Cases: []caseResult{}, Errors: []string{}}
	// Bind the report to the executable's compiled source, never the caller's
	// current checkout (the binary may have been built elsewhere).
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				r.NativeRevision = setting.Value
			case "vcs.modified":
				r.NativeDirty = setting.Value == "true"
			}
		}
	}
	if sourceRevision != "" {
		if r.NativeRevision != "" && r.NativeRevision != sourceRevision {
			r.Errors = append(r.Errors, "compiled source stamps disagree")
		}
		r.NativeRevision = sourceRevision
		r.NativeDirty = sourceModified == "true"
	}
	if r.NativeRevision == "" {
		r.Errors = append(r.Errors, "executable lacks compiled VCS source identity")
	}
	root := flags.Arg(0)
	var files []string
	if info, e := os.Stat(root); e != nil {
		r.Errors = append(r.Errors, e.Error())
	} else if info.IsDir() {
		files, e = eftest.DiscoverFixtures(root)
		if e != nil {
			r.Errors = append(r.Errors, e.Error())
		}
	} else {
		files = []string{root}
	}
	for _, file := range files {
		fixtures, e := eftest.LoadBlockchainFixtures(file)
		if e != nil {
			r.Errors = append(r.Errors, fmt.Sprintf("%s: %v", file, e))
			continue
		}
		names := make([]string, 0, len(fixtures))
		for name := range fixtures {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			fixture := fixtures[name]
			if fixture == nil || fixture.Network != *fork {
				r.Errors = append(r.Errors, fmt.Sprintf("%s/%s: network does not match selected %s", file, name, *fork))
				continue
			}
			relative, e := filepath.Rel(root, file)
			if e != nil || relative == "." {
				relative = filepath.Base(file)
			}
			c := caseResult{File: relative, Name: name, Fork: fixture.Network, BlockchainResult: fixture.RunNative()}
			if c.Passed {
				r.Passed++
			} else {
				r.Failed++
			}
			r.Cases = append(r.Cases, c)
			if len(r.Cases)%1000 == 0 {
				fmt.Fprintf(stderr, "native blockchain %s: %d passed, %d failed\n", *fork, r.Passed, r.Failed)
			}
		}
	}
	if len(r.Cases) == 0 {
		r.Errors = append(r.Errors, "no selected-network blockchain fixtures executed")
	}
	if e := json.NewEncoder(stdout).Encode(r); e != nil {
		fmt.Fprintln(stderr, e)
		return 1
	}
	if r.Failed > 0 || len(r.Errors) > 0 {
		return 1
	}
	return 0
}
