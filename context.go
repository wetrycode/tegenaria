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
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/sirupsen/logrus"
)

type ContextInterface interface {
	IsDone() bool
}

// Context 在引擎中的数据流通载体，负责单个抓取任务的生命周期维护
type Context struct {
	// Request 请求对象
	Request *Request

	// Response 响应对象
	Response *Response

	//parent 父 context
	parent context.Context

	// CtxId context 唯一id由uuid生成
	CtxId string

	// Error
	Error error

	//
	Cancel context.CancelFunc
	//
	Ref int64

	//
	Items chan *ItemMeta

	Spider SpiderInterface
}

// contextPool context 内存池
var contextPool *sync.Pool = &sync.Pool{
	New: func() interface{} {
		return new(Context)
	},
}

// 全局的context 管理接口
type contextManager struct {
	// ctxCount context对象的数量
	ctxCount int64
	// ctxMap context缓存
	ctxMap cmap.ConcurrentMap[*Context]
}

var onceContextManager sync.Once

type ContextOption func(c *Context)

var ctxManager *contextManager

// add 向context 管理组件添加新的context
func (c *contextManager) add(ctx *Context) {
	c.ctxMap.Set(ctx.CtxId, ctx)
	atomic.AddInt64(&c.ctxCount, 1)

}

// remove 从contextManager中删除指定的ctx
func (c *contextManager) remove(ctx *Context) {
	c.ctxMap.Remove(ctx.CtxId)
	atomic.AddInt64(&c.ctxCount, -1)

}

// isEmpty未处理的ctx是否为空
func (c *contextManager) isEmpty() bool {
	engineLog.Debugf("Number of remaining tasks:%d", atomic.LoadInt64(&c.ctxCount))
	return atomic.LoadInt64(&c.ctxCount) == 0
}

// Clear 清空ctx
func (c *contextManager) Clear() {
	atomic.StoreInt64(&c.ctxCount, 0)
	c.ctxMap.Clear()
}

// var ctxCount int64
func newContextManager() {
	onceContextManager.Do(func() {
		ctxManager = &contextManager{
			ctxCount: 0,
			ctxMap:   cmap.New[*Context](),
		}
	})
}

// WithContextId 设置自定义的ctxId
func WithContextId(ctxId string) ContextOption {
	return func(c *Context) {
		c.CtxId = ctxId
	}
}
func WithItemChannelSize(size int) ContextOption {
	return func(c *Context) {
		c.Items = make(chan *ItemMeta, size)
	}
}

// NewContext 从内存池中构建context对象
func NewContext(request *Request, Spider SpiderInterface, opts ...ContextOption) *Context {
	ctx := contextPool.Get().(*Context)
	parent, cancel := context.WithCancel(context.TODO())
	ctx.Request = request
	ctx.Spider = Spider
	ctx.CtxId = GetUUID()
	ctx.Cancel = cancel
	ctx.Items = make(chan *ItemMeta, 32)
	ctx.parent = parent
	ctx.Error = nil
	log.Infof("Generate a new request%s %s", ctx.CtxId, request.Url)

	for _, o := range opts {
		o(ctx)
	}
	ctxManager.add(ctx)
	return ctx

}

// freeContext 重置context并返回到内存池
func freeContext(c *Context) {
	c.parent = nil
	c.Cancel = nil
	c.Items = nil
	c.CtxId = ""
	c.Spider = nil
	c.Error = nil
	contextPool.Put(c)
	c = nil
}

// setResponse 设置响应
func (c *Context) setResponse(resp *Response) {
	c.Response = resp
}

// setError 设置异常
func (c *Context) setError(msg string, stack string) {
	DebugStack := stack
	// logger.Errorf(DebugStack)
	for _, v := range strings.Split(DebugStack, "\n") {
		DebugStack += v
	}

	err := NewError(c, fmt.Errorf("%s", msg))
	c.Error = err
	// 取上一帧栈
	pc, file, lineNo, _ := runtime.Caller(1)
	f := runtime.FuncForPC(pc)
	fields := logrus.Fields{
		"request_id": c.CtxId,
		"func":       f.Name(),
		"file":       fmt.Sprintf("%s:%d", file, lineNo),
		"stack":      DebugStack,
	}
	log := engineLog.WithFields(fields)
	log.Logger.SetReportCaller(false)
	log.Errorf("%s", err.Error())

}

// Close 关闭context
func (c *Context) Close() {
	// if c.Request != nil {
	// 	// 释放request
	// 	// freeRequest(c.Request)
	// }
	if c.Response != nil {
		// 释放response
		freeResponse(c.Response)
	}
	ctxManager.remove(c)
	c.CtxId = ""
	freeContext(c)

}

// WithContext 设置父context
func WithContext(ctx context.Context) ContextOption {
	return func(c *Context) {
		parent, cancel := context.WithCancel(ctx)
		c.parent = parent
		c.Cancel = cancel
	}
}

// Deadline
func (c *Context) Deadline() (deadline time.Time, ok bool) {
	if c.Request == nil || c.parent == nil {
		return time.Time{}, false
	}
	return c.parent.Deadline()
}

func (c *Context) Done() <-chan struct{} {
	if c.Request == nil || c.parent == nil {
		return nil
	}
	return c.parent.Done()
}

func (c *Context) Err() error {
	if c.Request == nil || c.parent == nil {
		return nil
	}
	return c.parent.Err()
}

func (c *Context) Value(key interface{}) interface{} {
	if key == 0 {
		return c.Request
	}
	if c.Request == nil || c.parent == nil {
		return nil
	}
	return c.parent.Value(key)
}
func (c Context) GetCtxId() string {
	return c.CtxId
}
