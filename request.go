package tegenaria

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	bloom "github.com/bits-and-blooms/bloom/v3"
	jsoniter "github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
	"github.com/spaolacci/murmur3"
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
	Timeout         time.Duration          // Set request timeout
	TLS             bool                   // Set https request if skip tls check
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
type Parser func(resp *Context, item chan<- *ItemMeta, req chan<- *Context)

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
		body, err := jsoniter.Marshal(body)
		r.BodyReader = bytes.NewBuffer(body)
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
func RequestWithRequestTimeout(timeout time.Duration) Option {
	return func(r *Request) {
		r.Timeout = timeout
	}
}
func RequestWithRequestTLS(tls bool) Option {
	return func(r *Request) {
		r.TLS = tls
	}
}
func RequestWithRequestMethod(method string) Option {
	return func(r *Request) {
		r.Method = method
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
		if !allowRedirects{
			r.MaxRedirects = 0
		}
	}
}
func RequestWithMaxRedirects(maxRedirects int) Option {
	return func(r *Request) {
		r.MaxRedirects = maxRedirects
		if maxRedirects <=0{
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
	request.Timeout = 10 * time.Second
	request.ResponseWriter = nil
	request.BodyReader = nil
	request.Header = make(map[string]string)
	request.MaxRedirects = 3
	request.AllowRedirects = true
	// u4 := uuid.New()
	// request.RequestId = u4.String()
	for _, o := range opts {
		o(request)
	}
	request.updateQueryParams()
	return request

}

// canonicalizeUrl canonical request url before calculate request fingerprint
func (r *Request) canonicalizeUrl(keepFragment bool) url.URL {
	u, _ := url.ParseRequestURI(r.Url)
	u.RawQuery = u.Query().Encode()
	u.ForceQuery = true
	if !keepFragment {
		u.Fragment = ""
	}
	return *u
}

// encodeHeader encode request header before calculate request fingerprint
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
	// Sort by Header key
	sort.Strings(keys)
	for _, k := range keys {
		// Sort by value
		buf.WriteString(fmt.Sprintf("%s:%s;\n", strings.ToUpper(k), strings.ToUpper(h[k])))
	}
	return buf.String()
}

// fingerprint generate a request fingerprint by using 	murmur3.New128() sha128
// the fingerprint []byte will be cached into cache module or de-duplication
func (r *Request) fingerprint() ([]byte, error) {
	// get sha128
	sha := murmur3.New128()
	_, err := io.WriteString(sha, r.Method)
	if err != nil {
		return nil, err
	}
	// canonical request url
	u := r.canonicalizeUrl(false)
	_, err = io.WriteString(sha, u.String())
	if err != nil {
		return nil, err
	}
	// get request body
	if r.Body != nil {
		body := r.Body
		sha.Write(body)
	}
	// to handle request header
	if len(r.Header) != 0 {
		_,err:=io.WriteString(sha, r.encodeHeader())
		if err !=nil{
			return nil, err
		}
	}
	res := sha.Sum(nil)
	return res, nil
}

func (r *Request) doUnique(bloomFilter *bloom.BloomFilter) (bool,error) {
	// Use bloom filter to do fingerprint deduplication
	data, err:= r.fingerprint()
	if err !=nil{
		return false, err
	}
	return bloomFilter.TestOrAdd(data), nil
}

// freeRequest reset Request and the put it into requestPool
func freeRequest(r *Request) {
	r.parser = func(resp *Context, item chan<- *ItemMeta, req chan<- *Context) {}
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
	r.Timeout = 10 * time.Second
	r.TLS = false
	r.maxConnsPerHost = 512
	r.ResponseWriter = nil
	r.BodyReader = nil
	// r.RequestId = ""
	requestPool.Put(r)
	r = nil

}
