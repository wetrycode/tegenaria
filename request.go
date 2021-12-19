package tegenaria

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spaolacci/murmur3"
)

// Request a url
type Request struct {
	Url            string            // Request URL
	Header         map[string]string // Request header
	Method         string            // Request Method
	Body           []byte            // Request body
	Params         map[string]string // Request query params
	Proxy          string            // Request proxy addr
	Cookies        map[string]string
	Timeout        time.Duration
	TLS            bool
	Meta           map[string]interface{}
	AllowRedirects bool
	MaxRedirects   int
}
type Option func(r *Request)

var reqLog *logrus.Entry = GetLogger("request")

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
func WithAllowRedirects(allowRedirects bool) Option {
	return func(r *Request) {
		r.AllowRedirects = allowRedirects
	}
}
func WithMaxRedirects(maxRedirects int) Option {
	return func(r *Request) {
		r.MaxRedirects = maxRedirects
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
		Url:            url,
		Header:         map[string]string{},
		Method:         method,
		Body:           []byte{},
		Params:         map[string]string{},
		Proxy:          "",
		Cookies:        map[string]string{},
		Timeout:        10 * time.Second,
		TLS:            false,
		Meta:           map[string]interface{}{},
		AllowRedirects: true,
		MaxRedirects:   -1,
	}
	for _, o := range opts {
		o(request)
	}
	request.updateQueryParams()
	return request

}
func (r *Request) canonicalizeUrl(keepFragment bool) url.URL {
	u, _ := url.ParseRequestURI(r.Url)
	u.RawQuery = u.Query().Encode()
	u.ForceQuery = true
	if !keepFragment {
		u.Fragment = ""
	}
	return *u
}
func (r *Request) encodeHeader() string {
	h := r.Header
	if h == nil {
		return ""
	}
	var buf bytes.Buffer
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	// 对Header的键进行排序
	sort.Strings(keys)
	for _, k := range keys {
		// 对值进行排序
		buf.WriteString(fmt.Sprintf("%s:%s;\n", strings.ToUpper(k), strings.ToUpper(h[k])))
	}
	return buf.String()
}
func (r *Request) Fingerprint() []byte {
	sha := murmur3.New128()
	io.WriteString(sha, r.Method)
	u := r.canonicalizeUrl(false)
	io.WriteString(sha, u.String())
	if r.Body != nil {
		body := r.Body
		sha.Write(body)
	}
	if len(r.Header) != 0 {
		io.WriteString(sha, r.encodeHeader())
	}
	res := sha.Sum(nil)
	return res
}
