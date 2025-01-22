package utils

import (
	"cs425/HyDfs/models"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CacheEntry represents an entry in the cache
type CacheEntry struct {
	Content    []byte
	LastAccess time.Time
}

// LRU cache
type Cache struct {
	data    map[string]CacheEntry
	maxSize int
	mutex   sync.Mutex
}

func NewCache(maxSize int) *Cache {
	return &Cache{
		data:    make(map[string]CacheEntry),
		maxSize: maxSize,
	}
}

// Get retrieves an entry from the cache
func (c *Cache) Get(key string) ([]byte, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	entry, exists := c.data[key]
	if exists {
		entry.LastAccess = time.Now()
		c.data[key] = entry
		return entry.Content, true
	}
	return nil, false
}

func (c *Cache) Put(key string, content []byte) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if len(c.data) >= c.maxSize {
		c.evict()
	}

	c.data[key] = CacheEntry{
		Content:    content,
		LastAccess: time.Now(),
	}
}

func (c *Cache) evict() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.data {
		if oldestKey == "" || entry.LastAccess.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.LastAccess
		}
	}

	delete(c.data, oldestKey)
}

func (c *Cache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data = make(map[string]CacheEntry)
}

// Periodically clear old entries from the cache every hour.
func startPeriodicCacheClear(cache *Cache) {
	ticker := time.NewTicker(3 * time.Minute)

	go func() {
		for range ticker.C {
			log.Println("Clearing old entries from cache")
			cache.Clear()
		}
	}()
}
func getFromCacheIfPresent(fileCache *Cache, readCommand models.CreateCommand) (bool, models.CommandResponse) {

	if content, found := fileCache.Get(readCommand.HydfsFileName); found {
		log.Println("File found in cache")
		err := WriteToLocalFile(readCommand.LocalFileName, content)
		if err != nil {
			return true, models.CommandResponse{
				IsSuccess: false,
				Message:   fmt.Sprintf("Error while writing cached content to local file: %s", err.Error())}
		}
		return true, models.CommandResponse{
			IsSuccess: true,
			Message:   fmt.Sprintf("File read successfully from cache to localFileName: %s", readCommand.LocalFileName)}
	}
	return false, models.CommandResponse{}

}

func WriteToLocalFile(fileName string, content []byte) error {
	dir := filepath.Dir(fileName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file for writing: %v", err)
	}
	defer file.Close()

	_, err = file.Write(content)
	if err != nil {
		return fmt.Errorf("failed to write content to file: %v", err)
	}

	return nil
}
