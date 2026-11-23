package rawdb

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"testing"
)

func TestLargeFileBatchRetainsLastIntentAfterReopen(t *testing.T) {
	dir := t.TempDir()
	db, err := NewFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	b := db.NewBatch()
	for i := range 128 {
		key := []byte(fmt.Sprintf("key-%03d", i))
		b.Put(key, []byte("old"))
		b.Delete(key)
		b.Put(key, []byte("last"))
		if i%2 == 0 {
			b.Delete(key)
		}
	}
	if err := b.Write(); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = NewFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for i := range 128 {
		got, err := db.Get([]byte(fmt.Sprintf("key-%03d", i)))
		if i%2 == 0 {
			if err != ErrNotFound {
				t.Fatalf("deleted%d resurrected: %q %v", i, got, err)
			}
		} else if err != nil || !bytes.Equal(got, []byte("last")) {
			t.Fatalf("last intent%d lost: %q %v", i, got, err)
		}
	}
}
func TestLargeFileBatchFailureNeverPublishesAndReplaysCompleteWAL(t *testing.T) {
	dir := t.TempDir()
	db, err := NewFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	fault := FileDBValuePath(dir, []byte("key-070")) + ".tmp"
	if err := os.MkdirAll(fault, 0700); err != nil {
		t.Fatal(err)
	}
	b := db.NewBatch()
	for i := range 128 {
		b.Put([]byte(fmt.Sprintf("key-%03d", i)), []byte("committed"))
	}
	var mu sync.Mutex
	published := false
	if err := WriteBatchAndPublish(b, &mu, func() { published = true }); err == nil {
		t.Fatal("actual materialization failure accepted")
	}
	if published {
		t.Fatal("failed durable batch published head")
	}
	if db.Error() == nil {
		t.Fatal("write failure did not fence store")
	}
	// Close/reopen immediately. No worker may survive failure return and race
	// recovery; committed WAL must restore keys that failed materialization too.
	_ = db.Close()
	if err := os.Remove(fault); err != nil {
		t.Fatal(err)
	}
	db, err = NewFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for i := range 128 {
		got, err := db.Get([]byte(fmt.Sprintf("key-%03d", i)))
		if err != nil || !bytes.Equal(got, []byte("committed")) {
			t.Fatalf("committed intent%d missing after recovery: %q %v", i, got, err)
		}
	}
}
