package go_size_ttl_cache

import (
	"bytes"
	"encoding/gob"
	"sync"
	"time"
)

type memoryCache[TKey comparable, TValue any] struct {
	sync.RWMutex
	capBytes          int
	elems             map[TKey]cacheElem[TKey, TValue]
	defaultExpiration time.Duration
	closeCh           chan struct{}
	cleanCh           chan TKey
	checkerTicker     *time.Ticker
	isClose           bool
}

func NewMemoryCache[TKey comparable, TValue any](
	capMB int,
	GCDuration time.Duration,
	defaultExpiration time.Duration) (SizedTTLCache[TKey, TValue], error) {

	if capMB <= 0 {
		return nil, ErrCapMBHasZeroOrLessValue
	}

	if defaultExpiration <= 0 {
		return nil, ErrDefaultExpirationZeroOrLess
	}

	cache := &memoryCache[TKey, TValue]{
		capBytes:          capMB,
		elems:             make(map[TKey]cacheElem[TKey, TValue]),
		defaultExpiration: defaultExpiration,
		closeCh:           make(chan struct{}),
		cleanCh:           make(chan TKey, 10),
		checkerTicker:     time.NewTicker(GCDuration),
		isClose:           false,
	}

	go cache.cleaner()
	go cache.checker()

	return cache, nil
}

func (m *memoryCache[TKey, TValue]) Put(key TKey, value TValue, ttl time.Duration) error {
	if ttl == DefaultExpiration {
		ttl = m.defaultExpiration
	}

	newElem := newCacheElem[TKey, TValue](ttl, value, key)
	size, err := newElem.size()
	if err != nil {
		return err
	}

	m.RLock()

	if m.isClose {
		m.RUnlock()
		return ErrCacheClosed
	}

	freeSpace, err := m.internalFreeSpace()
	if err != nil {
		m.RUnlock()
		return err
	}
	canPut := freeSpace-size >= 0
	if _, ok := m.elems[key]; !ok && !canPut {
		m.RUnlock()
		return ErrNotEnoughSpace
	}
	m.RUnlock()

	m.Lock()
	m.elems[key] = newElem
	m.Unlock()
	return nil
}

func (m *memoryCache[TKey, TValue]) Get(key TKey) (value TValue, err error) {
	m.RLock()

	if m.isClose {
		m.RUnlock()
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

	if m.isClose {
		m.RUnlock()
		return false, ErrCacheClosed
	}

	elem, ok := m.elems[key]
	if elem.isExpired() {
		m.RUnlock()
		m.cleanCh <- key
		return false, nil
	}

	m.RUnlock()
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

	m.elems = make(map[TKey]cacheElem[TKey, TValue])
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
			toClear := m.expiredKeys()
			if len(toClear) == 0 {
				return
			}
			m.Lock()
			for _, k := range toClear {
				delete(m.elems, k)
			}
			m.Unlock()
		case <-m.closeCh:
			return
		}
	}
}

func (m *memoryCache[TKey, TValue]) expiredKeys() (keys []TKey) {
	m.RLock()
	defer m.RUnlock()
	for key, itm := range m.elems {
		if itm.isExpired() {
			keys = append(keys, key)
		}
	}
	return
}
