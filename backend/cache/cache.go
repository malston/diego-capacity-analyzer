// ABOUTME: In-memory cache with TTL-based expiration
// ABOUTME: Thread-safe cache using sync.Map with automatic cleanup

package cache

import (
	"sync"
	"time"
)

type entry struct {
	data      interface{}
	expiresAt time.Time
}

type Cache struct {
	store sync.Map
	ttl   time.Duration
}

func New(ttl time.Duration) *Cache {
	c := &Cache{
		ttl: ttl,
	}
	go c.startCleanup()
	return c
}

func (c *Cache) Get(key string) (interface{}, bool) {
	val, ok := c.store.Load(key)
	if !ok {
		return nil, false
	}

	e := val.(entry)
	if time.Now().After(e.expiresAt) {
		c.store.Delete(key)
		return nil, false
	}

	return e.data, true
}

func (c *Cache) Set(key string, value interface{}) {
	e := entry{
		data:      value,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.store.Store(key, e)
}

// SetWithTTL stores a value with a custom TTL
func (c *Cache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	e := entry{
		data:      value,
		expiresAt: time.Now().Add(ttl),
	}
	c.store.Store(key, e)
}

func (c *Cache) Clear(key string) {
	c.store.Delete(key)
}

func (c *Cache) startCleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		c.store.Range(func(key, val interface{}) bool {
			e := val.(entry)
			if now.After(e.expiresAt) {
				c.store.Delete(key)
			}
			return true
		})
	}
}
