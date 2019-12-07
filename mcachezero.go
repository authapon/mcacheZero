package mcacheZero

import (
	"container/list"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

const (
	writeAlways = 0
	writeEvict  = 1
)

type writeFuncType func(string, interface{})
type readFuncType func(string) (interface{}, error)
type deleteFuncType func(string)

type Cache struct {
	sync.Mutex

	items             map[string]*list.Element
	size              int
	writeFunc         writeFuncType
	readFunc          readFuncType
	deleteFunc        deleteFuncType
	writeMode         int
	writeWaitingCount int64
	expire            time.Duration
	evict             *list.List
}

type item struct {
	key    string
	value  interface{}
	expire int64
	dirty  bool
}

func New(size int) *Cache {
	data := new(Cache)
	data.size = size
	data.items = make(map[string]*list.Element)
	data.writeMode = writeAlways
	data.writeWaitingCount = int64(0)
	data.evict = list.New()
	data.expire = 0
	return data
}

func (c *Cache) Set(key string, value interface{}) {
	c.Lock()
	defer c.Unlock()

	defer c.evictAct()
	c.evictAct()

	if data, found := c.items[key]; found {
		data.Value.(*item).value = value
		data.Value.(*item).dirty = true
		if c.expire != 0 {
			expireTime := time.Now()
			exT := expireTime.Add(c.expire)
			data.Value.(*item).expire = exT.Unix()
		}
		c.evict.MoveToFront(data)
	} else {
		c.addItem(key, value, true)
	}

	if c.writeMode == writeAlways {
		if c.writeFunc != nil {
			c.writeWaiting()
			atomic.AddInt64(&c.writeWaitingCount, int64(1))
			go func(k string, v interface{}, cc *Cache) {
				cc.writeFunc(k, v)
				atomic.AddInt64(&cc.writeWaitingCount, int64(-1))
			}(key, value, c)
		}
	}
}

func (c *Cache) Get(key string) (interface{}, error) {
	c.Lock()
	defer c.Unlock()

	defer c.evictAct()
	c.evictAct()

	if data, found := c.items[key]; found {
		c.evict.MoveToFront(data)
		if c.expire != 0 {
			expireTime := time.Now()
			exT := expireTime.Add(c.expire)
			data.Value.(*item).expire = exT.Unix()
		}
		return data.Value.(*item).value, nil
	} else {
		if c.readFunc != nil {
			c.writeWaiting()
			readData, err := c.readFunc(key)
			if err != nil {
				return nil, err
			}
			c.addItem(key, readData, false)
			return readData, nil
		} else {
			return nil, errors.New("Not found error.")
		}
	}
}

func (c *Cache) Delete(key string) {
	c.Lock()
	defer c.Unlock()

	defer c.evictAct()
	c.evictAct()

	if data, found := c.items[key]; found {
		c.evict.Remove(data)
		delete(c.items, key)
	}

	if c.deleteFunc != nil {
		c.writeWaiting()
		atomic.AddInt64(&c.writeWaitingCount, int64(1))
		go func(k string, cc *Cache) {
			cc.deleteFunc(k)
			atomic.AddInt64(&cc.writeWaitingCount, int64(-1))
		}(key, c)
	}
}

func (c *Cache) addItem(key string, value interface{}, dirty bool) {
	expireT := int64(0)

	if c.expire != 0 {
		expireTime := time.Now()
		exT := expireTime.Add(c.expire)
		expireT = exT.Unix()
	}

	newdata := &item{key: key, value: value, expire: expireT, dirty: dirty}
	newitem := c.evict.PushFront(newdata)
	c.items[key] = newitem
	c.evictAct()
}

func (c *Cache) Purge() {
	c.Lock()
	c.items = make(map[string]*list.Element)
	c.evict = list.New()
	c.Unlock()
}

func (c *Cache) SetWriteFunc(fn writeFuncType) {
	c.Lock()
	c.writeFunc = fn
	c.Unlock()
}

func (c *Cache) SetReadFunc(fn readFuncType) {
	c.Lock()
	c.readFunc = fn
	c.Unlock()
}

func (c *Cache) SetDeleteFunc(fn deleteFuncType) {
	c.Lock()
	c.deleteFunc = fn
	c.Unlock()
}

func (c *Cache) SetExpireDuration(expire time.Duration) {
	c.Lock()
	c.expire = expire
	c.Unlock()
}

func (c *Cache) keys() []string {
	keys := make([]string, len(c.items))
	i := 0

	for key, _ := range c.items {
		keys[i] = key
		i++
	}
	return keys
}

func (c *Cache) Keys() []string {
	c.Lock()
	defer c.Unlock()

	return c.keys()
}

func (c *Cache) Remove(key string) {
	c.Lock()
	c.remove(key)
	c.Unlock()
}

func (c *Cache) remove(key string) {
	if data, found := c.items[key]; found {
		deletekey := data.Value.(*item).key
		deletevalue := data.Value.(*item).value
		dirty := data.Value.(*item).dirty

		c.evict.Remove(data)
		delete(c.items, deletekey)

		if c.writeMode == writeEvict {
			if c.writeFunc != nil {
				if dirty {
					atomic.AddInt64(&c.writeWaitingCount, int64(1))
					go func(k string, v interface{}, cc *Cache) {
						cc.writeFunc(k, v)
						atomic.AddInt64(&cc.writeWaitingCount, int64(-1))
					}(deletekey, deletevalue, c)
				}
			}
		}
	}
}

func (c *Cache) SetWriteAlways() {
	c.Lock()
	c.writeMode = writeAlways
	c.Unlock()
}

func (c *Cache) SetWriteEvict() {
	c.Lock()
	c.writeMode = writeEvict
	c.Unlock()
}

func (c *Cache) EvictAct() {
	c.Lock()
	c.evictAct()
	c.Unlock()
}

func (c *Cache) evictAct() {
	if c.expire != 0 {
		nowTime := time.Now().Unix()
		for {
			deletedata := c.evict.Back()
			if deletedata == nil {
				break
			}
			deleteexpire := deletedata.Value.(*item).expire
			if nowTime >= deleteexpire {
				deletekey := deletedata.Value.(*item).key
				c.remove(deletekey)
			} else {
				break
			}
		}
	}

	if c.size != 0 {
		for len(c.items) > c.size {
			deletedata := c.evict.Back()
			if deletedata == nil {
				break
			}
			deletekey := deletedata.Value.(*item).key
			c.remove(deletekey)
		}
	}
}

func (c *Cache) Len() int {
	c.Lock()
	defer c.Unlock()

	return len(c.items)
}

func (c *Cache) WriteWaiting() {
	c.Lock()
	c.writeWaiting()
	c.Unlock()
}

func (c *Cache) writeWaiting() {
	for atomic.LoadInt64(&c.writeWaitingCount) != 0 {
		runtime.Gosched()
	}
}

func (c *Cache) flush() {
	allkeys := c.keys()

	for i := range allkeys {
		c.remove(allkeys[i])
	}
}

func (c *Cache) Flush() {
	c.Lock()
	c.flush()
	c.Unlock()
}

func (c *Cache) Sync() {
	c.Lock()
	c.sync()
	c.Unlock()
}

func (c *Cache) sync() {
	if c.writeFunc == nil {
		return
	}

	c.writeWaiting()
	allkeys := c.keys()

	for i := range allkeys {
		key := allkeys[i]

		if data, found := c.items[key]; found {
			value := data.Value.(*item).value
			dirty := data.Value.(*item).dirty
			data.Value.(*item).dirty = false

			if dirty {
				atomic.AddInt64(&c.writeWaitingCount, int64(1))
				go func(k string, v interface{}, cc *Cache) {
					cc.writeFunc(k, v)
					atomic.AddInt64(&cc.writeWaitingCount, int64(-1))
				}(key, value, c)
			}
		}
	}
}
