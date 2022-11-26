package go_size_ttl_cache

import (
	"bytes"
	"encoding/gob"
	"sync"
	"time"
)

type cacheElem[TKey comparable, TValue any] struct {
	DestructionTime time.Time
	Value           TValue
	Key             TKey
	mx              sync.Mutex
}

func newCacheElem[TKey comparable, TValue any](
	ttl time.Duration,
	value TValue,
	key TKey) cacheElem[TKey, TValue] {

	return cacheElem[TKey, TValue]{
		DestructionTime: time.Now().UTC().Add(ttl),
		Value:           value,
		Key:             key,
	}
}

// isExpired вышло ли время жизини элемента.
func (e cacheElem[TKey, TValue]) isExpired() bool {
	return time.Now().UTC().Unix() > e.DestructionTime.Unix()
}

func (e cacheElem[TKey, TValue]) update(ttl time.Duration, value TValue) {
	e.Value = value
	e.DestructionTime = time.Now().UTC().Add(ttl)
}

func (e cacheElem[TKey, TValue]) size() (int, error) {
	e.mx.Lock()
	defer e.mx.Unlock()
	b := new(bytes.Buffer)
	if err := gob.NewEncoder(b).Encode(e); err != nil {
		return 0, err
	}
	return b.Len(), nil
}
