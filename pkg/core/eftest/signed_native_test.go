package eftest

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/eth2030/eth2030/core/types"
	"path/filepath"
	"strings"
	"testing"
)

func signedOfficialFixture(t *testing.T, kind int) *StateTest {
	t.Helper()
	tests, err := LoadStateTests(filepath.Join("testdata", "eest-v5.4.0", "envelope-"+string(rune('0'+kind))+".json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(tests) != 1 {
		t.Fatalf("expected one complete official fixture, got %d", len(tests))
	}
	for _, st := range tests {
		return st
	}
	panic("unreachable")
}
func TestOfficialOsakaSignedEnvelopesThroughNativeRunner(t *testing.T) {
	for kind := 0; kind <= 4; kind++ {
		st := signedOfficialFixture(t, kind)
		r := st.Run(StateSubtest{Fork: "Osaka"})
		if !r.Passed || r.Error != nil {
			t.Fatalf("native type %d: %+v", kind, r)
		}
		raw, err := hex.DecodeString(st.json.Post["Osaka"][0].TxBytes[2:])
		if err != nil {
			t.Fatal(err)
		}
		tx, err := types.DecodeTxRLP(raw)
		if err != nil || tx.Type() != uint8(kind) {
			t.Fatalf("fixture envelope type %d: %v", kind, err)
		}
	}
}
func TestSignedFixtureFailuresNeverBecomeExpectedExceptionPasses(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*StateTest)
	}{
		{"missing_signed_bytes", func(st *StateTest) {
			st.json.Post["Osaka"][0].TxBytes = ""
			st.json.Post["Osaka"][0].ExpectException = "TransactionException.GAS_LIMIT_EXCEEDS_MAXIMUM"
		}},
		{"malformed_signed_bytes", func(st *StateTest) {
			st.json.Post["Osaka"][0].TxBytes = "0xzz"
			st.json.Post["Osaka"][0].ExpectException = "TransactionException.INTRINSIC_GAS_TOO_LOW"
		}},
		{"absent_expected_root", func(st *StateTest) { st.json.Post["Osaka"][0].Hash = "0x" + strings.Repeat("0", 64) }},
		{"wrong_expected_root", func(st *StateTest) { st.json.Post["Osaka"][0].Hash = "0x" + strings.Repeat("1", 64) }},
		{"wrong_logs_root", func(st *StateTest) { st.json.Post["Osaka"][0].Logs = "0x" + strings.Repeat("1", 64) }},
		{"unknown_exception", func(st *StateTest) { st.json.Post["Osaka"][0].ExpectException = "TransactionException.ANY_ERROR" }},
		{"negative_selector", func(st *StateTest) { st.json.Post["Osaka"][0].Indexes.Data = -1 }},
		{"malformed_prestate", func(st *StateTest) {
			for a, v := range st.json.Pre {
				v.Balance = "0xzz"
				st.json.Pre[a] = v
				break
			}
		}},
		{"missing_base_fee", func(st *StateTest) { st.json.Env.CurrentBaseFee = "" }},
		{"missing_randomness", func(st *StateTest) { st.json.Env.CurrentRandom = "" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := signedOfficialFixture(t, 2)
			tc.mutate(st)
			r := st.RunSigned(StateSubtest{Fork: "Osaka"})
			if r.Passed || r.Error == nil {
				t.Fatalf("bad fixture passed: %+v", r)
			}
		})
	}
	st := signedOfficialFixture(t, 2)
	if r := st.RunSigned(StateSubtest{Fork: "Osaka", Index: -1}); r.Passed || r.Error == nil {
		t.Fatal("negative subtest passed")
	}
	// Unrelated errors cannot masquerade as a gas-cap rejection.
	post := st.json.Post["Osaka"][0]
	raw, _ := hex.DecodeString(post.TxBytes[2:])
	tx, err := types.DecodeTxRLP(raw)
	if err != nil {
		t.Fatal(err)
	}
	cfg := ForkConfig("Osaka")
	if fixtureExceptionMatches("TransactionException.GAS_LIMIT_EXCEEDS_MAXIMUM", errors.New("disk read failure"), tx, cfg, st.buildHeader(cfg)) {
		t.Fatal("unrelated failure counted as expected rejection")
	}
}
func TestCanonicalNativeFixtureForkContexts(t *testing.T) {
	osaka, amsterdam := ForkConfig("Osaka"), ForkConfig("Amsterdam")
	if osaka == nil || amsterdam == nil || !osaka.IsOsaka(0) || osaka.IsAmsterdam(0) || !amsterdam.IsAmsterdam(0) || !amsterdam.IsOsaka(0) || amsterdam.IsGlamsterdan(0) {
		t.Fatal("incoherent native fixture fork context")
	}
}

func TestOfficialSlotNumberAndAmsterdamBlobContext(t *testing.T) {
	var env stEnv
	if err := json.Unmarshal([]byte(`{"slotNumber":"0xc0ffee"}`), &env); err != nil {
		t.Fatal(err)
	}
	st := &StateTest{json: stJSON{Env: env}}
	header := st.buildHeader(ForkConfig("Amsterdam"))
	if header.SlotNumber == nil || *header.SlotNumber != 0xc0ffee {
		t.Fatal("official slotNumber lost at native fixture boundary")
	}
	osaka, amsterdam := ForkConfig("Osaka").BlobConfigAt(0), ForkConfig("Amsterdam").BlobConfigAt(0)
	if osaka.Target != 6 || osaka.Max != 9 || amsterdam.Target != 14 || amsterdam.Max != 21 {
		t.Fatalf("fixture BPO contexts osaka=%+v amsterdam=%+v", osaka, amsterdam)
	}
}
