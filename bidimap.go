package ipcall

import "sync"

// BidiMap is a concurrency-safe, bidirectional map implementation (allowing
// lookups by either key or value). NOTE: a BidiMap should only be used when
// there exists a bijection (one-to-one mapping) between the keys.
//
// The current implementaion builds on Go's `map` primitive, maintaining two
// maps (one for key and the other for value lookup). Synchronization is
// performed using a `sync.RWMutex`.
type BidiMap struct {
	mu *sync.RWMutex
	kv map[string]string
	vk map[string]string
}

func NewBidiMap(initSize int) *BidiMap {
	if initSize < 0 {
		initSize = 0
	}
	return &BidiMap{
		kv: make(map[string]string, initSize),
		vk: make(map[string]string, initSize),
	}
}

func (m *BidiMap) Add(key, val string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.kv[key] = val
	m.kv[val] = key
}

func (m *BidiMap) GetByKey(key string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.kv[key]
	return v, ok
}

func (m *BidiMap) GetByVal(val string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	k, ok := m.vk[val]
	return k, ok
}

func (m *BidiMap) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if val, ok := m.kv[key]; ok {
		delete(m.kv, key)
		delete(m.vk, val)
	}
}
