package chain

import (
	"encoding/json"
	"errors"

	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/types"
)

// BogotaSatisfactionVersion identifies the EIP-7805 last-transaction state
// predicate. Version 0 used final-block state; version 1 did not include ordinary
// state gas in Frame capacity. Both require rechecking their exact retained view.
const BogotaSatisfactionVersion uint8 = 2

var bogotaPayloadPrefix = []byte("engine/bogota/")

// Completed finalized verdicts use a separate namespace so later finalization
// passes only scan live views, rather than the entire execution history.
var bogotaVerdictPrefix = []byte("engine/bogota-verdict/")

// BogotaPayload records the driving CL's list view. Block holds an accepted
// block until execution has completed. An empty Block stores a known verdict.
type BogotaPayload struct {
	Number              uint64
	Block               []byte
	InclusionList       [][]byte
	Satisfied           bool
	SatisfactionVersion uint8
	compacted           bool
}

// HasInclusionListView distinguishes a retained (possibly empty) CL view from
// a compact verdict whose original transaction bytes have been released.
func (p *BogotaPayload) HasInclusionListView() bool { return p != nil && !p.compacted }

func bogotaPayloadKey(hash types.Hash) []byte {
	return append(append([]byte(nil), bogotaPayloadPrefix...), hash[:]...)
}

func (bc *Blockchain) ReadBogotaPayload(hash types.Hash) (*BogotaPayload, error) {
	if bc.db == nil {
		return nil, errors.New("Bogotá payload storage unavailable")
	}
	compacted := false
	data, err := bc.db.Get(bogotaPayloadKey(hash))
	if errors.Is(err, rawdb.ErrNotFound) {
		compacted = true
		key := append(append([]byte(nil), bogotaVerdictPrefix...), hash[:]...)
		data, err = bc.db.Get(key)
		if errors.Is(err, rawdb.ErrNotFound) {
			return nil, nil
		}
	}
	if err != nil {
		return nil, err
	}
	var record BogotaPayload
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, err
	}
	record.compacted = compacted
	return &record, nil
}

func (bc *Blockchain) WriteBogotaPayload(hash types.Hash, record *BogotaPayload) error {
	if bc.db == nil {
		return errors.New("Bogotá payload storage unavailable")
	}
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return bc.db.Put(bogotaPayloadKey(hash), data)
}

// PruneBogotaPayloads releases list bytes below the finalized checkpoint.
// Completed verdicts remain available for inherited Paris stale-head handling:
// Bogota FCU must still return the correct Boolean for a known VALID payload.
// Unexecuted branches below finality can no longer be selected and are removed.
func (bc *Blockchain) PruneBogotaPayloads(finalized uint64) error {
	if bc.db == nil {
		return nil
	}
	iter := bc.db.NewIterator(bogotaPayloadPrefix)
	defer iter.Release()
	batch := bc.db.NewBatch()
	for iter.Next() {
		var record BogotaPayload
		if err := json.Unmarshal(iter.Value(), &record); err != nil {
			return err
		}
		if record.Number >= finalized {
			continue
		}
		key := append([]byte(nil), iter.Key()...)
		if len(record.Block) == 0 {
			data, err := json.Marshal(struct {
				Number              uint64
				Satisfied           bool
				SatisfactionVersion uint8
			}{record.Number, record.Satisfied, record.SatisfactionVersion})
			if err != nil {
				return err
			}
			verdictKey := append(append([]byte(nil), bogotaVerdictPrefix...), key[len(bogotaPayloadPrefix):]...)
			if err := batch.Put(verdictKey, data); err != nil {
				return err
			}
		}
		if err := batch.Delete(key); err != nil {
			return err
		}
	}
	if err := rawdb.ReadError(iter); err != nil {
		return err
	}

	return batch.Write()
}
