package go_size_ttl_cache

import (
	"bytes"
	"encoding/gob"
	"time"
)

type cacheElem[TKey comparable, TValue any] struct {
	DestructionTime int64
	Value           TValue
	Key             TKey
}

func newCacheElem[TKey comparable, TValue any](
	ttl time.Duration,
	value TValue,
	key TKey) cacheElem[TKey, TValue] {

	var destructionTime int64
	if ttl == NoExpiration {
		destructionTime = 0
	} else {
		destructionTime = time.Now().UTC().Add(ttl).Unix()
	}

	return cacheElem[TKey, TValue]{
		DestructionTime: destructionTime,
		Value:           value,
		Key:             key,
	}
}

// isExpired вышло ли время жизини элемента.
func (e cacheElem[TKey, TValue]) isExpired() bool {
	if e.DestructionTime == 0 {
		return false
	}

	return time.Now().UTC().Unix() > e.DestructionTime
}

func (e cacheElem[TKey, TValue]) size() (int, error) {
	b := new(bytes.Buffer)
	if err := gob.NewEncoder(b).Encode(e); err != nil {
		return 0, err
	}
	return b.Len(), nil
}
