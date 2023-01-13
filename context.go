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
	"context"
	"sync"
	"sync/atomic"
	"time"

	cmap "github.com/orcaman/concurrent-map/v2"
)

type ContextInterface interface {
	IsDone() bool
}

// Context spider crawl request schedule unit
// it is used on all data flow
type Context struct {
	// Request
	Request *Request

	// DownloadResult downloader handler result
	Response *Response

	//parent parent context
	parent context.Context

	// CtxId
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

type contextManager struct {
	ctxCount int64
	ctxMap   cmap.ConcurrentMap[*Context]
}

var onceContextManager sync.Once

type ContextOption func(c *Context)

var ctxManager *contextManager

func (c *contextManager) add(ctx *Context) {
	c.ctxMap.Set(ctx.CtxId, ctx)
	atomic.AddInt64(&c.ctxCount, 1)

}

func (c *contextManager) remove(ctx *Context) {
	c.ctxMap.Remove(ctx.CtxId)
	atomic.AddInt64(&c.ctxCount, -1)

}
func (c *contextManager) isEmpty() bool {
	engineLog.Debugf("Number of remaining tasks:%d", atomic.LoadInt64(&c.ctxCount))
	return atomic.LoadInt64(&c.ctxCount) == 0
}
func (c *contextManager)Clear(){
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
func WithContextId(ctxId string) ContextOption {
	return func(c *Context) {
		c.CtxId = ctxId
	}
}
func NewContext(request *Request, Spider SpiderInterface, opts ...ContextOption) *Context {
	parent, cancel := context.WithCancel(context.TODO())
	ctx := &Context{
		Request: request,
		parent:  parent,
		CtxId:   GetUUID(),
		Cancel:  cancel,
		Spider:  Spider,
		Items:   make(chan *ItemMeta, 16),
	}
	log.Debugf("Generate a new request%s %s", ctx.CtxId, request.Url)

	for _, o := range opts {
		o(ctx)
	}
	ctxManager.add(ctx)
	return ctx

}
func (c *Context) setResponse(resp *Response) {
	c.Response = resp
}
func (c *Context) Close() {
	if c.Request != nil {
		freeRequest(c.Request)
	}
	if c.Response != nil {
		freeResponse(c.Response)
	}
	ctxManager.remove(c)

}
func WithContext(ctx context.Context) ContextOption {
	return func(c *Context) {
		parent, cancel := context.WithCancel(ctx)
		c.parent = parent
		c.Cancel = cancel
	}
}

// Deadline returns that there is no deadline (ok==false) when c has no Context.
func (c *Context) Deadline() (deadline time.Time, ok bool) {
	if c.Request == nil || c.parent == nil {
		return time.Time{}, false
	}
	return c.parent.Deadline()
}

// Done returns nil (chan which will wait forever) when c.Request has no Context.
func (c *Context) Done() <-chan struct{} {
	if c.Request == nil || c.parent == nil {
		return nil
	}
	return c.parent.Done()
}

// Err returns nil when ct has no Context.
func (c *Context) Err() error {
	if c.Request == nil || c.parent == nil {
		return nil
	}
	return c.parent.Err()
}

// Value returns the value associated with this context for key, or nil
// if no value is associated with key. Successive calls to Value with
// the same key returns the same result.
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
