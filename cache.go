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
	Enqueue(ctx *Context) error
	// dequeue ctx 从缓存出队列
	Dequeue() (interface{}, error)
	// isEmpty 缓存是否为空
	IsEmpty() bool
	// getSize 缓存大小
	GetSize() uint64
	// close 关闭缓存
	Close() error
	// SetCurrentSpider 设置当前的spider
	SetCurrentSpider(spider SpiderInterface)
}

// RequestCache request缓存队列
type DefaultQueue struct {
	queue  *queue.EsQueue
	spider SpiderInterface
}

// enqueue request对象入队列
func (c *DefaultQueue) Enqueue(ctx *Context) error {
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
func (c *DefaultQueue) Dequeue() (interface{}, error) {
	val, ok, _ := c.queue.Get()
	if !ok {
		return nil, ErrGetCacheItem
	}
	return val, nil

}

// isEmpty 缓存是否为空
func (c *DefaultQueue) IsEmpty() bool {
	fmt.Printf("queue size:%d", c.queue.Quantity())
	return int64(c.queue.Quantity()) == 0
}

// getSize 缓存大小
func (c *DefaultQueue) GetSize() uint64 {
	return uint64(c.queue.Quantity())
}

// close 关闭缓存
func (c *DefaultQueue) Close() error {
	return nil
}

// SetCurrentSpider 设置当前的spider
func (c *DefaultQueue) SetCurrentSpider(spider SpiderInterface) {
	c.spider = spider
}

// NewDefaultQueue get a new DefaultQueue
func NewDefaultQueue(size int) *DefaultQueue {
	return &DefaultQueue{
		queue: queue.NewQueue(uint32(size)),
	}
}
