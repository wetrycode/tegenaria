package tegenaria

import (
	queue "github.com/yireyun/go-queue"
)

type CacheInterface interface {
	enqueue(req *Request) error
	dequeue() (interface{}, error)
	getSize() int64
}
type requestCache struct {
	queue *queue.EsQueue
}

func (c *requestCache) enqueue(req *Request) error {
	for {
		ok, _ := c.queue.Put(req)
		if ok {
			return nil
		}
	}
}

func (c *requestCache) dequeue() (interface{}, error) {
	val, ok, _ := c.queue.Get()
	if !ok {
		return nil, ErrGetCacheItem
	} else {
		return val, nil
	}

}
func (c *requestCache) getSize() int64 {
	return int64(c.queue.Quantity())
}
func NewRequestCache() *requestCache {
	return &requestCache{
		queue: queue.NewQueue(1024 * 2),
	}
}
