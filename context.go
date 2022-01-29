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
	"time"
)
// Context spider crawl request schedule unit
// it is used on all data flow
type Context struct {
	// Request
	Request *Request

	// DownloadResult downloader handler result
	DownloadResult *RequestResult

	//parent parent context
	parent context.Context

	// CtxId
	CtxId string

	// Error
	Error error
}
type ContextOption func(c *Context)

func NewContext(request *Request, opts ...ContextOption) *Context {
	ctx := &Context{
		Request: request,
		parent:            nil,
		CtxId:          GetUUID(),
		DownloadResult: NewDownloadResult(),
	}
	for _, o := range opts {
		o(ctx)
	}
	return ctx

}
func WithContext(ctx context.Context) ContextOption {
	return func(c *Context) {
		c.parent = ctx
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
