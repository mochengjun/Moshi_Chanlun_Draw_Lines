package cache

import (
	"container/list"
	"sync"
	"time"
)

// MemoryCache 内存LRU缓存
type MemoryCache struct {
	maxSize   int
	ttl       time.Duration
	cache     map[string]*list.Element
	lruList   *list.List
	mu        sync.RWMutex
	hits      int64
	misses    int64
	stopChan  chan struct{}
}

// cacheEntry 缓存条目
type cacheEntry struct {
	key       string
	value     interface{}
	expireAt  time.Time
}

// NewMemoryCache 创建内存缓存
func NewMemoryCache(maxSize int, ttl time.Duration) *MemoryCache {
	mc := &MemoryCache{
		maxSize:  maxSize,
		ttl:      ttl,
		cache:    make(map[string]*list.Element),
		lruList:  list.New(),
		stopChan: make(chan struct{}),
	}
	
	// 启动清理协程
	go mc.cleanupLoop()
	
	return mc
}

// Get 获取缓存
func (mc *MemoryCache) Get(key string) (interface{}, bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	elem, ok := mc.cache[key]
	if !ok {
		mc.misses++
		return nil, false
	}
	
	entry := elem.Value.(*cacheEntry)
	
	// 检查是否过期
	if time.Now().After(entry.expireAt) {
		mc.removeElement(elem)
		mc.misses++
		return nil, false
	}
	
	// 移动到列表头部（LRU）
	mc.lruList.MoveToFront(elem)
	mc.hits++
	
	return entry.value, true
}

// Set 设置缓存
func (mc *MemoryCache) Set(key string, value interface{}) {
	mc.SetWithTTL(key, value, mc.ttl)
}

// SetWithTTL 设置带TTL的缓存
func (mc *MemoryCache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	// 如果key已存在，更新
	if elem, ok := mc.cache[key]; ok {
		entry := elem.Value.(*cacheEntry)
		entry.value = value
		entry.expireAt = time.Now().Add(ttl)
		mc.lruList.MoveToFront(elem)
		return
	}
	
	// 如果缓存已满，删除最老的条目
	if mc.lruList.Len() >= mc.maxSize {
		mc.removeOldest()
	}
	
	// 添加新条目
	entry := &cacheEntry{
		key:      key,
		value:    value,
		expireAt: time.Now().Add(ttl),
	}
	elem := mc.lruList.PushFront(entry)
	mc.cache[key] = elem
}

// Delete 删除缓存
func (mc *MemoryCache) Delete(key string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	if elem, ok := mc.cache[key]; ok {
		mc.removeElement(elem)
	}
}

// removeElement 删除元素
func (mc *MemoryCache) removeElement(elem *list.Element) {
	entry := elem.Value.(*cacheEntry)
	delete(mc.cache, entry.key)
	mc.lruList.Remove(elem)
}

// removeOldest 删除最老的元素
func (mc *MemoryCache) removeOldest() {
	if elem := mc.lruList.Back(); elem != nil {
		mc.removeElement(elem)
	}
}

// cleanupLoop 清理过期缓存
func (mc *MemoryCache) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			mc.cleanup()
		case <-mc.stopChan:
			return
		}
	}
}

// cleanup 清理过期缓存
func (mc *MemoryCache) cleanup() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	now := time.Now()
	for elem := mc.lruList.Back(); elem != nil; {
		entry := elem.Value.(*cacheEntry)
		if now.After(entry.expireAt) {
			prev := elem.Prev()
			mc.removeElement(elem)
			elem = prev
		} else {
			elem = elem.Prev()
		}
	}
}

// Stats 获取缓存统计
func (mc *MemoryCache) Stats() (hits, misses int64, size int, hitRate float64) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	hits = mc.hits
	misses = mc.misses
	size = mc.lruList.Len()
	
	total := hits + misses
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}
	
	return
}

// Clear 清空缓存
func (mc *MemoryCache) Clear() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.cache = make(map[string]*list.Element)
	mc.lruList.Init()
}

// Close 关闭缓存，停止清理协程
func (mc *MemoryCache) Close() {
	close(mc.stopChan)
}
