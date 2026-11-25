package eftest

// The acceptance baseline deliberately does not use the permissive legacy
// runners' exception/zero-root behavior or the ETH2030 geth wrapper.
import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	gethvm "github.com/ethereum/go-ethereum/core/vm"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
	gethtests "github.com/ethereum/go-ethereum/tests"
)

type BaselineFixture struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Cases  int    `json:"cases"`
	Source string `json:"source"`
}
type BaselineManifest struct {
	Version       int               `json:"version"`
	ExpectedCases int               `json:"expected_cases"`
	Fixtures      []BaselineFixture `json:"fixtures"`
}
type BaselineOutcome struct {
	Path          string `json:"path"`
	Name          string `json:"name"`
	Fork          string `json:"fork"`
	Index         int    `json:"index"`
	NativeRoot    string `json:"native_root"`
	ReferenceRoot string `json:"reference_root"`
	NativeLogs    string `json:"native_logs"`
	ReferenceLogs string `json:"reference_logs"`
	NativeGas     uint64 `json:"native_gas"`
	ReferenceGas  uint64 `json:"reference_gas"`
	Passed        bool   `json:"passed"`
	Error         string `json:"error,omitempty"`
}
type BaselineReport struct {
	Version          int               `json:"version"`
	NativeBackend    string            `json:"native_backend"`
	ReferenceBackend string            `json:"reference_backend"`
	Expected         int               `json:"expected"`
	Executed         int               `json:"executed"`
	Passed           int               `json:"passed"`
	Failed           int               `json:"failed"`
	Skipped          int               `json:"skipped"`
	Results          []BaselineOutcome `json:"results"`
}

// RunBaseline executes every locked case, or fails. No implicit exclusions,
// fixture discovery, missing-file skips, exception wildcards, or golden updates.
func RunBaseline(manifestPath string) (*BaselineReport, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	var m BaselineManifest
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err = dec.Decode(&m); err != nil {
		return nil, err
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("baseline manifest must contain exactly one JSON value")
	}
	if m.Version != 1 || m.ExpectedCases <= 0 || len(m.Fixtures) == 0 {
		return nil, fmt.Errorf("empty or unsupported baseline manifest")
	}
	report := &BaselineReport{Version: 1, NativeBackend: "native/core/execution.ApplyTransaction", ReferenceBackend: "upstream/go-ethereum@v1.17.10/tests.StateTest.RunNoVerify", Expected: m.ExpectedCases}
	seen := map[string]bool{}
	for _, f := range m.Fixtures {
		if f.Path == "" || filepath.IsAbs(f.Path) || filepath.Clean(f.Path) != f.Path || f.Path == ".." || len(f.Path) > 2 && f.Path[:3] == "../" || seen[f.Path] || f.Cases <= 0 || f.Source == "" {
			return report, fmt.Errorf("invalid/duplicate fixture: %q", f.Path)
		}
		seen[f.Path] = true
		raw, err := os.ReadFile(filepath.Join(filepath.Dir(manifestPath), f.Path))
		if err != nil {
			return report, err
		}
		sum := sha256.Sum256(raw)
		if hex.EncodeToString(sum[:]) != f.SHA256 {
			return report, fmt.Errorf("fixture digest mismatch: %s", f.Path)
		}
		native, reference, err := decodeBaseline(raw)
		if err != nil {
			return report, fmt.Errorf("%s: %w", f.Path, err)
		}
		names := make([]string, 0, len(native))
		for name := range native {
			names = append(names, name)
		}
		sort.Strings(names)
		count := 0
		for _, name := range names {
			n := native[name]
			forks := make([]string, 0, len(n.json.Post))
			for fork := range n.json.Post {
				forks = append(forks, fork)
			}
			sort.Strings(forks)
			for _, fork := range forks {
				for i := range n.json.Post[fork] {
					count++
					outcome := compareBaseline(f.Path, n, reference[name], StateSubtest{Fork: fork, Index: i})
					report.Results = append(report.Results, outcome)
					report.Executed++
					if outcome.Passed {
						report.Passed++
					} else {
						report.Failed++
					}
				}
			}
		}
		if count != f.Cases {
			return report, fmt.Errorf("fixture case count mismatch: %s: %d != %d", f.Path, count, f.Cases)
		}
	}
	if report.Executed != m.ExpectedCases {
		return report, fmt.Errorf("baseline case count mismatch: %d != %d", report.Executed, m.ExpectedCases)
	}
	if report.Failed != 0 {
		return report, fmt.Errorf("%d baseline cases failed", report.Failed)
	}
	return report, nil
}

func decodeBaseline(data []byte) (map[string]*StateTest, map[string]*gethtests.StateTest, error) {
	var raws map[string]json.RawMessage
	if err := json.Unmarshal(data, &raws); err != nil {
		return nil, nil, err
	}
	if len(raws) == 0 {
		return nil, nil, fmt.Errorf("empty fixture")
	}
	native := map[string]*StateTest{}
	reference := map[string]*gethtests.StateTest{}
	for name, raw := range raws {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil {
			return nil, nil, err
		}
		// Upstream provenance does not affect execution. All execution fields are strict.
		delete(fields, "_info")
		clean, err := json.Marshal(fields)
		if err != nil {
			return nil, nil, err
		}
		var j stJSON
		dec := json.NewDecoder(bytes.NewReader(clean))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&j); err != nil {
			return nil, nil, err
		}
		if len(j.Pre) == 0 || len(j.Post) == 0 {
			return nil, nil, fmt.Errorf("%s: missing pre/post state", name)
		}
		var ref gethtests.StateTest
		if err := json.Unmarshal(clean, &ref); err != nil {
			return nil, nil, fmt.Errorf("%s: invalid typed fixture: %w", name, err)
		}
		for fork, posts := range j.Post {
			if !ForkSupported(fork) || len(posts) == 0 {
				return nil, nil, fmt.Errorf("unsupported/empty fork %s", fork)
			}
			for _, post := range posts {
				if post.ExpectException != "" {
					return nil, nil, fmt.Errorf("exception classification is outside baseline v1; refusing wildcard pass")
				}
				for _, h := range []string{post.Hash, post.Logs} {
					b, err := hex.DecodeString(trimHex(h))
					if err != nil || len(b) != 32 || bytes.Equal(b, make([]byte, 32)) {
						return nil, nil, fmt.Errorf("missing/invalid root or logs oracle")
					}
				}
				x := post.Indexes
				if x.Data < 0 || x.Gas < 0 || x.Value < 0 || x.Data >= len(j.Tx.Data) || x.Gas >= len(j.Tx.GasLimit) || x.Value >= len(j.Tx.Value) {
					return nil, nil, fmt.Errorf("invalid transaction indexes")
				}
			}
		}
		native[name] = &StateTest{Name: name, json: j}
		reference[name] = &ref
	}
	return native, reference, nil
}
func trimHex(s string) string {
	if len(s) >= 2 && s[:2] == "0x" {
		return s[2:]
	}
	return s
}

func compareBaseline(path string, n *StateTest, ref *gethtests.StateTest, sub StateSubtest) (out BaselineOutcome) {
	out = BaselineOutcome{Path: path, Name: n.Name, Fork: sub.Fork, Index: sub.Index}
	defer func() {
		if p := recover(); p != nil {
			out.Passed = false
			out.Error = fmt.Sprintf("baseline panic: %v", p)
		}
	}()
	r := n.Run(sub)
	out.NativeRoot = r.GotRoot.Hex()
	out.NativeLogs = r.GotLogs.Hex()
	out.NativeGas = r.GasUsed
	if r.ExecutionError != nil {
		out.Error = fmt.Sprintf("native execution: %v", r.ExecutionError)
		return
	}
	st, root, gas, err := ref.RunNoVerify(gethtests.StateSubtest{Fork: sub.Fork, Index: sub.Index}, gethvm.Config{}, false, "hash")
	defer st.Close()
	if err != nil {
		out.Error = fmt.Sprintf("upstream execution: %v", err)
		return
	}
	logs, err := gethrlp.EncodeToBytes(st.StateDB.Logs())
	if err != nil {
		out.Error = err.Error()
		return
	}
	out.ReferenceRoot = root.Hex()
	out.ReferenceLogs = gethcrypto.Keccak256Hash(logs).Hex()
	out.ReferenceGas = gas
	if !r.Passed || r.Error != nil {
		out.Error = fmt.Sprintf("native oracle: %v", r.Error)
		return
	}
	if out.NativeRoot != out.ReferenceRoot || out.NativeLogs != out.ReferenceLogs || out.NativeGas != out.ReferenceGas {
		out.Error = "native/reference root, logs, or gas mismatch"
		return
	}
	out.Passed = true
	return
}
