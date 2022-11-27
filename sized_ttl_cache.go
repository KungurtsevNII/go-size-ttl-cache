package go_size_ttl_cache

import (
	"errors"
	"time"
)

const (
	// DefaultGCDuration дефолтный период отчистки элементов, чей TTL закончился.
	DefaultGCDuration = time.Millisecond * 300

	// NoExpiration указывается при Put методе, чтобы элемент кэша НИКОГДА не был удален.
	NoExpiration time.Duration = -1

	// DefaultExpiration указывается при Put методе, чтобы использовать дефолтный TTL, указанный в экземпляре кэша.
	DefaultExpiration time.Duration = 0
)

var (
	ErrCapMBHasZeroOrLessValue     = errors.New("cache size in megabytes can't be less or equal to zero")
	ErrElemNotFound                = errors.New("can't find element with specified key")
	ErrNotEnoughSpace              = errors.New("can't add element - not enough space")
	ErrCacheClosed                 = errors.New("can't work with the cache after it is closed")
	ErrDefaultExpirationZeroOrLess = errors.New("can't create cache with default expiration equal to or less than 0")
)

// SizedTTLCache интерфейс для работы с кэшом.
// 1. Put добавляет новый элемент в кэш с определенным TTL. Если элемент уже есть, то увеличивает TTL на указанный и обновляет элемент.
// 2. Get получает элемент по ключу. Если элемент не найден, то возвращает ошибку - ErrElemNotFound.
// 3. Delete удаляет элемент по ключу. В случае успеха ошибка == nil. Так же может вернуть ошибку - ErrElemNotFound.
// 4. Exists возвращает true если элемент есть в кэше, false если нет.
// 5. FreeSpace возвращает свободное место в Bytes.
// 6. Cap вместимость кэша.
// 7. Count кол-во элементов.
// 8. Close отчистка ресурсов.
type SizedTTLCache[TKey comparable, TValue any] interface {
	Put(key TKey, value TValue, ttl time.Duration) error
	Get(key TKey) (TValue, error)
	Delete(key TKey) error
	Exists(key TKey) (bool, error)
	FreeSpace() (int, error)
	Cap() (int, error)
	Count() (int, error)
	Close()
}
