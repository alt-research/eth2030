package rawdb

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func legacyLayoutFixture(t *testing.T, dir string, values map[string]string, wal []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "data"), 0700); err != nil {
		t.Fatal(err)
	}
	for key, value := range values {
		if err := os.WriteFile(filepath.Join(dir, "data", hex.EncodeToString([]byte(key))), []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "wal"), wal, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestFileDBLegacyLayoutReadsAndTouchedKeysMigrateWithoutResurrection(t *testing.T) {
	dir := t.TempDir()
	legacyLayoutFixture(t, dir, map[string]string{"keep": "old", "change": "old", "delete": "old"}, nil)
	db, err := NewFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := db.Get([]byte("keep")); err != nil || string(got) != "old" {
		t.Fatalf("legacy read%q %v", got, err)
	}
	if _, err := os.Stat(FileDBValuePath(dir, []byte("keep"))); !os.IsNotExist(err) {
		t.Fatal("untouched legacy value was bulk rewritten")
	}
	b := db.NewBatch()
	b.Put([]byte("change"), []byte("new"))
	b.Put([]byte("delete"), []byte("intermediate"))
	b.Delete([]byte("delete"))
	if err := b.Write(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "data", hex.EncodeToString([]byte("change")))); !os.IsNotExist(err) {
		t.Fatal("touched legacy value retained a resurrectable alias")
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = NewFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if got, err := db.Get([]byte("change")); err != nil || string(got) != "new" {
		t.Fatalf("migrated read%q %v", got, err)
	}
	if _, err := db.Get([]byte("delete")); err != ErrNotFound {
		t.Fatal("deleted value resurrected")
	}
}

func TestFileDBUpgradeResumesJournalBeforeAndAfterAtomicGuard(t *testing.T) {
	for _, stage := range []string{"legacy", "linked", "guarded"} {
		t.Run(stage, func(t *testing.T) {
			dir := t.TempDir()
			wal := append(recoveryRecord(walPut, "head", "committed"), walCommit)
			legacyLayoutFixture(t, dir, map[string]string{"head": "old"}, wal)
			if stage != "legacy" {
				if err := os.Link(filepath.Join(dir, "wal"), FileDBJournalPath(dir)); err != nil {
					t.Fatal(err)
				}
			}
			if stage == "guarded" {
				if err := os.MkdirAll(filepath.Join(dir, fileDBShardDirectory), 0700); err != nil {
					t.Fatal(err)
				}
				tmp := filepath.Join(dir, "guard.tmp")
				if err := os.Symlink(fileDBShardDirectory, tmp); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(tmp, filepath.Join(dir, "wal")); err != nil {
					t.Fatal(err)
				}
			}
			db, err := NewFileDB(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if got, err := db.Get([]byte("head")); err != nil || string(got) != "committed" {
				t.Fatalf("recovery%q %v", got, err)
			}
			if target, err := os.Readlink(filepath.Join(dir, "wal")); err != nil || target != fileDBShardDirectory {
				t.Fatal("rollback guard absent")
			}
			if file, err := os.OpenFile(filepath.Join(dir, "wal"), os.O_RDWR|os.O_TRUNC, 0600); err == nil {
				file.Close()
				t.Fatal("old-format journal open/truncate accepted new format")
			}
		})
	}
}

func TestFileDBAmbiguousUpgradePreservesBothJournals(t *testing.T) {
	dir := t.TempDir()
	legacyLayoutFixture(t, dir, map[string]string{"keep": "old"}, []byte("old-journal"))
	newBytes := []byte("different-journal")
	if err := os.WriteFile(FileDBJournalPath(dir), newBytes, 0600); err != nil {
		t.Fatal(err)
	}
	if db, err := NewFileDB(dir); err == nil {
		db.Close()
		t.Fatal("ambiguous journals guessed")
	}
	for path, want := range map[string][]byte{filepath.Join(dir, "wal"): []byte("old-journal"), FileDBJournalPath(dir): newBytes} {
		got, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(got, want) {
			t.Fatal("failed upgrade altered journal")
		}
	}
	if err := os.Remove(FileDBJournalPath(dir)); err != nil {
		t.Fatal(err)
	}
	db, err := NewFileDB(dir)
	if err != nil {
		t.Fatalf("failed startup retained the exclusive lock: %v", err)
	}
	db.Close()
}

func TestFileDBLegacyAliasesDoNotResurrectAfterDelete(t *testing.T) {
	dir := t.TempDir()
	legacyLayoutFixture(t, dir, nil, nil)
	key := []byte{0xab}
	for name, value := range map[string]string{"ab": "selected", "AB": "inactive"} {
		if err := os.WriteFile(filepath.Join(dir, "data", name), []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	lower, err := os.Stat(filepath.Join(dir, "data", "ab"))
	if err != nil {
		t.Fatal(err)
	}
	upper, err := os.Stat(filepath.Join(dir, "data", "AB"))
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(lower, upper) {
		t.Skip("legacy hex alias fixture requires a case-sensitive filesystem")
	}
	db, err := NewFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := db.Get(key); err != nil || string(got) != "selected" {
		t.Fatalf("legacy alias selection%q %v", got, err)
	}
	if err := db.Delete(key); err != nil {
		t.Fatal(err)
	}
	db.Close()
	db, err = NewFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Get(key); err != ErrNotFound {
		t.Fatal("inactive legacy alias resurrected")
	}
	// The first reopen checkpoints the deletion WAL. Reopen again so a stale
	// physical alias cannot be hidden by that retained delete intent.
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = NewFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Get(key); err != ErrNotFound {
		t.Fatal("legacy alias resurrected after journal checkpoint")
	}
}

func TestFileDBShardsActualCommonPrefixAndPreservesReopenedValues(t *testing.T) {
	dir := t.TempDir()
	db, err := NewFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	b := db.NewBatch()
	const count = 4096
	key := func(i int) []byte { return binary.BigEndian.AppendUint64([]byte("mpt-history-node/"), uint64(i)) }
	for i := range count {
		b.Put(key(i), binary.BigEndian.AppendUint64(nil, uint64(i)))
	}
	if err := b.Write(); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	buckets, err := os.ReadDir(filepath.Join(dir, fileDBShardDirectory))
	if err != nil {
		t.Fatal(err)
	}
	if len(buckets) < 128 || len(buckets) > 256 {
		t.Fatalf("common-prefix concentration: %d buckets", len(buckets))
	}
	for _, bucket := range buckets {
		entries, err := os.ReadDir(filepath.Join(dir, fileDBShardDirectory, bucket.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) > count/8 {
			t.Fatal("one directory still concentrates common-prefix keys")
		}
	}
	db, err = NewFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for i := range count {
		got, err := db.Get(key(i))
		if err != nil || !bytes.Equal(got, binary.BigEndian.AppendUint64(nil, uint64(i))) {
			t.Fatalf("reopened value%d differs", i)
		}
	}
}

func TestFileDBRejectsUnguardedValuesAndWrongBucket(t *testing.T) {
	t.Run("unguarded", func(t *testing.T) {
		dir := t.TempDir()
		legacyLayoutFixture(t, dir, nil, nil)
		path := FileDBValuePath(dir, []byte("key"))
		os.MkdirAll(filepath.Dir(path), 0700)
		os.WriteFile(path, []byte("unproven"), 0600)
		if db, err := NewFileDB(dir); err == nil {
			db.Close()
			t.Fatal("unguarded sharded value accepted")
		}
	})
	t.Run("wrong-bucket", func(t *testing.T) {
		dir := t.TempDir()
		db, err := NewFileDB(dir)
		if err != nil {
			t.Fatal(err)
		}
		key := []byte("key")
		if err := db.Put(key, []byte("value")); err != nil {
			t.Fatal(err)
		}
		db.Close()
		path := FileDBValuePath(dir, key)
		bucket := "00"
		if filepath.Base(filepath.Dir(path)) == bucket {
			bucket = "01"
		}
		wrong := filepath.Join(dir, fileDBShardDirectory, bucket, filepath.Base(path))
		os.MkdirAll(filepath.Dir(wrong), 0700)
		if err := os.Rename(path, wrong); err != nil {
			t.Fatal(err)
		}
		if db, err := NewFileDB(dir); err == nil {
			db.Close()
			t.Fatal("wrong bucket silently hidden")
		}
	})
}

func TestFileDBRejectsAliasedShardRoot(t *testing.T) {
	dir := t.TempDir()
	other := t.TempDir()
	if err := os.Symlink(other, filepath.Join(dir, fileDBShardDirectory)); err != nil {
		t.Fatal(err)
	}
	if db, err := NewFileDB(dir); err == nil {
		db.Close()
		t.Fatal("distinct locks accepted an aliased value root")
	}
}
