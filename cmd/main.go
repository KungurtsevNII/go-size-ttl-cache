package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"runtime"
	"time"

	go_size_ttl_cache "go-size-ttl-cache"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	fmt.Printf("1 - gorutine count - %d\n", runtime.NumGoroutine())
	cache, err := go_size_ttl_cache.NewMemoryCache[int, int64](1000)
	if err != nil {
		log.Fatalf("can't create cache because - %s", err.Error())
	}
	fmt.Printf("2 - gorutine count - %d\n", runtime.NumGoroutine())

	go func(ctx context.Context, ttlCache go_size_ttl_cache.SizedTTLCache[int, int64]) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				key := rand.Intn(100)
				value, err := cache.Get(key)
				if err != nil {
					log.Printf("gorutin can't get from cache by ket <%v>, because - %s", key, err.Error())
				} else {
					log.Printf("gorutin from cache - <%v,%v>", key, value)
				}
				time.Sleep(time.Millisecond * 10)
			}
		}
	}(ctx, cache)
	fmt.Printf("3 - gorutine count - %d\n", runtime.NumGoroutine())

	go func(ctx context.Context, ttlCache go_size_ttl_cache.SizedTTLCache[int, int64]) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				keyValue := rand.Intn(50)
				err = cache.Put(keyValue, int64(keyValue), time.Millisecond*100)
				if err != nil {
					log.Printf("gorutine can't add to cache (%v, %v) because - %s", keyValue, keyValue, err.Error())
				} else {
					log.Printf("gorutine put to cache (%v, %v)", keyValue, keyValue)
				}
				time.Sleep(time.Millisecond * 10)
			}
		}
	}(ctx, cache)

	fmt.Printf("4 - gorutine count - %d\n", runtime.NumGoroutine())

	go func(ctx context.Context, ttlCache go_size_ttl_cache.SizedTTLCache[int, int64]) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				key := rand.Intn(50)
				exists, err := cache.Exists(key)
				if err != nil {
					log.Printf("gorutine can't check exists for (%v) because - %s", key, err.Error())
				} else {
					log.Printf("gorutine exists (%v) - %v", key, exists)
				}

				capacity, err := cache.Cap()
				if err != nil {
					log.Printf("gorutine can't get capacity because - %s", err.Error())
				} else {
					log.Printf("gorutine cache capacity (%d)", capacity)
				}

				c, err := cache.Count()
				if err != nil {
					log.Printf("gorutine can't get count because - %s", err.Error())
				} else {
					log.Printf("gorutine cache count (%d)", c)
				}
				time.Sleep(time.Millisecond * 10)
			}
		}
	}(ctx, cache)

	fmt.Printf("5 - gorutine count - %d\n", runtime.NumGoroutine())

	for i := 0; i < 57; i++ {
		err = cache.Put(i, int64(i), time.Millisecond*100)
		if err != nil {
			log.Printf("can't add to cache (%v, %v) because - %s", i, i, err.Error())
		}

		space, err := cache.FreeSpace()
		if err != nil {
			log.Printf("can't get free space in cache because - %s", err.Error())
		} else {
			log.Printf("free spase - %d", space)
		}
	}

	for i := 0; i < 57; i++ {
		value, err := cache.Get(i)
		if err != nil {
			log.Printf("can't get from cache by ket <%v>, because - %s", i, err.Error())
		} else {
			log.Printf("from cache - <%v,%v>", i, value)
		}

		space, err := cache.FreeSpace()
		if err != nil {
			log.Printf("can't get free space in cache because - %s", err.Error())
		} else {
			log.Printf("free spase - %d", space)
		}
	}

	fmt.Printf("6 - gorutine count - %d\n", runtime.NumGoroutine())
	time.Sleep(time.Second * 5)

	cancel()
	cache.Close()

	fmt.Printf("7 - gorutine count - %d\n", runtime.NumGoroutine())

	time.Sleep(time.Second * 10)

	fmt.Printf("8 - gorutine count - %d\n", runtime.NumGoroutine())
}
