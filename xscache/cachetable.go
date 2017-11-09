package xscache

import (
	"log"
	"sort"
	"sync"
	"time"
)

// CacheTable 是缓存中的表
type CacheTable struct {
	// 表内同步锁
	sync.RWMutex

	// 表名
	name string
	// 所有缓存的元素
	items map[interface{}]*CacheItem

	// 负责触发清理的定时器
	cleanupTimer *time.Timer
	// 下次运行清理定时器时间
	cleanupInterval time.Duration

	// 当前表使用的日志记录器
	logger *log.Logger

	// 试图获取不存在元素时触发的回调方法
	loadData func(key interface{}, args ...interface{}) *CacheItem
	// 添加新元素时触发的回调方法
	addedItem func(item *CacheItem)
	// 删除元素时触发的回调方法
	aboutToDeleteItem func(item *CacheItem)
}

// 统计当前表中元素总数
func (table *CacheTable) Count() int {
	table.RLock()
	defer table.RUnlock()
	return len(table.items)
}

// 遍历所有元素
func (table *CacheTable) Foreach(trans func(key interface{}, item *CacheItem)) {
	table.RLock()
	defer table.RUnlock()

	for k, v := range table.items {
		trans(k, v)
	}
}

// 设置访问不存在元素时回调函数
func (table *CacheTable) SetDataLoader(f func(interface{}, ...interface{}) *CacheItem) {
	table.Lock()
	defer table.Unlock()
	table.loadData = f
}

// 设置新增元素时回调函数
func (table *CacheTable) SetAddedItemCallback(f func(*CacheItem)) {
	table.Lock()
	defer table.Unlock()
	table.addedItem = f
}

// 设置删除元素时回调函数
func (table *CacheTable) SetAboutToDeleteItemCallback(f func(*CacheItem)) {
	table.Lock()
	defer table.Unlock()
	table.aboutToDeleteItem = f
}

// 设置日志
func (table *CacheTable) SetLogger(logger *log.Logger) {
	table.Lock()
	defer table.Unlock()
	table.logger = logger
}

// 添加缓存 - 键值
func (table *CacheTable) Add(key interface{}, data interface{}, lifeSpan time.Duration) *CacheItem {
	item := NewCacheItem(key, data, lifeSpan)

	// 添加元素到缓存
	table.Lock()
	table.addInternal(item)

	return item
}

// 过期检测, 自动调节定时器触发
func (table *CacheTable) expirationCheck() {
	table.Lock()
	if table.cleanupTimer != nil {
		table.cleanupTimer.Stop()
	}
	if table.cleanupInterval > 0 {
		table.log("Expiration check triggered after", table.cleanupInterval, "for table", table.name)
	} else {
		table.log("Expiration check intalled for table", table.name)
	}

	now := time.Now()
	smallestDuration := 0 * time.Second // 最小持续时间(最近过期时间)
	for key, item := range table.items {
		item.RLock()
		lifeSpan := item.lifeSpan
		accessedOn := item.accessedOn
		item.RUnlock()

		if lifeSpan == 0 {
			continue
		}
		if now.Sub(accessedOn) >= lifeSpan { // 当前时间 - 最近访问时间 >= 过期时间
			// 元素已过期
			table.deleteInternal(key)
		} else {
			// 还未过期, 按时间顺序找到最接近生命周期结束的元素
			// 过期时间 - (当前时间 - 最近访问时间) < 最近过期时间
			if smallestDuration == 0 || lifeSpan-now.Sub(accessedOn) < smallestDuration {
				smallestDuration = lifeSpan - now.Sub(accessedOn) // 下一个即将过期的元素
			}
		}
	}

	// 设置下一次运行清理方法的时间
	table.cleanupInterval = smallestDuration
	if smallestDuration > 0 {
		table.cleanupTimer = time.AfterFunc(smallestDuration, func() {
			go table.expirationCheck()
		})
	}

	table.Unlock()
}

func (table *CacheTable) addInternal(item *CacheItem) {
	// 注意: 除非表互斥锁被锁定，否则不要运行此方法！
	// 这将打开它的调用者的回调和运行前的检查
	table.log("Adding item with key", item.key, "and lifespan of", item.lifeSpan, "to table", table.name)
	table.items[item.key] = item

	expDur := table.cleanupInterval
	addedItem := table.addedItem
	table.Unlock()

	// 执行添加元素回调函数
	if addedItem != nil {
		addedItem(item)
	}

	// 如果设置了失效时间, 并且定时器等于0或设置的失效时间小于定时器, 执行失效检测
	if item.lifeSpan > 0 && (expDur == 0 || item.lifeSpan < expDur) {
		table.expirationCheck()
	}
}

func (table *CacheTable) deleteInternal(key interface{}) (*CacheItem, error) {
	r, ok := table.items[key]
	if !ok {
		return nil, ErrKeyNotFound
	}

	aboutToDeleteItem := table.aboutToDeleteItem
	table.Unlock()

	// 执行删除前元素回调函数
	if aboutToDeleteItem != nil {
		aboutToDeleteItem(r)
	}

	r.RLock()
	defer r.RUnlock()
	if r.aboutToExpire != nil {
		r.aboutToExpire(key)
	}

	table.Lock()
	table.log("Deleting item with key", key, "created on", r.createdOn, "and hit", r.accessCount, "items from table", table.name)
	delete(table.items, key)

	return r, nil
}

// 从缓存中删除元素
func (table *CacheTable) Delete(key interface{}) (*CacheItem, error) {
	table.Lock()
	defer table.Unlock()

	return table.deleteInternal(key)
}

// 判断元素是否存在
func (table *CacheTable) Exists(key interface{}) bool {
	table.RLock()
	defer table.RUnlock()
	_, ok := table.items[key]

	return ok
}

// 如果元素不存在则添加元素
func (table *CacheTable) NotFoundAdd(key interface{}, data interface{}, lifeSpan time.Duration) bool {
	table.Lock()

	if _, ok := table.items[key]; ok {
		table.Unlock()
		return false
	}

	item := NewCacheItem(key, data, lifeSpan)
	table.addInternal(item)

	return true
}

// 从缓存中返回一个元素, 并将其标记为保存
// 还可以通过额外的参数传递到dataloader回调函数
func (table *CacheTable) Value(key interface{}, args ...interface{}) (*CacheItem, error) {
	table.RLock()
	r, ok := table.items[key]
	loadData := table.loadData
	table.RUnlock()

	if ok {
		// 更新访问计数和时间
		r.KeepAlive()
		return r, nil
	}

	// 元素不存在, 尝试使用数据加载器来获取
	if loadData != nil {
		item := loadData(key, args...)
		if item != nil {
			table.Add(key, item.data, item.lifeSpan)
			return item, nil
		}

		return nil, ErrKeyNotFoundOrLoadable
	}

	return nil, ErrKeyNotFound
}

// 清空当前表
func (table *CacheTable) Flush() {
	table.Lock()
	defer table.Unlock()

	table.log("Flushing table", table.name)

	table.items = make(map[interface{}]*CacheItem)
	if table.cleanupTimer != nil {
		table.cleanupTimer.Stop()
	}
}

// 缓存表中key与访问次数map
type CacheItemPair struct {
	Key         interface{}
	AccessCount int64
}

// CacheItemPairList是CacheItemPair切片的排序实现
// 实现排序接口
type CacheItemPairList []CacheItemPair

func (p CacheItemPairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p CacheItemPairList) Less(i, j int) bool { return p[i].AccessCount > p[j].AccessCount }
func (p CacheItemPairList) Len() int           { return len(p) }

// 返回最常用的元素
func (table *CacheTable) MostAccessed(count int64) []*CacheItem {
	table.RLock()
	defer table.RUnlock()

	p := make(CacheItemPairList, len(table.items))
	i := 0
	for k, v := range table.items {
		p[i] = CacheItemPair{k, v.accessCount}
		i++
	}
	sort.Sort(p)

	var r []*CacheItem
	c := int64(0)
	for _, v := range p {
		if c >= count {
			break
		}

		item, ok := table.items[v.Key]
		if ok {
			r = append(r, item)
		}
		c++
	}

	return r
}

// 日志记录封装
func (table *CacheTable) log(v ...interface{}) {
	if table.logger == nil {
		return
	}

	table.logger.Println(v)
}
