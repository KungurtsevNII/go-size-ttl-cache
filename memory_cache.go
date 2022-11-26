package go_size_ttl_cache

import (
	"bytes"
	"encoding/gob"
	"sync"
	"time"
)

type memoryCache[TKey comparable, TValue any] struct {
	sync.RWMutex
	capBytes      int
	elems         map[TKey]cacheElem[TKey, TValue]
	closeCh       chan struct{}
	cleanCh       chan TKey
	checkerTicker *time.Ticker
	isClose       bool
}

func NewMemoryCache[TKey comparable, TValue any](capMB int) (SizedTTLCache[TKey, TValue], error) {
	if capMB <= 0 {
		return nil, ErrCapMBHasZeroOrLessValue
	}

	cache := &memoryCache[TKey, TValue]{
		capBytes:      capMB,
		elems:         make(map[TKey]cacheElem[TKey, TValue]),
		closeCh:       make(chan struct{}),
		cleanCh:       make(chan TKey, 10),
		checkerTicker: time.NewTicker(500 * time.Millisecond),
		isClose:       false,
	}

	go cache.cleaner()
	go cache.checker()

	return cache, nil
}

func (m *memoryCache[TKey, TValue]) Put(key TKey, value TValue, ttl time.Duration) error {
	newElem := newCacheElem[TKey, TValue](ttl, value, key)
	size, err := newElem.size()
	if err != nil {
		return err
	}

	m.RLock()
	if m.isClose {
		return ErrCacheClosed
	}

	freeSpace, err := m.internalFreeSpace()
	if err != nil {
		m.RUnlock()
		return err
	}
	canPut := freeSpace-size >= 0

	elem, ok := m.elems[key]
	if ok {
		m.RUnlock()
		elem.update(ttl, value)
		return nil
	}

	m.RUnlock()

	if !canPut {
		return ErrNotEnoughSpace
	}

	m.Lock()
	m.elems[key] = newElem
	m.Unlock()
	return nil
}

func (m *memoryCache[TKey, TValue]) Get(key TKey) (value TValue, err error) {
	m.RLock()

	if m.isClose {
		err = ErrCacheClosed
		return
	}

	if elem, ok := m.elems[key]; ok {
		if !elem.isExpired() {
			value = elem.Value
			m.RUnlock()
			return
		}
		m.RUnlock()
		m.cleanCh <- elem.Key
		err = ErrElemNotFound
		return
	}
	m.RUnlock()
	err = ErrElemNotFound
	return
}

func (m *memoryCache[TKey, TValue]) Delete(key TKey) error {
	m.Lock()
	defer m.Unlock()

	if m.isClose {
		return ErrCacheClosed
	}

	if _, ok := m.elems[key]; ok {
		delete(m.elems, key)
		return nil
	}

	return ErrElemNotFound
}

func (m *memoryCache[TKey, TValue]) Exists(key TKey) (bool, error) {
	m.RLock()
	defer m.RUnlock()

	if m.isClose {
		return false, ErrCacheClosed
	}

	elem, ok := m.elems[key]
	if elem.isExpired() {
		m.cleanCh <- key
		return false, nil
	}

	return ok, nil
}

func (m *memoryCache[TKey, TValue]) internalFreeSpace() (int, error) {
	size, err := m.elemsSize()
	if err != nil {
		return 0, err
	}
	return m.capBytes - size, nil
}

func (m *memoryCache[TKey, TValue]) FreeSpace() (int, error) {
	m.RLock()
	defer m.RUnlock()

	if m.isClose {
		return 0, ErrCacheClosed
	}

	size, err := m.elemsSize()
	if err != nil {
		return 0, err
	}
	return m.capBytes - size, nil
}

func (m *memoryCache[TKey, TValue]) elemsSize() (int, error) {
	b := new(bytes.Buffer)
	if err := gob.NewEncoder(b).Encode(m.elems); err != nil {
		return 0, err
	}
	return b.Len(), nil
}

func (m *memoryCache[TKey, TValue]) Cap() (int, error) {
	m.RLock()
	defer m.RUnlock()
	if m.isClose {
		return 0, ErrCacheClosed
	}
	return m.capBytes, nil
}

func (m *memoryCache[TKey, TValue]) Count() (int, error) {
	m.RLock()
	defer m.RUnlock()

	if m.isClose {
		return 0, ErrCacheClosed
	}

	return len(m.elems), nil
}

func (m *memoryCache[TKey, TValue]) Close() {
	m.Lock()
	defer m.Unlock()

	close(m.closeCh)
	close(m.cleanCh)
	m.checkerTicker.Stop()

	m.isClose = true

	for _, v := range m.elems {
		delete(m.elems, v.Key)
	}
}

func (m *memoryCache[TKey, TValue]) cleaner() {
	for {
		select {
		case key, ok := <-m.cleanCh:
			if !ok {
				return
			}
			m.Lock()
			delete(m.elems, key)
			m.Unlock()
		case <-m.closeCh:
			return
		}
	}
}

func (m *memoryCache[TKey, TValue]) checker() {
	for {
		select {
		case <-m.checkerTicker.C:
			m.Lock()
			for _, v := range m.elems {
				if v.isExpired() {
					delete(m.elems, v.Key)
				}
			}
			m.Unlock()
		case <-m.closeCh:
			return
		}
	}
}
