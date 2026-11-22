package cache

import (
	"sync"
	"time"
)

type cacheEntry struct {
	createdAt time.Time
	val       []byte
}

// Cache is a thread-safe in-memory store with TTL-based expiration.
type Cache struct {
	mu      sync.Mutex
	entries map[string]cacheEntry
	ttl     time.Duration
}

// NewCache creates a Cache whose entries expire after ttl and starts a background reaper.
func NewCache(ttl time.Duration) *Cache {
	c := &Cache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
	}
	go c.reapLoop()
	return c
}

// Add stores val under key with the current timestamp.
func (c *Cache) Add(key string, val []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	cp := make([]byte, len(val))
	copy(cp, val)
	c.entries[key] = cacheEntry{
		createdAt: time.Now(),
		val:       cp,
	}
}

// Get retrieves the cached value for key if it exists and has not expired.
func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	if time.Since(entry.createdAt) >= c.ttl {
		delete(c.entries, key)
		return nil, false
	}

	cp := make([]byte, len(entry.val))
	copy(cp, entry.val)
	return cp, true
}

func (c *Cache) reapLoop() {
	ticker := time.NewTicker(c.ttl)
	defer ticker.Stop()

	for range ticker.C {
		c.reap()
	}
}

func (c *Cache) reap() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for k, entry := range c.entries {
		if now.Sub(entry.createdAt) >= c.ttl {
			delete(c.entries, k)
		}
	}
}
