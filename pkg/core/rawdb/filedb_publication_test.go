package rawdb

import (
	"bytes"
	"errors"
	"os"
	"sync"
	"testing"
	"time"
)

type heldPublication struct {
	sync.Mutex
	entered, release chan struct{}
}

func (h *heldPublication) Lock() { close(h.entered); <-h.release; h.Mutex.Lock() }
func TestFileDBCommittedCacheRemainsReadableDuringPublication(t *testing.T) {
	db, err := NewFileDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	key := []byte("hb")
	if err := db.Put(key, []byte("old")); err != nil {
		t.Fatal(err)
	}
	gate := &heldPublication{entered: make(chan struct{}), release: make(chan struct{})}
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(gate.release) }) }
	defer release()
	batch := db.NewBatch()
	batch.Put(key, []byte("new"))
	batch.Put([]byte("cold-new"), []byte("new"))
	done := make(chan error, 1)
	published := false
	go func() { done <- WriteBatchAndPublish(batch, gate, func() { published = true }) }()
	<-gate.entered
	read := make(chan error, 1)
	go func() {
		value, err := db.Get(key)
		if err == nil && !bytes.Equal(value, []byte("old")) {
			err = errors.New("unpublished cache value exposed")
		}
		if err == nil {
			err = db.Error()
		}
		read <- err
	}()
	select {
	case err := <-read:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(300 * time.Millisecond):
		release()
		<-done
		t.Fatal("cached committed reader waits behind durable writer")
	}
	cold := make(chan []byte, 1)
	go func() { v, _ := db.Get([]byte("cold-new")); cold <- v }()
	select {
	case <-cold:
		t.Fatal("cold unpublished value became visible")
	default:
	}
	release()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if !published {
		t.Fatal("memory publication skipped")
	}
	if v := <-cold; string(v) != "new" {
		t.Fatal("cold read did not join committed publication")
	}
	if v, err := db.Get(key); err != nil || string(v) != "new" {
		t.Fatal("committed cache missing", err)
	}
}
func TestFileDBPreparedFailureFencesWarmCacheAndDoesNotPublish(t *testing.T) {
	db, err := NewFileDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	key := []byte("warm")
	if err := db.Put(key, []byte("old")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(db.keyPath(key)+".tmp", 0700); err != nil {
		t.Fatal(err)
	}
	batch := db.NewBatch()
	batch.Put(key, []byte("new"))
	published := false
	var lock sync.Mutex
	if err := WriteBatchAndPublish(batch, &lock, func() { published = true }); err == nil {
		t.Fatal("real materialization failure was hidden")
	}
	if published {
		t.Fatal("failed selection published")
	}
	if _, err := db.Get(key); err == nil {
		t.Fatal("cached old value hid persistent failure")
	}
	if db.Error() == nil {
		t.Fatal("failure fence missing")
	}
}
func TestFileDBCloseFencesCommittedCache(t *testing.T) {
	db, err := NewFileDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	db.Put([]byte("k"), []byte("v"))
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Get([]byte("k")); err == nil {
		t.Fatal("cache outlived database close")
	}
}

func TestFileDBEmptyPublicationDoesNotAppendWAL(t *testing.T) {
	db, err := NewFileDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	before, err := db.walFile.Stat()
	if err != nil {
		t.Fatal(err)
	}
	var lock sync.Mutex
	published := false
	if err := WriteBatchAndPublish(db.NewBatch(), &lock, func() { published = true }); err != nil {
		t.Fatal(err)
	}
	after, err := db.walFile.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if !published || before.Size() != after.Size() {
		t.Fatal("empty publication skipped memory callback or appended WAL")
	}
}
