package rawdb

import (
	"encoding/hex"
	"os"
	"path/filepath"
)

// FileDBValuePath returns the current format's value-write target. Existing
// legacy values may reside elsewhere; logical consumers must use Database.Get.
func FileDBValuePath(dir string, key []byte) string {
	return filepath.Join(dir, "data", hex.EncodeToString(key))
}

// FileDBJournalPath returns the current format's write journal path.
func FileDBJournalPath(dir string) string { return filepath.Join(dir, "wal") }

// materializeOp changes files only. Callers serialize writes with db.mu and
// publish the committed index/cache separately after all operations succeed.
func (db *FileDB) materializeOp(op fileBatchOp) error {
	if !op.delete {
		return db.writeDataFile(op.key, op.value)
	}
	path := db.keyPath(op.key)
	if name, ok := db.index[string(op.key)]; ok {
		path = filepath.Join(db.dataDir, name)
	}
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
