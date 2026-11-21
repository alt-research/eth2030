package main

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

func profileCall(args ...string) error {
	oldFlags, oldArgs := flag.CommandLine, os.Args
	defer func() { flag.CommandLine = oldFlags; os.Args = oldArgs }()
	flag.CommandLine = flag.NewFlagSet("profile-test", flag.ContinueOnError)
	os.Args = append([]string{"filedb-profile"}, args...)
	return run()
}
func TestProfileOwnershipAndAllValueVerification(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "owned")
	if err := profileCall("--mode=prepare", "--dir="+dir, "--keys=12", "--value-bytes=64", "--wal-generations=3"); err != nil {
		t.Fatal(err)
	}
	if err := profileCall("--mode=prepare", "--dir="+dir); err == nil {
		t.Fatal("preparation overwrote an existing directory")
	}
	if err := profileCall("--mode=open", "--dir="+dir); err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(dir, "data", fmtKey(1))
	body, err := os.ReadFile(data)
	if err != nil {
		t.Fatal(err)
	}
	body[len(body)-1] ^= 1
	if err := os.WriteFile(data, body, 0600); err != nil {
		t.Fatal(err)
	}
	if err := profileCall("--mode=open", "--dir="+dir); err == nil {
		t.Fatal("corrupt stored value passed verification")
	}
	if err := profileCall("--mode=open", "--dir="+t.TempDir()); err == nil {
		t.Fatal("unmarked directory accepted")
	}
}
func fmtKey(i int) string {
	const digits = "0123456789abcdef"
	k := key(i)
	out := make([]byte, len(k)*2)
	for j, v := range k {
		out[j*2] = digits[v>>4]
		out[j*2+1] = digits[v&15]
	}
	return string(out)
}
