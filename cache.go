// MIT License

// Copyright (c) 2023 wetrycode

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

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
