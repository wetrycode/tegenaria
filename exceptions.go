package tegenaria

import "errors"

var (
	ErrSpiderMiddleware    error = errors.New("Do handle spider middleware error")
	ErrSpiderCrawls        error = errors.New("Do handle spider crawl error")
	ErrDuplicateSpiderName error = errors.New("Register a duplicate spider name error")
	ErrEmptySpiderName     error = errors.New("Register a empty spider name error")
	ErrSpiderNotExist      error = errors.New("Not found spider")
	ErrNotAllowStatusCode  error = errors.New("Not allow handle status code")
	ErrGetCacheItem        error = errors.New("Getting item from cache error")
)
