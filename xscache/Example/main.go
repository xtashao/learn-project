package main

import (
	"fmt"
	"strconv"
	"time"
	"xscache"
)

func main() {
	// logs := log.New(os.Stdout, "", 1)

	cache := xscache.Cache("test")

	// cache.SetLogger(logs)

	cache.SetDataLoader(func(key interface{}, args ...interface{}) *xscache.CacheItem {
		// Apply some clever loading logic here, e.g. read values for
		// this key from database, network or file.
		val := "This is a test with key " + key.(string)

		// This helper method creates the cached item for us. Yay!
		item := xscache.NewCacheItem(key, val, 5*time.Second)
		return item
	})

	// Let's retrieve a few auto-generated items from the cache.
	for i := 0; i < 10; i++ {
		res, err := cache.Value("someKey_" + strconv.Itoa(i))
		if err == nil {
			fmt.Println("Found value in cache:", res.Data())
		} else {
			fmt.Println("Error retrieving value from cache:", err)
		}
	}

	fmt.Println("Count : ", cache.Count())
	// item, _ := cache.Delete("someKey_1")
	// fmt.Println("Delete : ", item.Data())
	time.Sleep(3 * time.Second)

	cache.Foreach(func(key interface{}, item *xscache.CacheItem) {
		fmt.Println("Key: ", key, "; Val: ", item.Data())
	})

}
