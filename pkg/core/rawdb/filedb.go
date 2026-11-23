// FileDB implements a persistent key-value store with a durable WAL. New
// values use 256 directories selected by a hash of the full key, avoiding flat
// directory limits even when application keys share a long common prefix.
// Legacy data/ files remain readable and migrate only when touched.
//
// Layout: LOCK, wal-v2, wal -> data-v2/ (old-reader guard), data/ (legacy),
// data-v2/<bucket>/<hex-key>. The index stores membership for new keys without
// retaining a duplicate full hex filename. Values remain bounded and cold.
package rawdb

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"syscall"
)

// WAL record types.
const (
	walPut    byte = 0x01
	walDelete byte = 0x02
	walCommit byte = 0x03
)

// FileDB is a file-based persistent key-value store implementing the Database
// interface. It is safe for concurrent use from multiple goroutines within
// a single process. Cross-process safety is provided by a file lock.
type FileDB struct {
	mu      sync.RWMutex
	cacheMu sync.Mutex        // committed cache/fence only; never held across disk I/O
	dir     string            // root directory
	dataDir string            // dir + "/data"
	index   map[string]string // startup key/filename metadata only; values are bounded
	cache   fileValueCache
	walFile *os.File // write-ahead log file handle
	lockFd  int      // file descriptor for LOCK file
	failure error    // a persistent I/O failure requires reopen/recovery
	closed  bool
}

// NewFileDB opens or creates a file-based key-value database at dir.
// The directory is created if it does not exist. An exclusive file lock
// is acquired to prevent concurrent access from other processes.
func NewFileDB(dir string) (*FileDB, error) {
	// Create directory structure.
	dataDir := filepath.Join(dir, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("filedb: mkdir: %w", err)
	}

	// Acquire exclusive file lock.
	lockPath := filepath.Join(dir, "LOCK")
	lockFd, err := acquireLock(lockPath)
	if err != nil {
		return nil, fmt.Errorf("filedb: lock: %w", err)
	}

	db := &FileDB{
		index:   make(map[string]string),
		dir:     dir,
		dataDir: dataDir,
		lockFd:  lockFd,
	}

	if err := prepareFileDBLayout(dir); err != nil {
		releaseLock(lockFd)
		return nil, fmt.Errorf("filedb: prepare layout: %w", err)
	}

	// Load key/filename metadata in bounded directory batches; values remain cold.
	if err := db.loadIndex(); err != nil {
		releaseLock(lockFd)
		return nil, fmt.Errorf("filedb: load index: %w", err)
	}

	// Replay WAL to recover any uncommitted operations, then rebuild.
	if err := db.replayWAL(); err != nil {
		releaseLock(lockFd)
		return nil, fmt.Errorf("filedb: replay wal: %w", err)
	}

	// Sync every touched layout directory before discarding recovery intents.
	if err := db.syncDataDirectories(); err != nil {
		releaseLock(lockFd)
		return nil, err
	}

	// Open WAL for new writes (truncate after successful replay).
	walPath := FileDBJournalPath(dir)
	walFile, err := os.OpenFile(walPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o644)
	if err != nil {
		releaseLock(lockFd)
		return nil, fmt.Errorf("filedb: open wal: %w", err)
	}
	if err := syncDir(dir); err != nil {
		walFile.Close()
		releaseLock(lockFd)
		return nil, fmt.Errorf("filedb: sync WAL directory: %w", err)
	}
	db.walFile = walFile

	return db, nil
}

// Has returns true if the key exists in the database.
func (db *FileDB) Has(key []byte) (bool, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	if err := db.stateError(); err != nil {
		return false, err
	}
	_, ok := db.index[string(key)]
	return ok, nil
}

// Get retrieves the value for a key. Returns ErrNotFound if the key does not exist.
func (db *FileDB) Get(key []byte) ([]byte, error) {
	if value, ok, err := db.cachedRead(string(key)); ok || err != nil {
		return value, err
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	if value, ok, err := db.cachedRead(string(key)); ok || err != nil {
		return value, err
	}
	k := string(key)
	name, exists := db.index[k]
	if !exists {
		return nil, ErrNotFound
	}
	value, err := os.ReadFile(db.indexedPath(key, name))
	if err != nil {
		return nil, db.fenceRead(fmt.Errorf("filedb: reopen required after indexed read failure: %w", err))
	}
	db.cacheMu.Lock()
	db.cache.put(k, value)
	db.cacheMu.Unlock()
	return bytes.Clone(value), nil
}

// cachedRead serves only the last fully committed cache view. Writers retain
// the storage mutex during fsync but do not mutate this view until publication.
func (db *FileDB) cachedRead(key string) ([]byte, bool, error) {
	db.cacheMu.Lock()
	defer db.cacheMu.Unlock()
	if err := db.stateErrorLocked(); err != nil {
		return nil, false, err
	}
	value, ok := db.cache.get(key, true)
	if !ok {
		return nil, false, nil
	}
	return bytes.Clone(value), true, nil
}
func (db *FileDB) stateErrorLocked() error {
	if db.closed {
		return errors.New("filedb: database closed")
	}
	return db.failure
}
func (db *FileDB) stateError() error {
	db.cacheMu.Lock()
	defer db.cacheMu.Unlock()
	return db.stateErrorLocked()
}
func (db *FileDB) fenceRead(err error) error {
	db.cacheMu.Lock()
	defer db.cacheMu.Unlock()
	db.failure = err
	return err
}

// Put stores a key-value pair. The write is persisted to the WAL and data
// directory before returning.
func (db *FileDB) Put(key, value []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if err := db.stateError(); err != nil {
		return err
	}
	return db.putLocked(key, value)
}

// putLocked performs Put while the caller already holds db.mu.
func (db *FileDB) putLocked(key, value []byte) error {
	return db.applyOpsLocked([]fileBatchOp{{key: key, value: value}})
}

// Delete removes a key. Returns nil even if the key does not exist.
func (db *FileDB) Delete(key []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if err := db.stateError(); err != nil {
		return err
	}
	return db.deleteLocked(key)
}

// deleteLocked performs Delete while the caller already holds db.mu.
func (db *FileDB) deleteLocked(key []byte) error {
	return db.applyOpsLocked([]fileBatchOp{{key: key, delete: true}})
}

// Close closes the database, syncs the WAL, and releases the file lock.
func (db *FileDB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.cacheMu.Lock()
	if db.closed {
		db.cacheMu.Unlock()
		return nil
	}
	db.closed = true
	db.cacheMu.Unlock()
	var errs []error
	if db.walFile != nil {
		if err := db.walFile.Sync(); err != nil {
			errs = append(errs, err)
		}
		if err := db.walFile.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	releaseLock(db.lockFd)
	db.cacheMu.Lock()
	db.cache = fileValueCache{}
	db.cacheMu.Unlock()
	db.index = nil
	if len(errs) > 0 {
		return fmt.Errorf("filedb: close: %v", errs)
	}
	return nil
}

// NewBatch creates a new batch writer for atomic batch operations.
func (db *FileDB) NewBatch() Batch {
	return &fileBatch{db: db}
}

// NewIterator returns an iterator over all keys with the given prefix,
// in ascending key order.
func (db *FileDB) NewIterator(prefix []byte) Iterator {
	db.mu.Lock()
	defer db.mu.Unlock()
	failed := func(err error) Iterator {
		failure := db.fenceRead(fmt.Errorf("filedb: reopen required after iterator read failure: %w", err))
		return &memIterator{pos: -1, err: failure}
	}
	if err := db.stateError(); err != nil {
		return &memIterator{pos: -1, err: err}
	}
	var keys []string
	for key := range db.index {
		if bytes.HasPrefix([]byte(key), prefix) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	items := make([]kv, 0, len(keys))
	for _, key := range keys {
		db.cacheMu.Lock()
		value, cached := db.cache.get(key, false)
		db.cacheMu.Unlock()
		if !cached {
			var err error
			value, err = os.ReadFile(db.indexedPath([]byte(key), db.index[key]))
			if err != nil {
				return failed(fmt.Errorf("filedb: iterator read: %w", err))
			}
		}
		clone := make([]byte, len(value))
		copy(clone, value)
		items = append(items, kv{key: []byte(key), value: clone})
	}
	return &memIterator{items: items, pos: -1}
}

// Len returns the number of entries in the database.
func (db *FileDB) Len() int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return len(db.index)
}

// --- WAL operations ---

// walWrite appends a put or delete record to the WAL.
// Format: [type:1][keyLen:4][key:N][valLen:4][val:M]
// For delete, valLen is 0 and val is omitted.
func (db *FileDB) walWrite(op byte, key, value []byte) error {
	var buf []byte
	buf = append(buf, op)

	// Key length + key.
	kl := make([]byte, 4)
	binary.BigEndian.PutUint32(kl, uint32(len(key)))
	buf = append(buf, kl...)
	buf = append(buf, key...)

	// Value length + value.
	vl := make([]byte, 4)
	binary.BigEndian.PutUint32(vl, uint32(len(value)))
	buf = append(buf, vl...)
	buf = append(buf, value...)

	if _, err := db.walFile.Write(buf); err != nil {
		return fmt.Errorf("filedb: wal write: %w", err)
	}
	return nil
}

// walWriteCommit writes a commit marker to the WAL and syncs.
func (db *FileDB) walWriteCommit() error {
	if _, err := db.walFile.Write([]byte{walCommit}); err != nil {
		return fmt.Errorf("filedb: wal commit write: %w", err)
	}
	if err := db.walFile.Sync(); err != nil {
		return fmt.Errorf("filedb: wal sync: %w", err)
	}
	return nil
}

// replayWAL reads the WAL file and replays committed operations.
// Uncommitted operations (those without a trailing commit marker) are
// discarded. After replay, the data files and index are consistent.
func (db *FileDB) replayWAL() error {
	f, err := os.Open(FileDBJournalPath(db.dir))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	size := info.Size()
	// Retain last committed intent offsets, never the whole WAL/value history.
	type intent struct {
		op     byte
		offset int64
		size   uint32
	}
	committed := make(map[string]intent)
	pending := make(map[string]intent)
	var pos int64
	read := func(offset int64, dst []byte) error { _, err := f.ReadAt(dst, offset); return err }
parse:
	for pos < size {
		var tag [1]byte
		if err := read(pos, tag[:]); err != nil {
			return err
		}
		pos++
		switch tag[0] {
		case walCommit:
			for key, record := range pending {
				committed[key] = record
			}
			pending = make(map[string]intent)
		case walPut, walDelete:
			var number [4]byte
			if size-pos < 4 {
				break parse
			}
			if err := read(pos, number[:]); err != nil {
				return err
			}
			pos += 4
			keySize := int64(binary.BigEndian.Uint32(number[:]))
			if keySize > size-pos {
				break parse
			}
			key := make([]byte, int(keySize))
			if err := read(pos, key); err != nil {
				return err
			}
			pos += keySize
			if size-pos < 4 {
				break parse
			}
			if err := read(pos, number[:]); err != nil {
				return err
			}
			pos += 4
			valueSize := binary.BigEndian.Uint32(number[:])
			if int64(valueSize) > size-pos {
				break parse
			}
			pending[string(key)] = intent{tag[0], pos, valueSize}
			pos += int64(valueSize)
		default:
			break parse
		}
	}
	keys := make([]string, 0, len(committed))
	for key := range committed {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		record := committed[key]
		path := db.keyPath([]byte(key))
		if name, ok := db.index[key]; ok {
			path = db.indexedPath([]byte(key), name)
		}
		if record.op == walDelete {
			if err := db.removeDataFiles([]byte(key)); err != nil {
				return err
			}
			delete(db.index, key)
			continue
		}

		value := make([]byte, int(record.size))
		if len(value) > 0 {
			if err := read(record.offset, value); err != nil {
				return err
			}
		}
		if existing, err := os.ReadFile(path); err == nil && bytes.Equal(existing, value) {
			continue
		}
		if err := db.writeDataFile([]byte(key), value); err != nil {
			return err
		}
		db.index[key] = ""
	}

	return nil
}

// --- Data file operations ---

// keyPath returns the filesystem path for a key's data file.
func (db *FileDB) keyPath(key []byte) string {
	return FileDBValuePath(db.dir, key)
}

// writeDataFile atomically writes a value to the data file for key.
// It writes to a temporary file first, then renames for atomicity.
func (db *FileDB) writeDataFile(key, value []byte) error {
	path := db.keyPath(key)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmpPath := path + ".tmp"

	file, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("filedb: write tmp: %w", err)
	}
	_, err = file.Write(value)
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("filedb: flush tmp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath) // Best-effort cleanup.
		return fmt.Errorf("filedb: rename: %w", err)
	}
	return db.removeLegacyValue(key)
}

// --- File locking ---

// acquireLock opens (or creates) the lock file and acquires an exclusive
// flock on it. Returns the file descriptor.
func acquireLock(path string) (int, error) {
	fd, err := syscall.Open(path, syscall.O_CREAT|syscall.O_RDWR, 0o644)
	if err != nil {
		return -1, fmt.Errorf("open lock file: %w", err)
	}
	if err := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		syscall.Close(fd)
		return -1, fmt.Errorf("acquire flock: %w", err)
	}
	return fd, nil
}

// releaseLock releases the file lock and closes the file descriptor.
func releaseLock(fd int) {
	syscall.Flock(fd, syscall.LOCK_UN)
	syscall.Close(fd)
}

// --- Batch ---

// fileBatch buffers Put and Delete operations and applies them atomically.
type fileBatch struct {
	db   *FileDB
	ops  []fileBatchOp
	size int
}

type fileBatchOp struct {
	key    []byte
	value  []byte
	delete bool
}

func (b *fileBatch) Put(key, value []byte) error {
	b.ops = append(b.ops, fileBatchOp{
		key:   append([]byte{}, key...),
		value: append([]byte{}, value...),
	})
	b.size += len(key) + len(value)
	return nil
}

func (b *fileBatch) Delete(key []byte) error {
	b.ops = append(b.ops, fileBatchOp{
		key:    append([]byte{}, key...),
		delete: true,
	})
	b.size += len(key)
	return nil
}

func (b *fileBatch) ValueSize() int { return b.size }

// Write applies all buffered operations atomically. All WAL records for the
// batch are written before the commit marker, so either all operations in
// the batch are applied or none are (in case of a crash mid-write).
func (b *fileBatch) Write() error {
	b.db.mu.Lock()
	defer b.db.mu.Unlock()
	return b.db.applyOpsLocked(b.ops)
}

// Commit the WAL before materializing data files. Otherwise a crash after the
// first rename but before the commit marker leaves uncommitted file contents
// that replay cannot undo. Any write error fences this DB until recovery.
func (db *FileDB) applyOpsLocked(ops []fileBatchOp) error {
	if err := db.prepareOpsLocked(ops); err != nil {
		return err
	}
	db.publishOpsLocked(ops)
	return nil
}

// WriteAndPublish retains the old committed cache while durable I/O runs.
// Cold reads wait on mu and must not retain the caller's publication lock.
func (b *fileBatch) WriteAndPublish(lock sync.Locker, publish func()) error {
	b.db.mu.Lock()
	defer b.db.mu.Unlock()
	if err := b.db.prepareOpsLocked(b.ops); err != nil {
		return err
	}
	lock.Lock()
	defer lock.Unlock()
	b.db.publishOpsLocked(b.ops)
	publish()
	return nil
}

func (db *FileDB) prepareOpsLocked(ops []fileBatchOp) (err error) {
	if len(ops) == 0 {
		return db.stateError()
	}
	if err := db.stateError(); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			db.fenceRead(fmt.Errorf("filedb: reopen required after write failure: %w", err))
		}
	}()
	for _, op := range ops {
		kind := walPut
		if op.delete {
			kind = walDelete
		}
		if err = db.walWrite(kind, op.key, op.value); err != nil {
			return err
		}
	}
	if err = db.walWriteCommit(); err != nil {
		return err
	}
	for _, op := range ops {
		if err = db.materializeOp(op); err != nil {
			return err
		}
	}
	return nil
}

func (db *FileDB) publishOpsLocked(ops []fileBatchOp) {
	db.cacheMu.Lock()
	defer db.cacheMu.Unlock()
	for _, op := range ops {
		if op.delete {
			delete(db.index, string(op.key))
			db.cache.remove(string(op.key))
		} else {
			db.index[string(op.key)] = ""
			db.cache.put(string(op.key), op.value)
		}
	}

}

func (b *fileBatch) Reset() {
	b.ops = b.ops[:0]
	b.size = 0
}

// Compile-time interface checks.
var _ Database = (*FileDB)(nil)

func syncDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(dir.Sync(), dir.Close())
}

// Error reports a fenced or closed store to lazy state/proof consumers.
func (db *FileDB) Error() error { return db.stateError() }
