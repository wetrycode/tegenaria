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

type HandleError struct{
	CtxId string
	Err error
}
func NewError(ctxId string, err error) *HandleError{
	return &HandleError{
		CtxId: ctxId,
		Err: err,
	}
}
func (e *HandleError)Error() string{
	return fmt.Sprintf("%s with context id %s",e.Err.Error(), e.CtxId)
}
func (e *RedirectError) Error() string {
	return "exceeded the maximum number of redirects: " + strconv.Itoa(e.RedirectNum)
}
