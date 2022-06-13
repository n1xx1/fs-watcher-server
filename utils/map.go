package utils

import (
	"encoding/json"
	"github.com/labstack/gommon/log"
	"sync"
)

type RWMap[K comparable, V any] struct {
	mLock sync.RWMutex
	m     map[K]V

	reprLock    sync.RWMutex
	repr        string
	reprVersion int
}

func NewRWMap[K comparable, V any]() *RWMap[K, V] {
	return &RWMap[K, V]{m: map[K]V{}}
}

func (m *RWMap[K, V]) GetRepr() string {
	m.reprLock.Lock()
	defer m.reprLock.Unlock()
	return m.repr
}

func (m *RWMap[K, V]) Get(k K) V {
	m.mLock.RLock()
	val := m.m[k]
	m.mLock.RUnlock()
	return val
}

func (m *RWMap[K, V]) TryGet(k K) (V, bool) {
	m.mLock.RLock()
	val, ok := m.m[k]
	m.mLock.RUnlock()
	return val, ok
}

func (m *RWMap[K, V]) Set(k K, v V) {
	m.mLock.Lock()
	m.m[k] = v
	m.updateRepr()
	m.mLock.Unlock()
}

func (m *RWMap[K, V]) Delete(k K) {
	m.mLock.Lock()
	delete(m.m, k)
	m.updateRepr()
	m.mLock.Unlock()
}

func (m *RWMap[K, V]) Update(k K, f func(v V) V) {
	m.mLock.Lock()
	m.m[k] = f(m.m[k])
	m.updateRepr()
	m.mLock.Unlock()
}

func (m *RWMap[K, V]) updateRepr() {
	m.reprLock.Lock()
	m.reprVersion++
	go func(rv int, m1 map[K]V) {
		data, err := json.Marshal(m1)
		if err != nil {
			log.Warnf("marshal error: %v", err)
		}
		m.reprLock.Lock()
		if rv == m.reprVersion {
			m.repr = string(data)
		}
		m.reprLock.Unlock()
	}(m.reprVersion, copyMap(m.m))
	m.reprLock.Unlock()
}

func (m *RWMap[K, V]) Copy() map[K]V {
	m.mLock.Lock()
	res := copyMap(m.m)
	m.mLock.Unlock()
	return res
}

func copyMap[K comparable, V any](m map[K]V) map[K]V {
	res := map[K]V{}
	for k, v := range m {
		res[k] = v
	}
	return res
}
