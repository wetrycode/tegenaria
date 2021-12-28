package tegenaria

import (
	queue "github.com/yireyun/go-queue"
)
// CacheInterface request cache interface
// you can use redis to do cache
type CacheInterface interface {
	enqueue(req *Request) error // enqueue put request to cache
	dequeue() (interface{}, error) // dequeue get request from cache
	getSize() int64 // getSize get cache size
}

// requestCache request cache
type requestCache struct {
	queue *queue.EsQueue // A lock-free queue to use cache request  
}

// enqueue put request to cache queue
func (c *requestCache) enqueue(req *Request) error {
	for {
		// It will wait to put request until queue is not full
		ok, _ := c.queue.Put(req)
		if ok {
			return nil
		}
	}
}

// dequeue get request object from cache queue
func (c *requestCache) dequeue() (interface{}, error) {
	val, ok, _ := c.queue.Get()
	if !ok {
		return nil, ErrGetCacheItem
	} else {
		return val, nil
	}

}

// getSize get cache queue size
func (c *requestCache) getSize() int64 {
	return int64(c.queue.Quantity())
}

// NewRequestCache get a new requestCache
func NewRequestCache() *requestCache {
	return &requestCache{
		queue: queue.NewQueue(1024 * 2),
	}
}
