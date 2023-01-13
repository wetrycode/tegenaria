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
	ErrResponseRead        error = errors.New("read response to buffer error")
	ErrResponseParse       error = errors.New("parse response error")
)

type RedirectError struct {
	RedirectNum int
}

type HandleError struct {
	Ctx      *Context
	Err      error
	Request  *Request
	Response *Response
	Item     *ItemMeta
}
type ErrorOption func(e *HandleError)

func NewError(ctx *Context, err error, opts ...ErrorOption) *HandleError {
	h := &HandleError{
		Ctx:      ctx,
		Err:      err,
	}
	for _, o := range opts {
		o(h)
	}
	return h
}

func (e *HandleError) Error() string {
	return fmt.Sprintf("%s with context id %s", e.Err.Error(), e.Ctx.CtxId)
}
func (e *RedirectError) Error() string {
	return "exceeded the maximum number of redirects: " + strconv.Itoa(e.RedirectNum)
}
