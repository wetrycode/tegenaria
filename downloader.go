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

// ctxKey WithValueContext key
type ctxKey string

// Downloader 下载器接口
type Downloader interface {
	// Download 下载函数
	Download(ctx *Context) (*Response, error)

	// CheckStatus 检查响应状态码的合法性
	CheckStatus(statusCode uint64, allowStatus []uint64) bool
}

// SpiderDownloader tegenaria 爬虫下载器
type SpiderDownloader struct {
	// transport http.Transport 用于设置连接和连接池
	transport *http.Transport
	// client http.Client 网络请求客户端
	client *http.Client
	// ProxyFunc 对单个请求进行代理设置
	ProxyFunc func(req *http.Request) (*url.URL, error)
}


// DownloaderOption 下载器可选参数函数
type DownloaderOption func(d *SpiderDownloader)

// Request 请求方式
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

// globalClient 全局网络请求客户端
var globalClient *http.Client = nil

// onceClient 客户端单例
var onceClient sync.Once

// envProxyOnce 系统代理环境变量加载单例
var envProxyOnce sync.Once

// envProxyFuncValue 代理环境变量获取逻辑
var envProxyFuncValue func(*url.URL) (*url.URL, error)

// newClient 初始化客户端
func newClient(client http.Client) {
	onceClient.Do(func() {
		if globalClient == nil {
			globalClient = &client
		}
	})
}

// proxyFunc http.Transport.Proxy 自定义的代理提取函数
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

// redirectFunc 网络跳转函数
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

// DownloaderWithtransport 为下载器设置 http.Transport
func DownloaderWithtransport(transport *http.Transport) DownloaderOption {
	return func(d *SpiderDownloader) {
		d.transport = transport
	}

}

// DownloadWithClient 设置下载器的http.Client客户端
func DownloadWithClient(client http.Client) DownloaderOption {
	return func(d *SpiderDownloader) {
		d.client = &client
	}
}

// DownloadWithTimeout 设置下载器的网络请求超时时间
func DownloadWithTimeout(timeout time.Duration) DownloaderOption {
	return func(d *SpiderDownloader) {
		d.client.Timeout = timeout
	}
}

// DownloadWithTlsConfig 设置下载器的tls
func DownloadWithTlsConfig(tls *tls.Config) DownloaderOption {
	return func(d *SpiderDownloader) {
		d.transport.TLSClientConfig = tls

	}
}

// DownloadWithTlsConfig 下载器是否开启http2
func DownloadWithH2(h2 bool) DownloaderOption {
	return func(d *SpiderDownloader) {
		d.transport.ForceAttemptHTTP2 = h2

	}
}

// NewDownloader 构建新的下载器
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

// checkUrlVaildate 检查url是否合法
func checkUrlVaildate(requestUrl string) error {
	_, err := url.ParseRequestURI(requestUrl)
	return err

}

// CheckStatus 检查状态码是否合法
func (d *SpiderDownloader) CheckStatus(statusCode uint64, allowStatus []uint64) bool {
	if len(allowStatus) == 0 {
		return true
	}
	if statusCode >= 400 && arrays.ContainsUint(allowStatus, statusCode) == -1 {
		return false
	}
	return true
}

// Download 处理request请求
func (d *SpiderDownloader) Download(ctx *Context) (*Response, error) {
	downloadLog := log.WithField("request_id", ctx.CtxId)
	defer func() {
		if err := recover(); err != nil {
			downloadLog.Fatalf("download panic: %v", err)
		}

	}()
	// 记录网络请求处理开始时间
	now := time.Now()

	if err := checkUrlVaildate(ctx.Request.Url); err != nil {
		// url不合法
		downloadLog.Errorf(err.Error())
		return nil, err
	}

	// ValueContext
	// 记录代理地址和最大的跳转次数
	ctxValue := map[string]interface{}{}
	if ctx.Request.Proxy != nil {

		ctxValue["proxy"] = ctx.Request.Proxy

	}
	ctxValue["redirectNum"] = ctx.Request.MaxRedirects

	// 动态更新url请求参数
	u, _ := url.ParseRequestURI(ctx.Request.Url)

	if ctx.Request.Params != nil {
		data := url.Values{}
		for k, v := range ctx.Request.Params {
			data.Set(k, v)
		}
		u.RawQuery = data.Encode()

	}
	// 构建网络请求上下文
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

	// 设置请求头
	for k, v := range ctx.Request.Header {
		req.Header.Set(k, v)
	}

	// 设置cookie
	for k, v := range ctx.Request.Cookies {
		req.AddCookie(&http.Cookie{
			Name:  k,
			Value: v,
		})
	}
	// 发起请求
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
	// 构建响应
	response := NewResponse()
	response.Header = resp.Header
	response.Status = resp.StatusCode
	response.URL = req.URL.String()
	response.Delay = time.Since(now).Seconds()
	response.ContentLength = uint64(resp.ContentLength)

	if ctx.Request.ResponseWriter != nil {
		// 通过自定义的io.Writer接口读取响应数据
		// 例如大文件下载
		_, err = io.Copy(ctx.Request.ResponseWriter, resp.Body)
		if err == io.EOF {
			err = nil
		}
	} else {
		// 响应数据写入缓存
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
