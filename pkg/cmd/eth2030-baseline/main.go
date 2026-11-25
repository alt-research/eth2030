// eth2030-baseline runs the locked native/upstream differential acceptance set.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/eth2030/eth2030/core/eftest"
	"os"
)

func main() {
	manifest := flag.String("manifest", "core/eftest/testdata/hegota/manifest.json", "locked fixture manifest")
	flag.Parse()
	report, err := eftest.RunBaseline(*manifest)
	if report != nil {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if e := enc.Encode(report); e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(1)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
