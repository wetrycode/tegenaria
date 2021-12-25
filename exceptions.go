package tegenaria

import (
	"errors"
	"strconv"
)

var (
	ErrSpiderMiddleware    error = errors.New("Do handle spider middleware error")
	ErrSpiderCrawls        error = errors.New("Do handle spider crawl error")
	ErrDuplicateSpiderName error = errors.New("Register a duplicate spider name error")
	ErrEmptySpiderName     error = errors.New("Register a empty spider name error")
	ErrSpiderNotExist      error = errors.New("Not found spider")
	ErrNotAllowStatusCode  error = errors.New("Not allow handle status code")
	ErrGetCacheItem        error = errors.New("Getting item from cache error")
	ErrGetHttpProxy        error = errors.New("Getting http proxy ")
	ErrGetHttpsProxy       error = errors.New("Getting https proxy ")
	ErrParseSocksProxy     error = errors.New("Parse socks proxy ")
)

type RedirectError struct {
	RedirectNum int
}

func (e *RedirectError) Error() string {
	return "exceeded the maximum number of redirects: " + strconv.Itoa(e.RedirectNum)
}
