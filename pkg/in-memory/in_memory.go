package inmemory

import "sync/atomic"

// Store is a generic in-memory key-value store for rarely changing whole datasets.
type Store[K comparable, V any] struct {
	v atomic.Value // map[K]V
	// mu sync.Mutex
}

func NewStore[K comparable, V any]() *Store[K, V] {
	s := &Store[K, V]{}
	s.v.Store(make(map[K]V))
	return s
}

func (s *Store[K, V]) Get(k K) (V, bool) {
	m := s.v.Load().(map[K]V)
	v, ok := m[k]
	return v, ok
}

func (s *Store[K, V]) UpsertMany(patch map[K]V) {
	old := s.v.Load().(map[K]V)

	cp := make(map[K]V, len(old)+len(patch))
	for k, v := range old {
		cp[k] = v
	}
	for k, v := range patch {
		cp[k] = v
	}

	s.v.Store(cp)
}

func (s *Store[K, V]) ReplaceAll(m map[K]V) {
	// s.mu.Lock()
	// defer s.mu.Unlock()

	copy := make(map[K]V, len(m))
	for k, v := range m {
		copy[k] = v
	}
	s.v.Store(copy)
}
