package tegenaria

import (
	"errors"
	"fmt"
	"strconv"
)

var (
	ErrSpiderMiddleware    error = errors.New("handle spider middleware error")
	ErrSpiderCrawls        error = errors.New("handle spider crawl error")
	ErrDuplicateSpiderName error = errors.New("register a duplicate spider name error")
	ErrEmptySpiderName     error = errors.New("register a empty spider name error")
	ErrSpiderNotExist      error = errors.New("not found spider")
	ErrNotAllowStatusCode  error = errors.New("not allow handle status code")
	ErrGetCacheItem        error = errors.New("getting item from cache error")
	ErrGetHttpProxy        error = errors.New("getting http proxy ")
	ErrGetHttpsProxy       error = errors.New("getting https proxy ")
	ErrParseSocksProxy     error = errors.New("parse socks proxy ")
	ErrResponseRead        error = errors.New("read response ro buffer error")
)

type RedirectError struct {
	RedirectNum int
}

type HandleError struct {
	CtxId    string
	Err      error
	Request  *Request
	Response *Response
	Item     *ItemMeta
}
type ErrorOption func(e *HandleError)

func NewError(ctxId string, err error, opts ...ErrorOption) *HandleError {
	h:= &HandleError{
		CtxId: ctxId,
		Err:   err,
		Request: nil,
		Response: nil,
		Item: nil,
	}
	for _, o := range opts {
		o(h)
	}
	return h
}

func ErrorWithRequest(request *Request) ErrorOption {
	return func(e *HandleError) {
		e.Request = request
	}
}

func ErrorWithResponse(response *Response) ErrorOption {
	return func(e *HandleError) {
		e.Response = response
	}
}

func ErrorWithItem(item *ItemMeta) ErrorOption {
	return func(e *HandleError) {
		e.Item = item
	}
}
func (e *HandleError) Error() string {
	return fmt.Sprintf("%s with context id %s", e.Err.Error(), e.CtxId)
}
func (e *RedirectError) Error() string {
	return "exceeded the maximum number of redirects: " + strconv.Itoa(e.RedirectNum)
}
