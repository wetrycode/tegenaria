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
	// Download core funcation
	Download(ctx *Context) (*Response, error)

	// CheckStatus check response status code if allow handle
	CheckStatus(statusCode uint64, allowStatus []uint64) bool
}

// SpiderDownloader tegenaria spider downloader
type SpiderDownloader struct {
	// transport The transport used by the downloader,
	// each request adopts a public network transmission configuration,
	// and a connection pool is used globally
	transport *http.Transport
	// client network request client
	client *http.Client
	// ProxyFunc update proxy for per request
	ProxyFunc func(req *http.Request) (*url.URL, error)
	// RateLimiter ratelimit.Limiter
}

// RequestResult network request response result
type RequestResult struct {
	Error    *HandleError // Error error exception during request
	Response *Response    // Response network request response object
}

// DownloaderOption optional parameters of the downloader
type DownloaderOption func(d *SpiderDownloader)

// Request method constant definition
type RequestMethod string

const (
	GET     RequestMethod = "GET"
	POST    RequestMethod = "POST"
	PUT     RequestMethod = "PUT"
	DELETE  RequestMethod = "DELETE"
	OPTIONS RequestMethod = "OPTIONS"
	HEAD    RequestMethod = "HEAD"
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

// func timeoutDialContext(ctx context.Context, network, addr string) (net.Conn, error){

// }

// proxyFunc http.Transport.Proxy return proxy
func proxyFunc(req *http.Request) (*url.URL, error) {
	// 从上下文管理器中获取代理配置，实现代理和请求的一对一配置关系
	value := req.Context().Value(ctxKey("key")).(map[string]interface{})
	proxy, ok := value["proxy"]
	if !ok {
		return nil, nil
	}
	p := proxy.(*Proxy)
	// If there is no proxy set, use default proxy from environment.
	// This mitigates expensive lookups on some platforms (e.g. Windows).
	envProxyOnce.Do(func() {
		// get proxies from system env
		envProxyFuncValue = httpproxy.FromEnvironment().ProxyFunc()
	})
	if p != nil && p.ProxyUrl != "" {
		proxyURL, err := urlParse(p.ProxyUrl)
		if err != nil {
			err := fmt.Sprint(ErrGetHttpProxy.Error(), err.Error())
			log.Error(err)
			return nil, errors.New(err)
		}
		return proxyURL, nil
	}
	return envProxyFuncValue(req.URL)
}

// redirectFunc redirect handle funcation
// limit max redirect times
func redirectFunc(req *http.Request, via []*http.Request) error {
	value := req.Context().Value(ctxKey("key")).(map[string]interface{})
	redirectNum := value["redirectNum"].(int)
	if len(via) > redirectNum {
		err := &RedirectError{redirectNum}
		return err
	}
	return nil
}
func urlParse(URL string) (*url.URL, error) {
	return url.Parse(URL)
}

// DownloaderWithtransport download transport configure http.Transport
func DownloaderWithtransport(transport *http.Transport) DownloaderOption {
	return func(d *SpiderDownloader) {
		d.transport = transport
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
func DownloadWithTlsConfig(tls *tls.Config) DownloaderOption {
	return func(d *SpiderDownloader) {
		d.transport.TLSClientConfig = tls

	}
}

// DownloadWithTlsConfig set tls configure for downloader
func DownloadWithH2(h2 bool) DownloaderOption {
	return func(d *SpiderDownloader) {
		d.transport.ForceAttemptHTTP2 = h2

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
			Timeout:   10 * time.Second,
			KeepAlive: 1 * 60 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     false,
		MaxIdleConns:          1024,
		IdleConnTimeout:       10 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 10 * time.Second,
		MaxIdleConnsPerHost:   1024,
		MaxConnsPerHost:       1024,
	}
	newClient(http.Client{
		Transport:     transport,
		CheckRedirect: redirectFunc,
	})
	downloader := &SpiderDownloader{
		transport: transport,
		client:    globalClient,
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
func (d *SpiderDownloader) CheckStatus(statusCode uint64, allowStatus []uint64) bool {
	if len(allowStatus) == 0 {
		return true
	}
	if statusCode >= 400 && arrays.ContainsUint(allowStatus, statusCode) == -1 {
		return false
	}
	return true
}

// Download network downloader
func (d *SpiderDownloader) Download(ctx *Context) (*Response, error) {
	downloadLog := log.WithField("request_id", ctx.CtxId)
	defer func() {
		// result <- ctx
		if err := recover(); err != nil {
			downloadLog.Fatalf("download panic: %v", err)
		}

	}()
	// record request handle start time
	now := time.Now()

	if err := checkUrlVaildate(ctx.Request.Url); err != nil {
		// request url is not vaildate
		// downloadErr := NewError(ctx, err, ErrorWithRequest(ctx.Request))
		downloadLog.Errorf(err.Error())
		return nil, err
	}

	// ValueContext
	// The value carried by the context, mainly the proxy and the maximum number of redirects
	ctxValue := map[string]interface{}{}
	if ctx.Request.Proxy != nil {

		ctxValue["proxy"] = ctx.Request.Proxy

	}
	ctxValue["redirectNum"] = ctx.Request.MaxRedirects

	// do set request params
	u, _ := url.ParseRequestURI(ctx.Request.Url)

	if ctx.Request.Params != nil {
		data := url.Values{}
		for k, v := range ctx.Request.Params {
			data.Set(k, v)
		}
		u.RawQuery = data.Encode()

	}
	// Build the request here and pass in the context information
	var asCtxKey ctxKey = "key"
	var timeoutCtx context.Context = nil
	var valCtx context.Context = nil
	if ctx.Request.Timeout > 0 {
		timeoutCtx, _ = context.WithTimeout(ctx, ctx.Request.Timeout)
		valCtx = context.WithValue(timeoutCtx, asCtxKey, ctxValue)
	} else {
		valCtx = context.WithValue(ctx, asCtxKey, ctxValue)
	}
	req, err := http.NewRequestWithContext(valCtx, string(ctx.Request.Method), u.String(), ctx.Request.BodyReader)
	if err != nil {
		downloadLog.Errorf(fmt.Sprintf("Create request error %s", err.Error()))
		return nil, err
	}

	// Set request header
	for k, v := range ctx.Request.Header {
		req.Header.Set(k, v)
	}

	// Set request cookie
	for k, v := range ctx.Request.Cookies {
		req.AddCookie(&http.Cookie{
			Name:  k,
			Value: v,
		})
	}
	// Start request
	downloadLog.Debugf("Downloader %s is downloading", ctx.Request.Url)
	resp, err := d.client.Do(req)
	defer func() {
		if resp != nil && resp.Body != nil {

			resp.Body.Close()

		}
		req.Close = true
	}()
	if err != nil {
		downloadErr := NewError(ctx, fmt.Errorf("Request url %s error %s when reading response", ctx.Request.Url, err.Error()))
		return nil, downloadErr

	}
	// Construct response body structure
	response := NewResponse()
	response.Header = resp.Header
	response.Status = resp.StatusCode
	response.URL = req.URL.String()
	response.Delay = time.Since(now).Seconds()
	response.ContentLength = uint64(resp.ContentLength)

	if ctx.Request.ResponseWriter != nil {
		// The response data is written into a custom io.Writer interface,
		// such as a file in the file download process
		_, err = io.Copy(ctx.Request.ResponseWriter, resp.Body)
		if err == io.EOF {
			err = nil
		}
	} else {
		// Response data is buffered to memory by default
		_, err = io.Copy(response.Buffer, resp.Body)
		if err == io.EOF {
			err = nil
		}

	}
	if err != nil {
		msg := fmt.Sprintf("%s %s", ErrResponseRead.Error(), err.Error())
		downloadLog.Errorf("%s\n", msg)

		return nil, err
	}
	return response, nil
}
