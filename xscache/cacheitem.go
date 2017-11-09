package xscache

import (
	"sync"
	"time"
)

// CacheItem 单个缓存项
type CacheItem struct {
	sync.RWMutex

	// 元素的key
	key interface{}
	// 元素的值
	data interface{}
	// 当元素未被访问时,该元素将在缓存中生存多久
	lifeSpan time.Duration

	// 创建时间
	createdOn time.Time
	// 最后一次访问时间
	accessedOn time.Time
	// 该元素访问次数
	accessCount int64

	// 缓存在被移除前触发的回到函数
	aboutToExpire func(key interface{})
}

func NewCacheItem(key interface{}, data interface{}, lifeSpan time.Duration) *CacheItem {
	t := time.Now()
	return &CacheItem{
		key:           key,
		data:          data,
		lifeSpan:      lifeSpan,
		createdOn:     t,
		accessedOn:    t,
		accessCount:   0,
		aboutToExpire: nil,
	}
}

// 标记一个元素访问时间及计数器
func (item *CacheItem) KeepAlive() {
	item.Lock()
	defer item.Unlock()
	item.accessedOn = time.Now()
	item.accessCount++
}

// LifeSpan returns this item's expiration duration.
func (item *CacheItem) LifeSpan() time.Duration {
	// immutable
	return item.lifeSpan
}

// AccessedOn returns when this item was last accessed.
func (item *CacheItem) AccessedOn() time.Time {
	item.RLock()
	defer item.RUnlock()
	return item.accessedOn
}

// CreatedOn returns when this item was added to the cache.
func (item *CacheItem) CreatedOn() time.Time {
	// immutable
	return item.createdOn
}

// AccessCount returns how often this item has been accessed.
func (item *CacheItem) AccessCount() int64 {
	item.RLock()
	defer item.RUnlock()
	return item.accessCount
}

// 返回元素的KEY
func (item *CacheItem) Key() interface{} {
	return item.key
}

// 返回元素的值
func (item *CacheItem) Data() interface{} {
	return item.data
}

// 设置元素失效回调, 在元素即将从缓存中移除之前将调用它
func (item *CacheItem) SetAboutToExpireCallback(f func(interface{})) {
	item.Lock()
	defer item.Unlock()
	item.aboutToExpire = f
}
