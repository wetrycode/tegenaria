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

package distributed

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wetrycode/tegenaria"
)

var logger = tegenaria.GetLogger("distributed")

// rdbCacheData request 队列缓存的数据结构
type rdbCacheData map[string]interface{}
type DistributeQueueOptions func(w *DistributedQueue)

// serialize 序列化组件
type serialize struct {
	buf bytes.Buffer
	val rdbCacheData
}
type DistributedQueue struct {
	rdb redis.Cmdable
	// getQueueKey 生成队列key的函数，允许用户自定义
	getQueueKey GetRDBKey
	// currentSpider 当前的spider
	currentSpider string
}

// getQueueKey 自定义的request缓存队列key生成函数
func getQueueKey() (string, time.Duration) {
	return "tegenaria:v1:request", 0 * time.Second

}
func NewDistributedQueue(rdb redis.Cmdable, queueKey GetRDBKey) *DistributedQueue {
	return &DistributedQueue{
		rdb:         rdb,
		getQueueKey: queueKey,
	}
}
func (d *DistributedQueue) queueKey() (string, time.Duration) {
	key, ttl := d.getQueueKey()
	return fmt.Sprintf("%s:%s", key, d.currentSpider), ttl

}

// doSerialize 对request 进行序列化操作方便缓存
// 返回的是二进制数组
func (d *DistributedQueue) doSerialize(ctx *tegenaria.Context) ([]byte, error) {
	// 先构建需要缓存的对象
	data, err := newRdbCache(ctx.Request, ctx.CtxID, ctx.Spider.GetName())
	if err != nil {
		return nil, err
	}
	s := &serialize{
		buf: bytes.Buffer{},
		val: data,
	}
	err = s.dumps()
	return s.buf.Bytes(), err
}

// enqueue ctx写入缓存
func (d *DistributedQueue) Enqueue(ctx *tegenaria.Context) error {
	key, _ := d.queueKey()

	bytes, err := d.doSerialize(ctx)
	if err != nil {
		return err
	}
	_, err = d.rdb.LPush(context.TODO(), key, bytes).Uint64()

	return err
}

// dequeue ctx 从缓存出队列
func (d *DistributedQueue) Dequeue() (interface{}, error) {
	key, _ := d.queueKey()
	data, err := d.rdb.RPop(context.TODO(), key).Bytes()
	if err != nil {
		return nil, err
	}
	req, err := unserialize(data)
	if err != nil {
		return nil, err
	}
	spider, err := tegenaria.SpidersList.GetSpider(req["spiderName"].(string))
	if err != nil {
		return nil, err
	}

	opts := []tegenaria.RequestOption{}
	if val, ok := req["proxyUrl"]; ok {
		opts = append(opts, tegenaria.RequestWithRequestProxy(tegenaria.Proxy{ProxyUrl: val.(string)}))
	}
	if val, ok := req["body"]; ok && val != nil {
		opts = append(opts, tegenaria.RequestWithRequestBytesBody(val.([]byte)))
	}
	request := tegenaria.RequestFromMap(req, opts...)
	request.Parser = req["parser"].(string)
	return tegenaria.NewContext(request, spider, tegenaria.WithContextID(req["ctxId"].(string))), nil
}

// isEmpty 缓存是否为空
func (d *DistributedQueue) IsEmpty() bool {
	key, _ := d.queueKey()
	length, err := d.rdb.LLen(context.TODO(), key).Uint64()
	if err != nil {
		length = 0
		logger.Errorf("get queue len error %s", err.Error())

	}

	return int64(length) == 0
}

// getSize 缓存大小
func (d *DistributedQueue) GetSize() uint64 {
	key, _ := d.queueKey()
	length, _ := d.rdb.LLen(context.TODO(), key).Uint64()
	return uint64(length)
}

// close 关闭缓存
func (d *DistributedQueue) Close() error {
	return nil
}

// SetCurrentSpider 设置当前的spider
func (d *DistributedQueue) SetCurrentSpider(spider tegenaria.SpiderInterface) {
	d.currentSpider = spider.GetName()
	funcs := []GetRDBKey{}
	funcs = append(funcs, d.queueKey)
	for _, f := range funcs {
		key, ttl := f()
		if ttl > 0 {
			d.rdb.Expire(context.TODO(), key, ttl)
		}

	}
}
