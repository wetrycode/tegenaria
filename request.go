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
	"bytes"
	"fmt"
	"io"
	"net/url"
	"sync"

	jsoniter "github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
)

type Proxy struct {
	ProxyUrl string
}

// Request a spider request config
type Request struct {
	Url             string                 // Set request URL
	Header          map[string]string      // Set request header
	Method          string                 // Set request Method
	Body            []byte                 // Set request body
	Params          map[string]string      // Set request query params
	Proxy           *Proxy                 // Set request proxy addr
	Cookies         map[string]string      // Set request cookie
	Meta            map[string]interface{} // Set other data
	AllowRedirects  bool                   // Set if allow redirects. default is true
	MaxRedirects    int                    // Set max allow redirects number
	parser          Parser                 // Set response parser funcation
	maxConnsPerHost int                    // Set max connect number for per host
	BodyReader      io.Reader              // Set request body reader
	ResponseWriter  io.Writer              // Set request response body writer,like file
	// RequestId       string                 // Set request uuid
}

// requestPool the Request obj pool
var requestPool *sync.Pool = &sync.Pool{
	New: func() interface{} {
		return new(Request)
	},
}

// Option NewRequest options
type Option func(r *Request)

// Parser response parse handler
type Parser func(resp *Context, item chan<- *ItemMeta, req chan<- *Context) error

// bufferPool buffer object pool
var bufferPool *sync.Pool = &sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 8192))
		// return new(bytes.Buffer)
	},
}

// reqLog request logger
var reqLog *logrus.Entry = GetLogger("request")

func RequestWithRequestBody(body map[string]interface{}) Option {
	return func(r *Request) {
		var err error

		r.Body, err = jsoniter.Marshal(body)
		r.BodyReader = bytes.NewBuffer(r.Body)
		if err != nil {
			reqLog.Errorf("set request body err %s", err.Error())
			panic(fmt.Sprintf("set request body err %s", err.Error()))
		}
	}
}
func RequestWithRequestParams(params map[string]string) Option {
	return func(r *Request) {
		r.Params = params
	}
}
func RequestWithRequestProxy(proxy Proxy) Option {
	return func(r *Request) {
		r.Proxy = &proxy
	}
}
func RequestWithRequestHeader(header map[string]string) Option {
	return func(r *Request) {
		r.Header = header
	}
}
func RequestWithRequestCookies(cookies map[string]string) Option {
	return func(r *Request) {
		r.Cookies = cookies
	}
}

func RequestWithRequestMeta(meta map[string]interface{}) Option {
	return func(r *Request) {
		r.Meta = meta
	}
}
func RequestWithAllowRedirects(allowRedirects bool) Option {
	return func(r *Request) {
		r.AllowRedirects = allowRedirects
		if !allowRedirects {
			r.MaxRedirects = 0
		}
	}
}
func RequestWithMaxRedirects(maxRedirects int) Option {
	return func(r *Request) {
		r.MaxRedirects = maxRedirects
		if maxRedirects <= 0 {
			r.AllowRedirects = false
		}
	}
}
func RequestWithResponseWriter(write io.Writer) Option {
	return func(r *Request) {
		r.ResponseWriter = write
	}
}
func RequestWithMaxConnsPerHost(maxConnsPerHost int) Option {
	return func(r *Request) {
		r.maxConnsPerHost = maxConnsPerHost
	}
}

// updateQueryParams update url query  params
func (r *Request) updateQueryParams() {
	defer func() {
		if p := recover(); p != nil {
			reqLog.Errorf("panic recover! p: %v", p)
		}
	}()
	if len(r.Params) != 0 {
		u, err := url.Parse(r.Url)
		if err != nil {
			panic(fmt.Sprintf("set request query params err %s", err.Error()))
		}
		q := u.Query()
		for key, value := range r.Params {
			q.Set(key, value)
		}
		u.RawQuery = q.Encode()
		r.Url = u.String()
	}
}

// NewRequest create a new Request.
// It will get a nil request form requestPool and then init params
func NewRequest(url string, method string, parser Parser, opts ...Option) *Request {
	request := requestPool.Get().(*Request)
	request.Url = url
	request.Method = method
	request.parser = parser
	request.ResponseWriter = nil
	request.BodyReader = nil
	request.Header = make(map[string]string)
	request.MaxRedirects = 3
	request.AllowRedirects = true
	request.Proxy = nil
	for _, o := range opts {
		o(request)
	}
	request.updateQueryParams()
	return request

}

// freeRequest reset Request and the put it into requestPool
func freeRequest(r *Request) {
	r.parser = func(resp *Context, item chan<- *ItemMeta, req chan<- *Context) error {
		// default response parser
		return nil
	}
	r.AllowRedirects = true
	r.Meta = nil
	r.MaxRedirects = 3
	r.Url = ""
	r.Header = nil
	r.Method = ""
	r.Body = r.Body[:0]
	r.Params = nil
	r.Proxy = nil
	r.Cookies = nil
	r.maxConnsPerHost = 512
	r.ResponseWriter = nil
	r.BodyReader = nil
	requestPool.Put(r)
	r = nil

}
