package utils

import "sync"

type RWMap[K comparable, V any] struct {
	l sync.RWMutex
	m map[K]V
}

func NewRWMap[K comparable, V any]() *RWMap[K, V] {
	return &RWMap[K, V]{m: map[K]V{}}
}

func (m *RWMap[K, V]) Get(k K) V {
	m.l.RLock()
	val := m.m[k]
	m.l.RUnlock()
	return val
}

func (m *RWMap[K, V]) TryGet(k K) (V, bool) {
	m.l.RLock()
	val, ok := m.m[k]
	m.l.RUnlock()
	return val, ok
}

func (m *RWMap[K, V]) Set(k K, v V) {
	m.l.Lock()
	m.m[k] = v
	m.l.Unlock()
}

func (m *RWMap[K, V]) Update(k K, f func(v V) V) {
	m.l.Lock()
	m.m[k] = f(m.m[k])
	m.l.Unlock()
}

func (m *RWMap[K, V]) Copy() map[K]V {
	m.l.Lock()
	res := map[K]V{}
	for k, v := range m.m {
		res[k] = v
	}
	m.l.Unlock()
	return res
}
