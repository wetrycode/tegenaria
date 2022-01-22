package tegenaria

import (
	"context"
	"time"
)

type Context struct {
	// Request
	Request *Request

	// Response *Response
	DownloadResult *RequestResult

	// Item
	Item           ItemInterface

	//Ctx
	Ctx            context.Context

	// CtxId
	CtxId          string

	// Error
	Error          error
}
type ContextOption func(c *Context)

func NewContext(request *Request, opts ...ContextOption) *Context {
	ctx := &Context{
		Request: request,
		// Response: NewResponse(),
		Item:           nil,
		Ctx:            context.TODO(),
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
		c.Ctx = ctx
	}
}

func ContextWithItem(item ItemInterface) ContextOption {
	return func(c *Context) {
		c.Item = item
	}
}

// Deadline returns that there is no deadline (ok==false) when c has no Context.
func (c *Context) Deadline() (deadline time.Time, ok bool) {
	if c.Request == nil || c.Ctx == nil {
		return
	}
	return c.Ctx.Deadline()
}

// Done returns nil (chan which will wait forever) when c.Request has no Context.
func (c *Context) Done() <-chan struct{} {
	if c.Request == nil || c.Ctx == nil {
		return nil
	}
	return c.Ctx.Done()
}

// Err returns nil when ct has no Context.
func (c *Context) Err() error {
	if c.Request == nil || c.Ctx == nil {
		return nil
	}
	return c.Ctx.Err()
}

// Value returns the value associated with this context for key, or nil
// if no value is associated with key. Successive calls to Value with
// the same key returns the same result.
func (c *Context) Value(key interface{}) interface{} {
	if key == 0 {
		return c.Request
	}
	if c.Request == nil || c.Ctx == nil {
		return nil
	}
	return c.Ctx.Value(key)
}
func (c Context) GetCtxId() string {
	return c.CtxId
}

func (c Context) AddItem(item ItemInterface) {
	// c.Items = append(c.Items, item)
}
