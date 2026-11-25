package rawdb

import (
	"container/list"
)

const fileDBValueCacheBytes = 16 * 1024 * 1024

type fileCacheEntry struct {
	key   string
	value []byte
	cost  int
}
type fileValueCache struct {
	entries map[string]*list.Element
	order   list.List
	bytes   int
}

func (c *fileValueCache) get(key string, touch bool) ([]byte, bool) {
	e, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	if touch {
		c.order.MoveToFront(e)
	}
	return e.Value.(*fileCacheEntry).value, true
}
func (c *fileValueCache) remove(key string) {
	if e, ok := c.entries[key]; ok {
		c.bytes -= e.Value.(*fileCacheEntry).cost
		c.order.Remove(e)
		delete(c.entries, key)
	}
}
func (c *fileValueCache) put(key string, value []byte) {
	c.remove(key)
	cost := len(key) + len(value) + 64
	if cost > fileDBValueCacheBytes {
		return
	}
	if c.entries == nil {
		c.entries = make(map[string]*list.Element)
	}
	for c.bytes+cost > fileDBValueCacheBytes {
		c.remove(c.order.Back().Value.(*fileCacheEntry).key)
	}
	clone := make([]byte, len(value))
	copy(clone, value)
	entry := &fileCacheEntry{key, clone, cost}
	c.entries[key] = c.order.PushFront(entry)
	c.bytes += cost
}
