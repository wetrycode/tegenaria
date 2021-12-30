package tegenaria

import (
	"context"
	"time"
)

type Context struct {
	Request  *Request
	Response *Response
	Items    []ItemInterface
	Ctx      context.Context
	CtxId    string
}
type ContextOption func(c *Context)

func NewContext(request *Request, opts ...ContextOption) *Context {
	ctx := &Context{
		Request:  request,
		Response: NewResponse(),
		Items:    make([]ItemInterface, 0),
		Ctx:      context.TODO(),
		CtxId:    GetUUID(),
	}
	for _, o := range opts {
		o(ctx)
	}
	return ctx

}
func WithContext(ctx context.Context) ContextOption{
	return func(c *Context) {
		c.Ctx = ctx
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
func (c *Context)GetCtxId() string{
	return c.CtxId
}
