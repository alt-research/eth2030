package rawdb

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
)

// FileDBValuePath returns the current format's value-write target. Existing
// legacy values may reside elsewhere; logical consumers must use Database.Get.
func FileDBValuePath(dir string, key []byte) string {
	bucket := sha256.Sum256(key)
	return filepath.Join(dir, fileDBShardDirectory, hex.EncodeToString(bucket[:1]), hex.EncodeToString(key))
}

// FileDBJournalPath returns the current format's write journal path.
func FileDBJournalPath(dir string) string { return filepath.Join(dir, "wal-v2") }

// materializeOp changes files only. Callers serialize writes with db.mu and
// publish the committed index/cache separately after all operations succeed.
func (db *FileDB) materializeOp(op fileBatchOp) error {
	if !op.delete {
		return db.writeDataFile(op.key, op.value)
	}
	return db.removeDataFiles(op.key)
}
