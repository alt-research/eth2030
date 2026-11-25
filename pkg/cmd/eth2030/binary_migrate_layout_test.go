package main

import (
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBinaryMigrationRejectsSharedShardedValuesBeforeOpeningStores(t *testing.T) {
	source, dest := filepath.Join(t.TempDir(), "source"), filepath.Join(t.TempDir(), "dest")
	db, err := rawdb.NewFileDB(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Put([]byte("sentinel"), []byte("unchanged")); err != nil {
		t.Fatal(err)
	}
	db.Close()
	if err := os.MkdirAll(dest, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(source, "data-v2"), filepath.Join(dest, "data-v2")); err != nil {
		t.Fatal(err)
	}
	if _, err := migrateBinaryFiles(source, dest, types.Hash{}, 1, 1); err == nil || !strings.Contains(err.Error(), "separate databases") {
		t.Fatalf("shared current layout not rejected early: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(dest, "wal")); !os.IsNotExist(err) {
		t.Fatal("failed validation initialized destination")
	}
	db, err = rawdb.NewFileDB(source)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if got, err := db.Get([]byte("sentinel")); err != nil || string(got) != "unchanged" {
		t.Fatal("failed migration changed source values")
	}
}
