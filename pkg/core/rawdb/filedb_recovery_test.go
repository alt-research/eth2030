package rawdb

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func recoveryRecord(op byte, key, value string) []byte {
	data := []byte{op}
	data = binary.BigEndian.AppendUint32(data, uint32(len(key)))
	data = append(data, key...)
	data = binary.BigEndian.AppendUint32(data, uint32(len(value)))
	return append(data, value...)
}

func TestFileDBRecoveryCoalescesCommittedIntentsAndPreservesMaterializedFiles(t *testing.T) {
	dir := t.TempDir()
	db, err := NewFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Put([]byte("head"), []byte("final")); err != nil {
		t.Fatal(err)
	}
	path := db.keyPath([]byte("head"))
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	var wal []byte
	for _, value := range []string{"old", "intermediate", "final"} {
		wal = append(wal, recoveryRecord(walPut, "head", value)...)
		wal = append(wal, walCommit)
	}
	if err := os.WriteFile(FileDBJournalPath(dir), wal, 0600); err != nil {
		t.Fatal(err)
	}
	// No intermediate rewrite is needed, including when a stale temporary
	// directory prevents writes. It must not prevent exact completed recovery.
	if err := os.MkdirAll(path+".tmp", 0700); err != nil {
		t.Fatal(err)
	}
	db, err = NewFileDB(dir)
	if err != nil {
		t.Fatalf("rewrote an already materialized committed head: %v", err)
	}
	defer db.Close()
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) {
		t.Fatal("recovery replaced the exact original data file")
	}
	if value, err := db.Get([]byte("head")); err != nil || string(value) != "final" {
		t.Fatalf("head=%q %v", value, err)
	}
	if data, err := os.ReadFile(FileDBJournalPath(dir)); err != nil || len(data) != 0 {
		t.Fatalf("successful recovery did not checkpoint WAL: %v", err)
	}
}

func TestFileDBRecoveryRepairsOnlyFinalCommittedValuesAndDeletes(t *testing.T) {
	dir := t.TempDir()
	db, err := NewFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Put([]byte("repair"), []byte("corrupt")); err != nil {
		t.Fatal(err)
	}
	deletedPath := db.keyPath([]byte("deleted"))
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	// The final deletion never needs to materialize its intermediate put.
	if err := os.MkdirAll(deletedPath+".tmp", 0700); err != nil {
		t.Fatal(err)
	}
	var wal []byte
	for _, rec := range [][]byte{
		recoveryRecord(walPut, "repair", "original"),
		recoveryRecord(walPut, "deleted", "obsolete"),
		recoveryRecord(walPut, "missing", "restored"),
	} {
		wal = append(wal, rec...)
	}
	wal = append(wal, walCommit)
	wal = append(wal, recoveryRecord(walDelete, "deleted", "")...)
	wal = append(wal, walCommit)
	wal = append(wal, recoveryRecord(walPut, "repair", "uncommitted")...)
	if err := os.WriteFile(FileDBJournalPath(dir), wal, 0600); err != nil {
		t.Fatal(err)
	}
	db, err = NewFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for key, value := range map[string]string{"repair": "original", "missing": "restored"} {
		if actual, err := db.Get([]byte(key)); err != nil || string(actual) != value {
			t.Fatalf("%s=%q %v", key, actual, err)
		}
	}
	if found, err := db.Has([]byte("deleted")); err != nil || found {
		t.Fatalf("final deletion lost: %t %v", found, err)
	}
	if _, err := os.Stat(deletedPath); !os.IsNotExist(err) {
		t.Fatal("deleted intermediate value was materialized")
	}
}

func TestFileDBRecoveryRetainsCompletePrefixAtEveryTornByte(t *testing.T) {
	transactions := [][]byte{
		append(append(recoveryRecord(walPut, "a", "first"), recoveryRecord(walPut, "b", "second")...), walCommit),
		append(append(recoveryRecord(walDelete, "a", ""), recoveryRecord(walPut, "b", "replacement")...), walCommit),
		append(recoveryRecord(walPut, "a", "last"), walCommit),
	}
	var whole []byte
	for _, tx := range transactions {
		whole = append(whole, tx...)
	}
	for cut := 0; cut <= len(whole); cut++ {
		t.Run(fmt.Sprintf("cut=%d", cut), func(t *testing.T) {
			dir := t.TempDir()
			if err := os.Mkdir(filepath.Join(dir, "data"), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "wal"), whole[:cut], 0600); err != nil {
				t.Fatal(err)
			}
			db, err := NewFileDB(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			expected := map[string]string{}
			if cut >= len(transactions[0]) {
				expected["a"], expected["b"] = "first", "second"
			}
			if cut >= len(transactions[0])+len(transactions[1]) {
				delete(expected, "a")
				expected["b"] = "replacement"
			}
			if cut == len(whole) {
				expected["a"] = "last"
			}
			for _, key := range []string{"a", "b"} {
				value, err := db.Get([]byte(key))
				want, exists := expected[key]
				if exists && (err != nil || !bytes.Equal(value, []byte(want))) {
					t.Fatalf("committed prefix %s=%q want %q: %v", key, value, want, err)
				}
				if !exists && err != ErrNotFound {
					t.Fatalf("uncommitted %s leaked: %q %v", key, value, err)
				}
			}
		})
	}
}

func TestFileDBRecoveryUnknownSuffixPreservesCommittedPrefix(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "data"), 0700); err != nil {
		t.Fatal(err)
	}
	wal := append(recoveryRecord(walPut, "committed", "original"), walCommit, 0xff)
	wal = append(wal, recoveryRecord(walPut, "other", "must-not-appear")...)
	wal = append(wal, walCommit)
	if err := os.WriteFile(filepath.Join(dir, "wal"), wal, 0600); err != nil {
		t.Fatal(err)
	}
	db, err := NewFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if got, err := db.Get([]byte("committed")); err != nil || string(got) != "original" {
		t.Fatalf("committed prefix lost: %q %v", got, err)
	}
	if _, err := db.Get([]byte("other")); err != ErrNotFound {
		t.Fatal("resynchronized through an unknown record")
	}
	if value, err := os.ReadFile(FileDBValuePath(dir, []byte("committed"))); err != nil || string(value) != "original" {
		t.Fatal("committed value missing from recovered disk")
	}
}

func TestFileDBRecoveryFailureRetainsJournalForAtomicRetry(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	if err := os.Mkdir(dataDir, 0700); err != nil {
		t.Fatal(err)
	}
	var wal []byte
	for _, key := range []string{"a", "b"} {
		wal = append(wal, recoveryRecord(walPut, key, "committed")...)
	}
	wal = append(wal, walCommit)
	journal := filepath.Join(dir, "wal")
	if err := os.WriteFile(journal, wal, 0600); err != nil {
		t.Fatal(err)
	}
	blocker := FileDBValuePath(dir, []byte("b")) + ".tmp"
	if err := os.MkdirAll(blocker, 0700); err != nil {
		t.Fatal(err)
	}
	if db, err := NewFileDB(dir); err == nil {
		db.Close()
		t.Fatal("recovery accepted an incomplete committed batch")
	}
	retained, err := os.ReadFile(FileDBJournalPath(dir))
	if err != nil || !bytes.Equal(retained, wal) {
		t.Fatalf("failed recovery discarded its durable commit: %v", err)
	}
	if err := os.Remove(blocker); err != nil {
		t.Fatal(err)
	}
	db, err := NewFileDB(dir)
	if err != nil {
		t.Fatalf("exclusive lock or committed replay did not recover: %v", err)
	}
	defer db.Close()
	for _, key := range []string{"a", "b"} {
		if got, err := db.Get([]byte(key)); err != nil || string(got) != "committed" {
			t.Fatalf("retry lost %s: %q %v", key, got, err)
		}
	}
}
