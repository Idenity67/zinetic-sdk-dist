package credential

import (
	"sync"
	"testing"
)

func TestMemStore_StoreRetrieve(t *testing.T) {
	store := NewMemStore()
	defer store.ZeroizeAll()

	data := []byte("secret-credential-value")
	if err := store.Store("db_password", data); err != nil {
		t.Fatalf("store failed: %v", err)
	}

	retrieved, ok := store.Retrieve("db_password")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if string(retrieved) != "secret-credential-value" {
		t.Fatalf("unexpected value: %s", retrieved)
	}
}

func TestMemStore_RetrieveNotFound(t *testing.T) {
	store := NewMemStore()
	_, ok := store.Retrieve("nonexistent")
	if ok {
		t.Fatal("expected key not to exist")
	}
}

func TestMemStore_Zeroize(t *testing.T) {
	store := NewMemStore()

	data := []byte("sensitive")
	store.Store("key1", data)
	store.Zeroize("key1")

	_, ok := store.Retrieve("key1")
	if ok {
		t.Fatal("expected key to be removed after zeroize")
	}
}

func TestMemStore_ZeroizeAll(t *testing.T) {
	store := NewMemStore()

	store.Store("a", []byte("val-a"))
	store.Store("b", []byte("val-b"))
	store.Store("c", []byte("val-c"))

	store.ZeroizeAll()

	keys := store.Keys()
	if len(keys) != 0 {
		t.Fatalf("expected empty store, got %d keys", len(keys))
	}
}

func TestMemStore_Overwrite(t *testing.T) {
	store := NewMemStore()
	defer store.ZeroizeAll()

	store.Store("key", []byte("first"))
	store.Store("key", []byte("second"))

	val, ok := store.Retrieve("key")
	if !ok {
		t.Fatal("expected key to exist after overwrite")
	}
	if string(val) != "second" {
		t.Fatalf("expected 'second', got %q", val)
	}
}

func TestMemStore_IsolationFromSource(t *testing.T) {
	store := NewMemStore()
	defer store.ZeroizeAll()

	original := []byte("original-value")
	store.Store("key", original)

	original[0] = 'X'

	retrieved, _ := store.Retrieve("key")
	if retrieved[0] == 'X' {
		t.Fatal("store should copy data, not reference original")
	}
}

func TestMemStore_ConcurrentAccess(t *testing.T) {
	store := NewMemStore()
	defer store.ZeroizeAll()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := "key"
			store.Store(key, []byte("value"))
			store.Retrieve(key)
		}(i)
	}
	wg.Wait()
}

func TestMemStore_Keys(t *testing.T) {
	store := NewMemStore()
	defer store.ZeroizeAll()

	store.Store("alpha", []byte("a"))
	store.Store("beta", []byte("b"))

	keys := store.Keys()
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
}

func TestMemStore_EmptyKey(t *testing.T) {
	store := NewMemStore()
	err := store.Store("", []byte("value"))
	if err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestMemStore_Len(t *testing.T) {
	store := NewMemStore()
	defer store.ZeroizeAll()

	if store.Len() != 0 {
		t.Fatalf("expected len 0, got %d", store.Len())
	}

	store.Store("a", []byte("1"))
	store.Store("b", []byte("2"))

	if store.Len() != 2 {
		t.Fatalf("expected len 2, got %d", store.Len())
	}

	store.Zeroize("a")
	if store.Len() != 1 {
		t.Fatalf("expected len 1 after zeroize, got %d", store.Len())
	}
}

func TestMemStore_Has(t *testing.T) {
	store := NewMemStore()
	defer store.ZeroizeAll()

	if store.Has("key") {
		t.Fatal("expected Has to return false for missing key")
	}

	store.Store("key", []byte("val"))
	if !store.Has("key") {
		t.Fatal("expected Has to return true for existing key")
	}

	store.Zeroize("key")
	if store.Has("key") {
		t.Fatal("expected Has to return false after zeroize")
	}
}

func TestMemStore_ZeroizeNonExistentKey(t *testing.T) {
	store := NewMemStore()
	store.Zeroize("nonexistent")
}

func TestMemStore_LargeValue(t *testing.T) {
	store := NewMemStore()
	defer store.ZeroizeAll()

	large := make([]byte, 1024*1024)
	for i := range large {
		large[i] = byte(i % 256)
	}

	if err := store.Store("large", large); err != nil {
		t.Fatalf("store large value: %v", err)
	}

	retrieved, ok := store.Retrieve("large")
	if !ok {
		t.Fatal("expected large key to exist")
	}
	if len(retrieved) != len(large) {
		t.Fatalf("expected %d bytes, got %d", len(large), len(retrieved))
	}
	for i := range retrieved {
		if retrieved[i] != large[i] {
			t.Fatalf("mismatch at byte %d", i)
			break
		}
	}
}
