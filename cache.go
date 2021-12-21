package tegenaria

import (
	"sync/atomic"

	"github.com/smallnest/queue"
)

type CacheInterface interface {
	enqueue(req *Request) error
	dequeue() (interface{}, error)
	getSize() int64
	isEmpty()bool
}
type requestCache struct {
	queue *queue.LKQueue
	size  *int64
}
var size int64 = 0

func (c *requestCache) enqueue(req *Request) error {
	c.queue.Enqueue(req)
	atomic.AddInt64(c.size, 1)
	return nil
}

func (c *requestCache) dequeue() (interface{}, error) {
	q :=c.queue.Dequeue()
	if q != nil{
		atomic.AddInt64(c.size, -1)
	}
	return q, nil
}
func (c *requestCache) getSize() int64 {
	return atomic.LoadInt64(c.size)
}
func NewrequestCache() *requestCache {
	return &requestCache{
		queue: queue.NewLKQueue(),
		size:  &size,
	}
}
func (c *requestCache)isEmpty()bool{
	return true
	// return atomic.CompareAndSwapInt64()
}
