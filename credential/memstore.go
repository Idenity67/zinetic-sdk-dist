package credential

import (
	"fmt"
	"sync"
)

type MemStore struct {
	mu      sync.RWMutex
	entries map[string]*memEntry
}

type memEntry struct {
	data   []byte
	locked bool
}

func NewMemStore() *MemStore {
	return &MemStore{
		entries: make(map[string]*memEntry),
	}
}

func (m *MemStore) Store(key string, value []byte) error {
	if key == "" {
		return fmt.Errorf("key is required")
	}

	buf := make([]byte, len(value))
	copy(buf, value)

	locked := mlockBytes(buf)

	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.entries[key]; ok {
		zeroize(existing.data)
		if existing.locked {
			munlockBytes(existing.data)
		}
	}

	m.entries[key] = &memEntry{
		data:   buf,
		locked: locked,
	}
	return nil
}

func (m *MemStore) Retrieve(key string) ([]byte, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.entries[key]
	if !ok {
		return nil, false
	}

	out := make([]byte, len(entry.data))
	copy(out, entry.data)
	return out, true
}

func (m *MemStore) Zeroize(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.entries[key]
	if !ok {
		return
	}

	zeroize(entry.data)
	if entry.locked {
		munlockBytes(entry.data)
	}
	delete(m.entries, key)
}

func (m *MemStore) ZeroizeAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for key, entry := range m.entries {
		zeroize(entry.data)
		if entry.locked {
			munlockBytes(entry.data)
		}
		delete(m.entries, key)
	}
}

func (m *MemStore) Keys() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]string, 0, len(m.entries))
	for k := range m.entries {
		keys = append(keys, k)
	}
	return keys
}

func (m *MemStore) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.entries)
}

func (m *MemStore) Has(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.entries[key]
	return ok
}

func zeroize(buf []byte) {
	for i := range buf {
		buf[i] = 0
	}
}
