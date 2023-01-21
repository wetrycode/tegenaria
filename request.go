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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
)

// Proxy 代理数据结构
type Proxy struct {
	// ProxyUrl 代理链接
	ProxyUrl string
}

// Request 请求对象的结构
type Request struct {
	// Url 请求Url
	Url string `json:"url"`
	// Header 请求头
	Header map[string]string `json:"header"`
	// Method 请求方式
	Method RequestMethod `json:"method"`
	// Body 请求body
	Body []byte `json:"body"`
	// Params 请求url的参数
	Params map[string]string `json:"params"`
	// Proxy 代理实例
	Proxy *Proxy `json:"-"`
	// Cookies 请求携带的cookies
	Cookies map[string]string `json:"cookies"`
	// Meta 请求携带的额外的信息
	Meta map[string]interface{} `json:"meta"`
	// AllowRedirects 是否允许跳转默认允许
	AllowRedirects bool `json:"allowRedirects"`
	// MaxRedirects 最大的跳转次数
	MaxRedirects int `json:"maxRedirects"`
	// Parser 该请求绑定的响应解析函数，必须是一个spider实例
	Parser Parser `json:"-"`
	// MaxConnsPerHost 单个域名最大的连接数
	MaxConnsPerHost int `json:"maxConnsPerHost"`
	// BodyReader 用于读取body
	BodyReader io.Reader `json:"-"`
	// ResponseWriter 响应读取到本地的接口
	ResponseWriter io.Writer `json:"-"`
	// AllowStatusCode 允许的状态码
	AllowStatusCode []uint64 `json:"allowStatusCode"`
	// Timeout 请求超时时间
	Timeout time.Duration `json:"timeout"`
}

// requestPool request对象内存池
var requestPool *sync.Pool = &sync.Pool{
	New: func() interface{} {
		return new(Request)
	},
}

// Option NewRequest 可选参数
type RequestOption func(r *Request)

// Parser 响应解析函数结构
type Parser func(resp *Context, req chan<- *Context) error

// bufferPool buffer 对象内存池
var bufferPool *sync.Pool = &sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 8192))
		// return new(bytes.Buffer)
	},
}

// reqLog request logger
var reqLog *logrus.Entry = GetLogger("request")

// RequestWithRequestBody 传入请求体到request
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

// RequestWithRequestBytesBody request绑定bytes body
func RequestWithRequestBytesBody(body []byte) RequestOption {
	return func(r *Request) {
		r.Body = body
	}
}

// RequestWithRequestParams 设置请求的url参数
func RequestWithRequestParams(params map[string]string) RequestOption {
	return func(r *Request) {
		r.Params = params
	}
}

// RequestWithRequestProxy 设置代理
func RequestWithRequestProxy(proxy Proxy) RequestOption {
	return func(r *Request) {
		r.Proxy = &proxy
	}
}

// RequestWithRequestHeader 设置请求头
func RequestWithRequestHeader(header map[string]string) RequestOption {
	return func(r *Request) {
		r.Header = header
	}
}

// RequestWithRequestCookies 设置cookie
func RequestWithRequestCookies(cookies map[string]string) RequestOption {
	return func(r *Request) {
		r.Cookies = cookies
	}
}

// RequestWithRequestMeta 设置 meta
func RequestWithRequestMeta(meta map[string]interface{}) RequestOption {
	return func(r *Request) {
		r.Meta = meta
	}
}

// RequestWithAllowRedirects 设置是否允许跳转
// 如果不允许则MaxRedirects=0
func RequestWithAllowRedirects(allowRedirects bool) RequestOption {
	return func(r *Request) {
		r.AllowRedirects = allowRedirects
		if !allowRedirects {
			r.MaxRedirects = 0
		}
	}
}

// RequestWithMaxRedirects 设置最大的跳转次数
// 若maxRedirects <= 0则认为不允许跳转AllowRedirects = false
func RequestWithMaxRedirects(maxRedirects int) RequestOption {
	return func(r *Request) {
		if maxRedirects <= 0 {
			r.AllowRedirects = false
		} else {
			r.MaxRedirects = maxRedirects
			r.AllowRedirects = true
		}
	}
}

// RequestWithResponseWriter 设置ResponseWriter
func RequestWithResponseWriter(write io.Writer) RequestOption {
	return func(r *Request) {
		r.ResponseWriter = write
	}
}

// RequestWithMaxConnsPerHost 设置MaxConnsPerHost
func RequestWithMaxConnsPerHost(maxConnsPerHost int) RequestOption {
	return func(r *Request) {
		r.MaxConnsPerHost = maxConnsPerHost
	}
}

// RequestWithAllowedStatusCode 设置AllowStatusCode
func RequestWithAllowedStatusCode(allowStatusCode []uint64) RequestOption {
	return func(r *Request) {
		r.AllowStatusCode = allowStatusCode
	}
}

// RequestWithParser 设置Parser
func RequestWithParser(parser Parser) RequestOption {
	return func(r *Request) {
		r.Parser = parser
	}
}

// RequestWithTimeout 设置请求超时时间
// 若timeout<=0则认为没有超时时间
func RequestWithTimeout(timeout time.Duration) RequestOption {
	return func(r *Request) {
		r.Timeout = timeout
	}
}

// updateQueryParams 将Params配置到url
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

// NewRequest 从Request对象内存池创建新的Request对象
func NewRequest(url string, method RequestMethod, parser Parser, opts ...RequestOption) *Request {
	request := requestPool.Get().(*Request)
	request.Url = url
	request.Method = method
	request.Parser = parser
	request.ResponseWriter = nil
	request.BodyReader = nil
	request.Meta = nil
	request.Header = make(map[string]string)
	request.MaxRedirects = 3
	request.AllowRedirects = true
	request.Proxy = nil
	request.AllowStatusCode = make([]uint64, 0)
	request.Timeout = -1 * time.Second
	for _, o := range opts {
		o(request)
	}
	request.updateQueryParams()
	return request

}

// freeRequest 重置request对象并将对象放回内存池
func freeRequest(r *Request) {
	r.Parser = func(resp *Context, req chan<- *Context) error {
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
	r.Timeout = -1 * time.Second
	requestPool.Put(r)
	r = nil

}

// ToMap 将request对象转为map
func (r *Request) ToMap() (map[string]interface{}, error) {
	b, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	err = json.Unmarshal(b, &m)
	return m, err

}

// RequestFromMap 从map创建requests
func RequestFromMap(src map[string]interface{}, opts ...RequestOption) *Request {
	request := requestPool.Get().(*Request)
	mapstructure.Decode(src, request)
	for _, o := range opts {
		o(request)
	}
	request.updateQueryParams()
	return request

}
