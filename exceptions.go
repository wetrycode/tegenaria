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
	"errors"
	"fmt"
	"strconv"
)

var (
	// ErrSpiderMiddleware 下载中间件处理异常
	ErrSpiderMiddleware error = errors.New("handle spider middleware error")
	// ErrSpiderCrawls 抓取流程错误
	ErrSpiderCrawls error = errors.New("handle spider crawl error")
	// ErrDuplicateSpiderName 爬虫名重复错误
	ErrDuplicateSpiderName error = errors.New("register a duplicate spider name error")
	// ErrEmptySpiderName 爬虫名不能为空
	ErrEmptySpiderName error = errors.New("register a empty spider name error")
	// ErrSpiderNotExist 爬虫实例不存在
	ErrSpiderNotExist error = errors.New("not found spider")
	// ErrNotAllowStatusCode 不允许的状态码
	ErrNotAllowStatusCode error = errors.New("not allow handle status code")
	// ErrGetCacheItem 获取item 错误
	ErrGetCacheItem error = errors.New("getting item from cache error")
	// ErrGetHttpProxy 获取http代理错误
	ErrGetHttpProxy error = errors.New("getting http proxy ")
	// ErrGetHttpsProxy 获取https代理错误
	ErrGetHttpsProxy error = errors.New("getting https proxy ")
	// ErrParseSocksProxy 解析socks代理错误
	ErrParseSocksProxy error = errors.New("parse socks proxy ")
	// ErrResponseRead 响应读取失败
	ErrResponseRead error = errors.New("read response to buffer error")
	// ErrResponseParse 响应解析失败
	ErrResponseParse error = errors.New("parse response error")
)

// RedirectError 重定向错误
type RedirectError struct {
	RedirectNum int
}

// HandleError 错误处理接口
type HandleError struct {
	// CtxID 上下文id
	CtxID string
	// Err 处理过程的错误
	Err error
	// Extras 携带的额外信息
	Extras map[string]interface{}
}

// ErrorOption HandleError 可选参数
type ErrorOption func(e *HandleError)

// ErrorWithExtras HandleError 添加额外的数据
func ErrorWithExtras(extras map[string]interface{}) ErrorOption {
	return func(e *HandleError) {
		e.Extras = extras
	}
}

// NewError 构建新的HandleError实例
func NewError(ctx *Context, err error, opts ...ErrorOption) *HandleError {
	h := &HandleError{
		CtxID: ctx.CtxID,
		Err:   err,
	}
	for _, o := range opts {
		o(h)
	}
	return h
}

// Error 获取HandleError错误信息
func (e *HandleError) Error() string {
	return fmt.Sprintf("%s with context id %s", e.Err.Error(), e.CtxID)
}

// Error获取RedirectError错误
func (e *RedirectError) Error() string {
	return "exceeded the maximum number of redirects: " + strconv.Itoa(e.RedirectNum)
}
