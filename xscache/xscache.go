package xscache

import (
	"sync"
)

var (
	cache = make(map[string]*CacheTable) // 缓存库
	mutex sync.RWMutex                   // 全局锁
)

// 返回一个存在的表, 或创建一个新的缓存表
func Cache(table string) *CacheTable {
	mutex.RLock()
	t, ok := cache[table]
	mutex.RUnlock()

	// 检查表是否存在, 不存在则创建
	if !ok {
		mutex.Lock()
		t = &CacheTable{
			name:  table,
			items: make(map[interface{}]*CacheItem),
		}

		cache[table] = t
		mutex.Unlock()
	}

	return t
}
