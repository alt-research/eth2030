package rawdb

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const fileDBShardDirectory = "data-v2"

// walkFileDBDirectory bounds directory-entry memory even for legacy stores.
func walkFileDBDirectory(path string, visit func(os.DirEntry) error) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	for {
		entries, err := f.ReadDir(128)
		for _, entry := range entries {
			if e := visit(entry); e != nil {
				return e
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// prepareFileDBLayout installs a durable rollback guard before any v2 values
// can be materialized. Hardlinking the existing journal, then atomically
// replacing only its old name with a directory symlink, leaves no missing-WAL
// crash interval. Old readers fail opening that directory rather than silently
// ignoring new values and resetting their canonical head.
func prepareFileDBLayout(dir string) error {
	shards := filepath.Join(dir, fileDBShardDirectory)
	if info, err := os.Lstat(shards); err == nil && !info.IsDir() {
		return errors.New("filedb: shard root must be a directory, not an alias")
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(shards, 0755); err != nil {
		return err
	}
	legacy, journal := filepath.Join(dir, "wal"), FileDBJournalPath(dir)
	old, err := os.Lstat(legacy)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil && old.Mode()&os.ModeSymlink != 0 {
		target, e := os.Readlink(legacy)
		if e != nil {
			return e
		}
		if target != fileDBShardDirectory {
			return fmt.Errorf("filedb: unsupported journal guard %q", target)
		}
		current, e := os.Lstat(journal)
		if e != nil {
			return e
		}
		if !current.Mode().IsRegular() {
			return errors.New("filedb: guarded journal is not a regular file")
		}
		return syncDir(dir)
	}
	if err == nil && !old.Mode().IsRegular() {
		return errors.New("filedb: legacy journal is not a regular file")
	}
	// A correct interrupted upgrade has no sharded values before its guard.
	// Do not guess authority if a manual/partial format rewrite violated that.
	err = walkFileDBDirectory(shards, func(bucket os.DirEntry) error {
		if !bucket.IsDir() {
			return fmt.Errorf("filedb: unexpected sharded entry %q", bucket.Name())
		}
		return walkFileDBDirectory(filepath.Join(shards, bucket.Name()), func(entry os.DirEntry) error {
			if !entry.IsDir() && !strings.HasSuffix(entry.Name(), ".tmp") {
				return errors.New("filedb: sharded values without rollback guard")
			}
			return nil
		})
	})
	if err != nil {
		return err
	}
	current, err := os.Lstat(journal)
	switch {
	case err == nil:
		if !current.Mode().IsRegular() {
			return errors.New("filedb: new journal is not a regular file")
		}
		if old != nil {
			if !os.SameFile(old, current) {
				return errors.New("filedb: ambiguous old/new journals")
			}
		} else if current.Size() != 0 {
			return errors.New("filedb: unguarded journal has data")
		}
	case os.IsNotExist(err):
		if old != nil {
			if err = os.Link(legacy, journal); err != nil {
				return err
			}
		} else {
			f, e := os.OpenFile(journal, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0644)
			if e != nil {
				return e
			}
			if e = errors.Join(f.Sync(), f.Close()); e != nil {
				return e
			}
		}
	default:
		return err
	}
	f, err := os.OpenFile(journal, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	if err = errors.Join(f.Sync(), f.Close()); err != nil {
		return err
	}
	if err = syncDir(dir); err != nil {
		return err
	}
	temporary := filepath.Join(dir, "wal-guard.tmp")
	if _, err = os.Lstat(temporary); os.IsNotExist(err) {
		err = os.Symlink(fileDBShardDirectory, temporary)
	} else if err == nil {
		target, e := os.Readlink(temporary)
		if e != nil || target != fileDBShardDirectory {
			return errors.New("filedb: unexpected journal guard staging entry")
		}
	}
	if err != nil {
		return err
	}
	if err = os.Rename(temporary, legacy); err != nil {
		return err
	}
	return syncDir(dir)
}

func (db *FileDB) indexedPath(key []byte, name string) string {
	if name == "" {
		return db.keyPath(key)
	}
	return filepath.Join(db.dataDir, name)
}

func (db *FileDB) removeLegacyValue(key []byte) error {
	if name, ok := db.index[string(key)]; ok && name != "" {
		if err := os.Remove(filepath.Join(db.dataDir, name)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
func (db *FileDB) removeDataFiles(key []byte) error {
	if err := os.Remove(db.keyPath(key)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return db.removeLegacyValue(key)
}

func (db *FileDB) loadIndex() error {
	if err := walkFileDBDirectory(db.dataDir, func(entry os.DirEntry) error {
		if entry.IsDir() {
			return nil
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".tmp") {
			_ = os.Remove(filepath.Join(db.dataDir, name))
			return nil
		}
		key, err := hex.DecodeString(name)
		if err != nil {
			return nil
		}
		k := string(key)
		if prior, ok := db.index[k]; ok {
			// Preserve the legacy lexical alias selection. Remove only inactive
			// representations, so a later delete cannot resurrect another spelling.
			if name > prior {
				if err := os.Remove(filepath.Join(db.dataDir, prior)); err != nil {
					return err
				}
			} else {
				return os.Remove(filepath.Join(db.dataDir, name))
			}
		}
		db.index[k] = name
		return nil
	}); err != nil {
		return err
	}
	return walkFileDBDirectory(filepath.Join(db.dir, fileDBShardDirectory), func(bucket os.DirEntry) error {
		name := bucket.Name()
		decoded, err := hex.DecodeString(name)
		if err != nil || len(decoded) != 1 || hex.EncodeToString(decoded) != name || !bucket.IsDir() {
			return fmt.Errorf("filedb: invalid shard %q", name)
		}
		directory := filepath.Join(db.dir, fileDBShardDirectory, name)
		return walkFileDBDirectory(directory, func(entry os.DirEntry) error {
			if entry.IsDir() {
				return nil
			}
			name := entry.Name()
			path := filepath.Join(directory, name)
			if strings.HasSuffix(name, ".tmp") {
				_ = os.Remove(path)
				return nil
			}
			key, err := hex.DecodeString(name)
			if err != nil {
				return nil
			}
			if path != db.keyPath(key) {
				return fmt.Errorf("filedb: value in wrong shard: %s", path)
			}
			if err := db.removeLegacyValue(key); err != nil {
				return err
			}
			// New entries derive their filename from the key, avoiding a duplicate
			// full hex filename per key in the resident metadata index.
			db.index[string(key)] = ""
			return nil
		})
	})
}

func (db *FileDB) syncDataDirectories() error {
	if err := syncDir(db.dataDir); err != nil {
		return err
	}
	root := filepath.Join(db.dir, fileDBShardDirectory)
	if err := walkFileDBDirectory(root, func(entry os.DirEntry) error {
		if !entry.IsDir() {
			return fmt.Errorf("filedb: non-directory shard %q", entry.Name())
		}
		return syncDir(filepath.Join(root, entry.Name()))
	}); err != nil {
		return err
	}
	return syncDir(root)
}
