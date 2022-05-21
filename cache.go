// Copyright 2022 geebytes
// Licensed under the Apache License, Version 2.0 (the 'License');
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an 'AS IS' BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tegenaria

import (
	"runtime"

	queue "github.com/yireyun/go-queue"
)

// CacheInterface request cache interface
// you can use redis to do cache
type CacheInterface interface {
	enqueue(ctx *Context) error    // enqueue put request to cache
	dequeue() (interface{}, error) // dequeue get request from cache
	getSize() int64                // getSize get cache size
}

// requestCache request cache
type requestCache struct {
	queue *queue.EsQueue // A lock-free queue to use cache request
}

// enqueue put request to cache queue
func (c *requestCache) enqueue(ctx *Context) error {
	// It will wait to put request until queue is not full
	if ctx == nil || ctx.Request == nil {
		return nil
	}
	for {
		ok, q := c.queue.Put(ctx)
		if ok {
			return nil
		} else {
			engineLog.Infof("请求入队列失败%d", q)
			runtime.Gosched()
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
		queue: queue.NewQueue(1024 * 1024),
	}
}
