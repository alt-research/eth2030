package rawdb

import "sync"

// WriteBatchAndPublish makes durable storage and its in-memory selection visible
// under one publication lock. A persistent batch may perform slow I/O before
// taking that lock, while retaining its old committed read view until publish.
// The callback must only publish in-memory state; it must not access storage.
func WriteBatchAndPublish(batch Batch, lock sync.Locker, publish func()) error {
	if prepared, ok := batch.(interface {
		WriteAndPublish(sync.Locker, func()) error
	}); ok {
		return prepared.WriteAndPublish(lock, publish)
	}
	lock.Lock()
	defer lock.Unlock()
	if err := batch.Write(); err != nil {
		return err
	}
	publish()
	return nil
}
