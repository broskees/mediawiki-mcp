package wiki

import (
	"sync"
	"time"
)

// Cache is a simple in-memory TTL cache
type Cache struct {
	items map[string]*cacheItem
	mu    sync.RWMutex
}

type cacheItem struct {
	value      interface{}
	expiration time.Time
}

// NewCache creates a new cache instance
func NewCache() *Cache {
	c := &Cache{
		items: make(map[string]*cacheItem),
	}

	// Start cleanup goroutine
	go c.cleanupLoop()

	return c
}

// Get retrieves a value from cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(item.expiration) {
		return nil, false
	}

	return item.value, true
}

// Set stores a value in cache with TTL
func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &cacheItem{
		value:      value,
		expiration: time.Now().Add(ttl),
	}
}

// Delete removes a value from cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// cleanupLoop periodically removes expired items
func (c *Cache) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

func (c *Cache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, item := range c.items {
		if now.After(item.expiration) {
			delete(c.items, key)
		}
	}
}

// CacheKey generates a cache key for a request
func CacheKey(parts ...string) string {
	key := ""
	for i, part := range parts {
		if i > 0 {
			key += ":"
		}
		key += part
	}
	return key
}

// Helper for common cache key patterns
func PageCacheKey(wikiURL, title string) string {
	return CacheKey("page", wikiURL, title)
}

func SectionCacheKey(wikiURL, title, sectionIndex string) string {
	return CacheKey("section", wikiURL, title, sectionIndex)
}

func SearchCacheKey(wikiURL, query string) string {
	return CacheKey("search", wikiURL, query)
}

func InfoCacheKey(wikiURL string) string {
	return CacheKey("info", wikiURL)
}

func CategoryCacheKey(wikiURL, category string) string {
	return CacheKey("category", wikiURL, category)
}

func BacklinksCacheKey(wikiURL, title string) string {
	return CacheKey("backlinks", wikiURL, title)
}
