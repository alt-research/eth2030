package rawdb

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"testing"
)

func TestFileDBBoundedValueCacheAndSnapshotSemantics(t *testing.T) {
	dir := t.TempDir()
	db, err := NewFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	value := bytes.Repeat([]byte{7}, 1<<20)
	for i := 0; i < 24; i++ {
		if err := db.Put([]byte(fmt.Sprintf("key/%02d", i)), value); err != nil {
			t.Fatal(err)
		}
	}
	if db.cache.bytes > fileDBValueCacheBytes || len(db.cache.entries) >= 24 {
		t.Fatal("value cache grows with the full dataset")
	}
	got, err := db.Get([]byte("key/00"))
	if err != nil || !bytes.Equal(got, value) {
		t.Fatal("eviction lost persisted value", err)
	}
	got[0] ^= 1
	again, _ := db.Get([]byte("key/00"))
	if !bytes.Equal(again, value) {
		t.Fatal("returned buffer mutates the cache")
	}
	iter := db.NewIterator([]byte("key/00"))
	defer iter.Release()
	batch := db.NewBatch()
	batch.Put([]byte("key/00"), []byte{1})
	batch.Delete([]byte("key/00"))
	batch.Put([]byte("key/00"), []byte{9})
	if err := batch.Write(); err != nil {
		t.Fatal(err)
	}
	if !iter.Next() || !bytes.Equal(iter.Value(), value) || IteratorError(iter) != nil {
		t.Fatal("iterator snapshot changed after atomic write")
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = NewFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if db.cache.bytes != 0 {
		t.Fatal("opening retained stored values")
	}
	got, err = db.Get([]byte("key/00"))
	if err != nil || !bytes.Equal(got, []byte{9}) {
		t.Fatal(got, err)
	}
	if db.Len() != 24 {
		t.Fatal("metadata cardinality changed")
	}
}
func TestFileDBColdReadFailureCannotBecomeAbsence(t *testing.T) {
	dir := t.TempDir()
	db, err := NewFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	db.Put([]byte("known"), []byte{1})
	db.Close()
	db, err = NewFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := os.Remove(db.keyPath([]byte("known"))); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Get([]byte("known")); err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("lost committed value treated as ordinary absence: %v", err)
	}
	if err := db.Put([]byte("new"), []byte{1}); err == nil {
		t.Fatal("read failure did not fence mutation")
	}
	if IteratorError(db.NewIterator(nil)) == nil {
		t.Fatal("fenced iterator reports a successful empty snapshot")
	}
}
func TestFileDBIteratorReadErrorAndTableForwarding(t *testing.T) {
	dir := t.TempDir()
	db, _ := NewFileDB(dir)
	db.Put([]byte("table/known"), []byte{1})
	db.Close()
	db, err := NewFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := os.Remove(db.keyPath([]byte("table/known"))); err != nil {
		t.Fatal(err)
	}
	iter := db.NewIterator([]byte("table/"))
	defer iter.Release()
	if iter.Next() || IteratorError(iter) == nil {
		t.Fatal("partial/error snapshot accepted")
	}
	wrapped := &tableIterator{inner: iter, tablePrefix: []byte("table/")}
	if ReadError(wrapped) == nil {
		t.Fatal("table hides iterator failure")
	}
}
