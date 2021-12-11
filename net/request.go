package net

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	logger "github.com/geebytes/Tegenaria/logging"
	"github.com/sirupsen/logrus"
)

// Request a url
type Request struct {
	Url     string            // Request URL
	Header  map[string]string // Request header
	Method  string            // Request Method
	Body    []byte            // Request body
	Params  map[string]string // Request query params
	Proxy   string            // Request proxy addr
	Cookies map[string]string
	Timeout time.Duration
	TLS     bool
	Meta    map[string]interface{}
}
type Option func(r *Request)

var reqLog *logrus.Entry = logger.GetLogger("request")

func WithRequestBody(body map[string]interface{}) Option {
	return func(r *Request) {
		defer func() {
			if p := recover(); p != nil {
				reqLog.Errorf("panic recover! p: %v", p)
			}
		}()
		var err error
		r.Body, err = json.Marshal(body)
		if err != nil {
			reqLog.Errorf("set request body err %s", err.Error())
			panic(fmt.Sprintf("set request body err %s", err.Error()))
		}
	}
}
func WithRequestParams(params map[string]string) Option {
	return func(r *Request) {
		r.Params = params

	}
}
func WithRequestProxy(proxy string) Option {
	return func(r *Request) {
		r.Proxy = proxy
	}
}
func WithRequestHeader(header map[string]string) Option {
	return func(r *Request) {
		r.Header = header
	}
}
func WithRequestCookies(cookies map[string]string) Option {
	return func(r *Request) {
		r.Cookies = cookies
	}
}
func WithRequestTimeout(timeout time.Duration) Option {
	return func(r *Request) {
		r.Timeout = timeout
	}
}
func WithRequestTLS(tls bool) Option {
	return func(r *Request) {
		r.TLS = tls
	}
}
func WithRequestMethod(method string) Option {
	return func(r *Request) {
		r.Method = method
	}
}
func WithRequestMeta(meta map[string]interface{}) Option {
	return func(r *Request) {
		r.Meta = meta
	}
}
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
func NewRequest(url string, method string, opts ...Option) *Request {
	request := &Request{
		Url:     url,
		Header:  map[string]string{},
		Method:  method,
		Body:    []byte{},
		Params:  map[string]string{},
		Proxy:   "",
		Cookies: map[string]string{},
		Timeout: 10 * time.Second,
		TLS:     false,
		Meta:    map[string]interface{}{},
	}
	for _, o := range opts {
		o(request)
	}
	request.updateQueryParams()
	return request

}
