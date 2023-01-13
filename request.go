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
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"sync"

	jsoniter "github.com/json-iterator/go"
	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
)

type Proxy struct {
	ProxyUrl string
}

// Request a spider request config
type Request struct {
	Url             string                 `json:"url"`             // Set request URL
	Header          map[string]string      `json:"header"`          // Set request header
	Method          string                 `json:"method"`          // Set request Method
	Body            []byte                 `json:"body"`            // Set request body
	Params          map[string]string      `json:"params"`          // Set request query params
	Proxy           *Proxy                 `json:"-"`               // Set request proxy addr
	Cookies         map[string]string      `json:"cookies"`         // Set request cookie
	Meta            map[string]interface{} `json:"meta"`            // Set other data
	AllowRedirects  bool                   `json:"allowRedirects"`  // Set if allow redirects. default is true
	MaxRedirects    int                    `json:"maxRedirects"`    // Set max allow redirects number
	Parser          Parser                 `json:"-"`               // Set response parser funcation
	MaxConnsPerHost int                    `json:"maxConnsPerHost"` // Set max connect number for per host
	BodyReader      io.Reader              `json:"-"`               // Set request body reader
	ResponseWriter  io.Writer              `json:"-"`               // Set request response body writer,like file
	AllowStatusCode []uint64               `json:"allowStatusCode"`
	// RequestId       string                 // Set request uuid
}

// requestPool the Request obj pool
var requestPool *sync.Pool = &sync.Pool{
	New: func() interface{} {
		return new(Request)
	},
}

// Option NewRequest options
type RequestOption func(r *Request)

// Parser response parse handler
type Parser func(resp *Context, req chan<- *Context) error

// bufferPool buffer object pool
var bufferPool *sync.Pool = &sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 8192))
		// return new(bytes.Buffer)
	},
}

// reqLog request logger
var reqLog *logrus.Entry = GetLogger("request")

func RequestWithRequestBody(body map[string]interface{}) RequestOption {
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
func RequestWithRequestBytesBody(body []byte) RequestOption {
	return func(r *Request) {
		r.Body = body
	}
}
func RequestWithRequestParams(params map[string]string) RequestOption {
	return func(r *Request) {
		r.Params = params
	}
}
func RequestWithRequestProxy(proxy Proxy) RequestOption {
	return func(r *Request) {
		r.Proxy = &proxy
	}
}
func RequestWithRequestHeader(header map[string]string) RequestOption {
	return func(r *Request) {
		r.Header = header
	}
}
func RequestWithRequestCookies(cookies map[string]string) RequestOption {
	return func(r *Request) {
		r.Cookies = cookies
	}
}

func RequestWithRequestMeta(meta map[string]interface{}) RequestOption {
	return func(r *Request) {
		r.Meta = meta
	}
}
func RequestWithAllowRedirects(allowRedirects bool) RequestOption {
	return func(r *Request) {
		r.AllowRedirects = allowRedirects
		if !allowRedirects {
			r.MaxRedirects = 0
		}
	}
}
func RequestWithMaxRedirects(maxRedirects int) RequestOption {
	return func(r *Request) {
		r.MaxRedirects = maxRedirects
		if maxRedirects <= 0 {
			r.AllowRedirects = false
		}
	}
}
func RequestWithResponseWriter(write io.Writer) RequestOption {
	return func(r *Request) {
		r.ResponseWriter = write
	}
}
func RequestWithMaxConnsPerHost(maxConnsPerHost int) RequestOption {
	return func(r *Request) {
		r.MaxConnsPerHost = maxConnsPerHost
	}
}

func RequestWithAllowedStatusCode(allowStatusCode []uint64) RequestOption {
	return func(r *Request) {
		r.AllowStatusCode = allowStatusCode
	}
}

func RequestWithParser(parser Parser) RequestOption {
	return func(r *Request) {
		r.Parser = parser
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
func NewRequest(url string, method string, parser Parser, opts ...RequestOption) *Request {
	request := requestPool.Get().(*Request)
	request.Url = url
	request.Method = method
	request.Parser = parser
	request.ResponseWriter = nil
	request.BodyReader = nil
	request.Header = make(map[string]string)
	request.MaxRedirects = 3
	request.AllowRedirects = true
	request.Proxy = nil
	request.AllowStatusCode = make([]uint64, 0)
	for _, o := range opts {
		o(request)
	}
	request.updateQueryParams()
	return request

}

// freeRequest reset Request and the put it into requestPool
func freeRequest(r *Request) {
	r.Parser = func(resp *Context, req chan<- *Context) error {
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
	r.MaxConnsPerHost = 512
	r.ResponseWriter = nil
	r.BodyReader = nil
	requestPool.Put(r)
	r = nil

}
func (r *Request) ToMap() (map[string]interface{}, error) {
	b, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	err = json.Unmarshal(b, &m)
	return m, err

}
func RequestFromMap(src map[string]interface{}, opts ...RequestOption) *Request {
	request := requestPool.Get().(*Request)
	mapstructure.Decode(src, request)
	for _, o := range opts {
		o(request)
	}
	request.updateQueryParams()
	return request

}
