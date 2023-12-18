package cache

import (
	"fmt"
	"hash/fnv"
	"sync"
	"time"
)

var (
	defaultShardPartitions               = 64 // default shard amount
	NoSpecifiedExpiration  time.Duration = 0  // no specified expiration
)

type Cache struct {
	Lock     sync.RWMutex
	CanRenew bool

	DefaultExpiration time.Duration
	Shards            []*CacheShard
	StopC             chan struct{}
}

type CacheConfig struct {
	CanRenew             bool
	DefaultExpiration    time.Duration
	CleanupInterval      time.Duration
	NumOfShardPartitions int
}

func (c *Cache) runJanitor() {
	ticker := time.NewTicker(c.DefaultExpiration)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			for _, shard := range c.Shards {
				shard.DeleteExpired()
			}
		case <-c.StopC:
			ticker.Stop()
			return
		}
	}
}

func NewWithConfig(config *CacheConfig) *Cache {
	shardsAmount := config.NumOfShardPartitions
	res := &Cache{
		CanRenew:          config.CanRenew,
		DefaultExpiration: config.DefaultExpiration,
		Shards:            make([]*CacheShard, shardsAmount),
		StopC:             make(chan struct{}),
	}
	for i := 0; i < shardsAmount; i++ {
		res.Shards[i] = NewCacheShard(config.DefaultExpiration)
	}
	go res.runJanitor()
	return res
}

func New(defaultExpiration, cleanupInterval time.Duration) *Cache {
	cfg := &CacheConfig{
		CanRenew:             true,
		DefaultExpiration:    defaultExpiration,
		CleanupInterval:      cleanupInterval,
		NumOfShardPartitions: defaultShardPartitions,
	}
	return NewWithConfig(cfg)
}

func (c *Cache) Stop() {
	close(c.StopC)
}

func (c *Cache) Flush() {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	for _, shard := range c.Shards {
		shard.Flush()
	}
}

func (c *Cache) getShardID(key string) int {
	hash := fnv.New32a()
	if _, err := hash.Write([]byte(key)); err != nil {
		panic(err)
	}
	return int(hash.Sum32() % uint32(len(c.Shards)))
}

func (c *Cache) Get(k string) (interface{}, bool) {
	shardID := c.getShardID(k)
	c.Lock.RLock()
	defer c.Lock.RUnlock()

	shard := c.Shards[shardID]
	return shard.Get(k, c.CanRenew)
}

func (c *Cache) SetDefault(k string, x interface{}) {
	c.Set(k, x, c.DefaultExpiration)
}

func (c *Cache) Set(k string, x interface{}, d time.Duration) {
	shardID := c.getShardID(k)
	c.Lock.RLock()
	defer c.Lock.RUnlock()

	shard := c.Shards[shardID]
	shard.Set(k, x, d)
}

func (c *Cache) Add(k string, x interface{}, d time.Duration) error {
	shardID := c.getShardID(k)
	c.Lock.RLock()
	defer c.Lock.RUnlock()

	shard := c.Shards[shardID]
	return shard.Add(k, x, d)
}

func (c *Cache) Replace(k string, x interface{}, d time.Duration) error {
	shardID := c.getShardID(k)
	c.Lock.RLock()
	defer c.Lock.RUnlock()

	shard := c.Shards[shardID]
	return shard.Replace(k, x, d)
}

// Delete
func (c *Cache) Delete(k string) {
	shardID := c.getShardID(k)
	c.Lock.RLock()
	defer c.Lock.RUnlock()

	shard := c.Shards[shardID]
	shard.Delete(k)
}

// Count
func (c *Cache) Count() int64 {
	var res int64
	for _, shard := range c.Shards {
		res += shard.Count
	}
	return res
}

type CacheShard struct {
	DefaultShardExpiration time.Duration
	Lock                   sync.RWMutex
	Items                  map[string]*CacheItem
	Count                  int64
}

type CacheItem struct {
	Object     interface{}
	Expiration int64
}

// NewCacheShard
func NewCacheShard(expiration time.Duration) *CacheShard {
	return &CacheShard{
		DefaultShardExpiration: expiration,
		Items:                  make(map[string]*CacheItem),
	}
}

func (cs *CacheShard) Set(k string, x interface{}, d time.Duration) {
	cs.Lock.Lock()
	defer cs.Lock.Unlock()
	if d == NoSpecifiedExpiration {
		d = cs.DefaultShardExpiration
	}
	if _, found := cs.Items[k]; !found {
		cs.Count++
	}
	cs.Items[k] = &CacheItem{
		Object:     x,
		Expiration: time.Now().Add(d).UnixNano(),
	}
}

// SetDefault
func (cs *CacheShard) SetDefault(k string, x interface{}) {
	cs.Set(k, x, NoSpecifiedExpiration)
}

func (cs *CacheShard) Get(k string, renew bool) (interface{}, bool) {
	cs.Lock.RLock()
	defer cs.Lock.RUnlock()
	item, found := cs.Items[k]
	if !found {
		return nil, false
	}
	if item.Expiration > 0 {
		if time.Now().UnixNano() > item.Expiration {
			return nil, false
		}
	}
	if renew {
		item.Expiration = time.Now().Add(NoSpecifiedExpiration).UnixNano()
	}
	return item.Object, true
}

func (cs *CacheShard) Add(k string, x interface{}, d time.Duration) error {
	if d == NoSpecifiedExpiration {
		d = cs.DefaultShardExpiration
	}
	cs.Lock.Lock()
	defer cs.Lock.Unlock()
	if _, found := cs.Items[k]; found {
		return fmt.Errorf("Item %s already exists", k)
	}
	cs.Count++
	cs.Items[k] = &CacheItem{
		Object:     x,
		Expiration: time.Now().Add(d).UnixNano(),
	}
	return nil
}

// replace
func (cs *CacheShard) Replace(k string, x interface{}, d time.Duration) error {
	if d == NoSpecifiedExpiration {
		d = cs.DefaultShardExpiration
	}
	cs.Lock.Lock()
	defer cs.Lock.Unlock()
	if _, found := cs.Items[k]; !found {
		return fmt.Errorf("Item %s doesn't exist", k)
	}
	cs.Items[k] = &CacheItem{
		Object:     x,
		Expiration: time.Now().Add(d).UnixNano(),
	}
	return nil
}

// Delete
func (cs *CacheShard) Delete(k string) {
	cs.Lock.Lock()
	defer cs.Lock.Unlock()
	delete(cs.Items, k)
	cs.Count--
}

// Flush
func (cs *CacheShard) Flush() {
	cs.Lock.Lock()
	defer cs.Lock.Unlock()
	cs.Items = make(map[string]*CacheItem)
	cs.Count = 0
}

// DeleteExpired
func (cs *CacheShard) DeleteExpired() {
	now := time.Now().UnixNano()
	cs.Lock.Lock()
	defer cs.Lock.Unlock()
	for k, v := range cs.Items {
		if v.Expiration > 0 && now > v.Expiration {
			delete(cs.Items, k)
			cs.Count--
		}
	}
}
