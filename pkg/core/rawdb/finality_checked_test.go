package rawdb

import (
	"errors"
	"testing"
)

type finalityErrorReader struct{ err error }

func (r finalityErrorReader) Has([]byte) (bool, error)   { return false, r.err }
func (r finalityErrorReader) Get([]byte) ([]byte, error) { return nil, r.err }
func TestCheckedFinalizedHashKeepsAbsenceCorruptionAndReadFailureDistinct(t *testing.T) {
	db := NewMemoryDB()
	if h, err := ReadFinalizedBlockHashChecked(db); err != nil || h != ([32]byte{}) {
		t.Fatal("unrecorded finality changed")
	}
	want := [32]byte{31: 7}
	if err := WriteFinalizedBlockHash(db, want); err != nil {
		t.Fatal(err)
	}
	if h, err := ReadFinalizedBlockHashChecked(db); err != nil || h != want {
		t.Fatalf("stored finality changed: %v", err)
	}
	if err := db.Put(headFinalizedKey, []byte{7}); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadFinalizedBlockHashChecked(db); err == nil {
		t.Fatal("malformed finality became zero")
	}
	failure := errors.New("read unavailable")
	if _, err := ReadFinalizedBlockHashChecked(finalityErrorReader{failure}); !errors.Is(err, failure) {
		t.Fatal("read error lost")
	}
}
