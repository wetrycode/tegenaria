package tegenaria

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wxnacy/wgo/arrays"
	"golang.org/x/net/http/httpproxy"
)

// ctxKey WithValueContext key data type
type ctxKey string

// Downloader interface
type Downloader interface {
	Download(ctx context.Context, request *Request, result chan<- *RequestResult) // Download core funcation

	CheckStatus(statusCode int, allowStatus []int64) bool // CheckStatus check response status code if allow handle

}

// SpiderDownloader tegenaria spider downloader
type SpiderDownloader struct {
	// StreamThreshold read body threshold using streaming TODO
	// if content length is bigger that,download will read response by streaming
	// it is a feature in the future
	StreamThreshold uint64
	// transport The transport used by the downloader,
	// each request adopts a public network transmission configuration,
	// and a connection pool is used globally
	transport *http.Transport
	// client network request client
	client *http.Client
	// ProxyFunc update proxy for per request
	ProxyFunc func(req *http.Request) (*url.URL, error)
}

// RequestResult network request response result
type RequestResult struct {
	Error error // Error error exception during request
	RequestId string // RequestId record request id
	Response *Response // Response network request response object
}

// DownloaderOption optional parameters of the downloader
type DownloaderOption func(d *SpiderDownloader)

// Request method constant definition
const (
	GET     string = "GET"
	POST    string = "POST"
	PUT     string = "PUT"
	DELETE  string = "DELETE"
	OPTIONS string = "OPTIONS"
	HEAD    string = "HEAD"
)

// log logging of downloader modules
var log *logrus.Entry = GetLogger("downloader")

// globalClient global network request client
var globalClient *http.Client = nil

// onceClient only one client init
var onceClient sync.Once

// envProxyOnce System proxies load only one
var envProxyOnce sync.Once

// envProxyFuncValue System proxies get funcation
var envProxyFuncValue func(*url.URL) (*url.URL, error)

// newClient get http client
func newClient(client http.Client) {
	onceClient.Do(func() {
		if globalClient == nil {
			globalClient = &client
		}
	})
}

// proxyFunc http.Transport.Proxy return proxy
func proxyFunc(req *http.Request) (*url.URL, error) {
	// 从上下文管理器中获取代理配置，实现代理和请求的一对一配置关系
	value := req.Context().Value(ctxKey("key")).(map[string]interface{})
	proxy, ok := value["proxy"]
	if !ok {
		return nil, nil
	}
	p := proxy.(*Proxy)
	// uri, err := url.Parse(p.HTTP)

	// If there is no proxy set, use default proxy from environment.
	// This mitigates expensive lookups on some platforms (e.g. Windows).
	envProxyOnce.Do(func() {
		// 从环境变量里面获取系统代理
		envProxyFuncValue = httpproxy.FromEnvironment().ProxyFunc()
	})

	// 根据目标站点的请求协议选择代理
	if req.URL.Scheme == "http" { // set proxy for http site
		if p.HTTP != "" {
			httpURL, err := url.Parse(p.HTTP)
			if err != nil {
				ERR := fmt.Sprint(ErrGetHttpProxy.Error(), err.Error())
				log.Error(ERR)
				return nil, errors.New(ERR)
			}
			return httpURL, nil
		}
	} else if req.URL.Scheme == "https" { // set proxy for https site
		if p.HTTPS != "" {
			httpsURL, err := url.Parse(p.HTTPS)
			if err != nil {
				ERR := fmt.Sprint(ErrGetHttpsProxy.Error(), err.Error())
				log.Error(ERR)
				return nil, errors.New(ERR)
			}
			return httpsURL, nil
		}

	}
	if p.SOCKS != "" {
		socksURL, err := url.Parse(p.SOCKS)
		if err != nil {
			return nil, errors.New(fmt.Sprint(ErrGetHttpsProxy.Error(), err.Error()))
		}
		return socksURL, nil
	}

	return envProxyFuncValue(req.URL)
}

// redirectFunc redirect handle funcation
// limit max redirect times
func redirectFunc(req *http.Request, via []*http.Request) error {
	redirectNum := req.Context().Value("redirectNum").(int)
	if len(via) > redirectNum {
		err := &RedirectError{redirectNum}
		return err
	}
	return nil
}

// StreamThreshold the must max size of response body  to use stream donload
func DownloaderWithStreamThreshold(streamThreshold uint64) DownloaderOption {
	return func(d *SpiderDownloader) {
		d.StreamThreshold = streamThreshold
	}

}

// DownloaderWithtransport download transport configure http.Transport
func DownloaderWithtransport(transport http.Transport) DownloaderOption {
	return func(d *SpiderDownloader) {
		d.transport = &transport
	}

}

// DownloadWithClient set http client for downloader
func DownloadWithClient(client http.Client) DownloaderOption {
	return func(d *SpiderDownloader) {
		d.client = &client
	}
}

// DownloadWithTimeout set request download timeout 
func DownloadWithTimeout(timeout time.Duration) DownloaderOption {
	return func(d *SpiderDownloader) {
		d.client.Timeout = timeout
	}
}

// DownloadWithTlsConfig set tls configure for downloader
func DownloadWithTlsConfig(tls tls.Config) DownloaderOption {
	return func(d *SpiderDownloader) {
		d.transport.TLSClientConfig = &tls

	}
}

// SpiderDownloader get a new spider downloader
func NewDownloader(opts ...DownloaderOption) Downloader {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
		Proxy: proxyFunc,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 3 * time.Second,
		MaxIdleConnsPerHost:   128,
		MaxConnsPerHost:       256,
	}
	newClient(http.Client{
		Transport:     transport,
		CheckRedirect: redirectFunc,
	})
	downloader := &SpiderDownloader{
		StreamThreshold: 1024 * 1024 * 10,
		transport:       transport,
		client:          globalClient,
	}
	for _, opt := range opts {
		opt(downloader)
	}
	return downloader
}

// checkUrlVaildate URL check validator
func checkUrlVaildate(requestUrl string) error {
	_, err := url.ParseRequestURI(requestUrl)
	return err

}

// CheckStatus check response status
func (d *SpiderDownloader) CheckStatus(statusCode int, allowStatus []int64) bool {
	if statusCode >= 400 && -1 == arrays.ContainsInt(allowStatus, int64(statusCode)) {
		return false
	}
	return true
}

// Download network downloader 
func (d *SpiderDownloader) Download(ctx context.Context, request *Request, result chan<- *RequestResult) {
	r := &RequestResult{}
	r.RequestId = request.RequestId
	downloadLog := log.WithField("request_id", request.RequestId)
	// record request handle start time
	now := time.Now()

	if err := checkUrlVaildate(request.Url); err != nil {
		// request url is not vaildate
		r.Response = nil
		r.Error = err
		result <- r
		return
	}

	// ValueContext 
	// The value carried by the context, mainly the proxy and the maximum number of redirects
	ctxValue := map[string]interface{}{}
	if request.Proxy != nil {

		ctxValue["proxy"] = request.Proxy

	}
	ctxValue["redirectNum"] = request.MaxRedirects

	// do set request params
	u, err := url.ParseRequestURI(request.Url)
	if err != nil {
		r.Error = fmt.Errorf(fmt.Sprintf("Parse url error %s", err.Error()))
		r.Response = nil
		result <- r
		return
	}

	if request.Params != nil {
		data := url.Values{}
		for k, v := range request.Params {
			data.Set(k, v)
		}
		u.RawQuery = data.Encode()

	}
	// Build the request here and pass in the context information
	var asCtxKey ctxKey = "key"
	ctx = context.WithValue(ctx, asCtxKey, ctxValue)
	req, err := http.NewRequestWithContext(ctx, request.Method, u.String(), request.BodyReader)
	if err != nil {
		r.Error = err
		r.Response = nil
		result <- r
		return
	}

	// Set request header
	for k, v := range request.Header {
		req.Header.Set(k, v)
	}

	// Set request cookie
	for k, v := range request.Cookies {
		req.AddCookie(&http.Cookie{
			Name:  k,
			Value: v,
		})
	}
	// Start request
	downloadLog.Debugf("Request %s is downloading", request.Url)
	resp, err := d.client.Do(req)
	if err != nil {
		r.Error = fmt.Errorf(fmt.Sprintf("Request url %s error %s", request.Url, err.Error()))
		r.Response = nil
		result <- r
		return

	}
	// Construct response body structure
	response := NewResponse()
	response.Header = resp.Header
	response.Status = resp.StatusCode
	response.Req = request
	response.URL = req.URL.String()
	response.Delay = time.Since(now).Seconds()
	if request.ResponseWriter != nil {
		// The response data is written into a custom io.Writer interface, 
		// such as a file in the file download process
		_, err = io.Copy(request.ResponseWriter, resp.Body)
	} else {
		// Response data is buffered to memory by default
		_, err = io.Copy(response.buffer, resp.Body)
		if err != nil {
			msg := fmt.Sprintf("%s %s", ErrResponseRead.Error(), err.Error())
			downloadLog.Errorf("%s\n", msg)
			r.Error = fmt.Errorf(msg)
			r.Response = nil
			result <- r
			return
		}

	}
	if err != nil {
		r.Error = fmt.Errorf("downloader io.copy failure error:%v", err)
		r.Response = nil
		result <- r
		return
	}
	r.Error = nil
	response.write()
	r.Response = response
	result <- r
	defer func() {
		if p := recover(); p != nil {
			downloadLog.Fatalf("Download handle error %v", p)
		}
		if resp.Body != nil {
			resp.Body.Close()
		}

	}()

}
