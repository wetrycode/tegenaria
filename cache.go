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
	"errors"
	"fmt"

	queue "github.com/yireyun/go-queue"
)

// CacheInterface request缓存组件
type CacheInterface interface {
	// enqueue ctx写入缓存
	enqueue(ctx *Context) error    
	// dequeue ctx 从缓存出队列
	dequeue() (interface{}, error)
	// isEmpty 缓存是否为空
	isEmpty() bool                 
	// getSize 缓存大小
	getSize() uint64
	// close 关闭缓存
	close() error
	// setCurrentSpider 设置当前的spider
	setCurrentSpider(spider string)
}

// requestCache request缓存队列
type requestCache struct {
	queue *queue.EsQueue
}

// enqueue request对象入队列
func (c *requestCache) enqueue(ctx *Context) error {
	// It will wait to put request until queue is not full
	if ctx == nil || ctx.Request == nil {
		return errors.New("context or request cannot be nil")
	}
	ok, q := c.queue.Put(ctx)
	if !ok {
		return fmt.Errorf("enter queue error %d", q)
	}
	return nil

}

// dequeue 从队列中获取request对象
func (c *requestCache) dequeue() (interface{}, error) {
	val, ok, _ := c.queue.Get()
	if !ok {
		return nil, ErrGetCacheItem
	} else {
		return val, nil
	}

}
// isEmpty 缓存是否为空
func (c *requestCache) isEmpty() bool {
	return int64(c.queue.Quantity()) == 0
}
// getSize 缓存大小
func (c *requestCache) getSize() uint64 {
	return uint64(c.queue.Quantity())
}
// close 关闭缓存
func (c *requestCache) close() error {
	return nil
}
// setCurrentSpider 设置当前的spider
func (c *requestCache) setCurrentSpider(spider string) {

}
// NewRequestCache get a new requestCache
func NewRequestCache() *requestCache {
	return &requestCache{
		queue: queue.NewQueue(1024 * 1024),
	}
}
