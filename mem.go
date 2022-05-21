package mem

import (
	"sync"
	"time"
)

type Cache[K comparable, V any] struct {
	getter         func(key K) (V, error)
	expireGetter   func(key K) (V, time.Time, error)
	ExpireInterval time.Duration
	MaxEntries     int

	list    List[K]
	syncMap sync.Map
}

func NewCache[K comparable, V any](expire time.Duration,
	getter func(key K) (V, error)) *Cache[K, V] {

	return &Cache[K, V]{
		getter:         getter,
		ExpireInterval: expire,
		MaxEntries:     1000,
	}
}
func NewExpireCache[K comparable, V any](
	getter func(key K) (V, time.Time, error)) *Cache[K, V] {

	return &Cache[K, V]{
		expireGetter: getter,
		MaxEntries:   1000,
	}
}

type entry[V any] struct {
	value V
	err   error

	rotateAt time.Time
	expireAt time.Time
	rw       sync.RWMutex
}

func (c *Cache[K, V]) Get(key K) (V, error) {
	now := time.Now()
	v, ok := c.syncMap.Load(key)
	if ok {
		e := v.(*entry[V])
		if e.expireAt.After(now) {
			return e.value, e.err
		}
	}

	e := &entry[V]{}
	if val, ok := c.syncMap.LoadOrStore(key, e); ok {
		e = val.(*entry[V])
	} else {
		c.list.PushFront(key)
	}

	c.fulfill(key, e)
	c.gc(now)
	return e.value, e.err
}

func (c *Cache[K, V]) fulfill(key K, e *entry[V]) {
	if !e.rw.TryLock() {
		e.rw.RLock() // wait for unlock
		e.rw.RUnlock()
		return
	}
	defer e.rw.Unlock()

	if c.expireGetter != nil {
		e.value, e.expireAt, e.err = c.expireGetter(key)
		return
	}

	if e.value, e.err = c.getter(key); e.err != nil {
		return
	}

	e.expireAt = time.Now().Add(c.ExpireInterval)

	c.list.MoveToFront(&Element[K]{Value: key})
}

func (c *Cache[K, V]) gc(now time.Time) {
	ele := c.list.Back()
	if ele == nil {
		return
	}

	if c.list.len > c.MaxEntries {
		c.list.Remove(ele)
		return
	}

	if val, ok := c.syncMap.Load(ele.Value); ok {
		if val.(*entry[V]).expireAt.Before(now) {
			c.syncMap.Delete(ele.Value)
			c.list.Remove(ele)
			c.gc(now)
		}
	}
}

func (c *Cache[K, V]) Remove(key K) {
	c.list.Remove(&Element[K]{Value: key})
	c.syncMap.Delete(key)
}
