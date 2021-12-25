package tegenaria

import (
	"bytes"
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
	StreamThreshold uint64 // StreamThreshold 采用流式传输的响应体阈值 TODO

	transport *http.Transport // transport 下载器使用的transport，每一个请求采用公共的网络传输配置，全局使用一个连接池

	client *http.Client // client 网络请求客户端

	ProxyFunc func(req *http.Request) (*url.URL, error) // ProxyFunc代理供应函数
}

// RequestResult 网络请求响应结果
type RequestResult struct {

	Error error // Error 各类请求过程中的异常

	Response *Response // Response 网络请求响应对象
}

// DownloaderOption 下载器的可选参数
type DownloaderOption func(d *SpiderDownloader)

// 请求方式常量定义
const (
	GET     string = "GET"
	POST    string = "POST"
	PUT     string = "PUT"
	DELETE  string = "DELETE"
	OPTIONS string = "OPTIONS"
	HEAD    string = "HEAD"
)

// log 下载模块的日志记录
var log *logrus.Entry = GetLogger("downloader")

// globalClient 全局的网络请求客户端
var globalClient *http.Client = nil

// onceClient 客户端单例
var onceClient sync.Once

// envProxyOnce 本地系统代理加载
var envProxyOnce sync.Once

// envProxyFuncValue 本地系统环境加载函数
var envProxyFuncValue func(*url.URL) (*url.URL, error)

// newClient 获取新的客户端，全局只实例化一个client
func newClient(client http.Client) {
	onceClient.Do(func() {
		if globalClient == nil {
			globalClient = &client
		}
	})
}

// proxyFunc http.Transport.Proxy 代理构建函数
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

// redirectFunc 重定向处理函数
// 限制最大的重定向次数
func redirectFunc(req *http.Request, via []*http.Request) error {
	redirectNum := req.Context().Value("redirectNum").(int)
	if len(via) > redirectNum {
		err := &RedirectError{redirectNum}
		return err
	}
	return nil
}

// StreamThreshold 采用流式传输的响应体阈值 TODO
func DownloaderWithStreamThreshold(streamThreshold uint64) DownloaderOption {
	return func(d *SpiderDownloader) {
		d.StreamThreshold = streamThreshold
	}

}

// DownloaderWithtransport 为下载器提供自定义配置的 http.Transport
func DownloaderWithtransport(transport http.Transport) DownloaderOption {
	return func(d *SpiderDownloader) {
		d.transport = &transport
	}

}

// DownloadWithClient 为下载器提供自定义的网络请求客户端
func DownloadWithClient(client http.Client) DownloaderOption {
	return func(d *SpiderDownloader) {
		d.client = &client
	}
}

// DownloadWithTimeout 设置下载的超时时间
func DownloadWithTimeout(timeout time.Duration) DownloaderOption {
	return func(d *SpiderDownloader) {
		d.client.Timeout = timeout
	}
}

// DownloadWithTlsConfig 设置tls请求的配置参数
func DownloadWithTlsConfig(tls tls.Config) DownloaderOption {
	return func(d *SpiderDownloader) {
		d.transport.TLSClientConfig = &tls

	}
}

// SpiderDownloader 构造函数
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

// checkUrlVaildate URL合法性校验器
func checkUrlVaildate(requestUrl string) error {
	_, err := url.ParseRequestURI(requestUrl)
	return err

}

// CheckStatus 响应状态码合法性校验
func (d *SpiderDownloader) CheckStatus(statusCode int, allowStatus []int64) bool {
	if statusCode > 400 && -1 == arrays.ContainsInt(allowStatus, int64(statusCode)) {
		return false
	}
	return true
}

// Download 下载的主逻辑，构建请求并处理响应
// 传入一个上下文管理，将下载结果发送到*RequestResult channel
func (d *SpiderDownloader) Download(ctx context.Context, request *Request, result chan<- *RequestResult) {
	r := &RequestResult{}

	// 整个请求的延迟开始计算
	now := time.Now()

	if err := checkUrlVaildate(request.Url); err != nil {
		// 校验URL的合法性
		r.Response = nil
		r.Error = err
		result <- r
		return
	}

	// ValueContext上下文携带的值，主要是代理和最大的重定向次数
	ctxValue := map[string]interface{}{}
	if request.Proxy != nil {

		ctxValue["proxy"] = request.Proxy

	}
	ctxValue["redirectNum"] = request.MaxRedirectNum

	// 配置请求参数
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
	// 此处构建请求并传入上下文信息
	var asCtxKey ctxKey = "key"
	ctx = context.WithValue(ctx, asCtxKey, ctxValue)
	req, err := http.NewRequestWithContext(ctx, request.Method, u.String(), request.BodyReader)
	if err != nil {
		r.Error = err
		r.Response = nil
		result <- r
		return
	}

	// 设置请求头
	for k, v := range request.Header {
		req.Header.Set(k, v)
	}

	// 设置请求cookie
	for k, v := range request.Cookies {
		req.AddCookie(&http.Cookie{
			Name:  k,
			Value: v,
		})
	}
	// Start request
	resp, err := d.client.Do(req)
	if err != nil {
		r.Error = fmt.Errorf(fmt.Sprintf("Request url %s error %s", request.Url, err.Error()))
		r.Response = nil
		result <- r
		return

	}
	// 构造响应体结构
	response := NewResponse()
	response.Header = resp.Header
	response.Status = resp.StatusCode
	response.Req = request
	response.Delay = time.Since(now).Seconds()
	// 从缓冲池中获取缓冲对象，用于读取响应体
	buffer := bufferPool.Get().(*bytes.Buffer)
	buffer.Reset()
	if request.ResponseWriter != nil {
		// 响应数据写入自定义的io.Writer接口，例如文件下载过程的文件
		_, err = io.Copy(request.ResponseWriter, resp.Body)
	} else {
		// 默认缓冲到内存
		_, err = io.Copy(buffer, resp.Body)
		response.Body = buffer.Bytes()

	}
	if err != nil {
		r.Error = fmt.Errorf("downloader io.copy failure error:%v", err)
		r.Response = nil
		result <- r
		return
	}
	r.Error = nil
	r.Response = response
	result <- r
	// 缓冲对象放回缓冲池
	bufferPool.Put(buffer)
	buffer = nil
	defer func() {
		if buffer != nil {
			resp.Body.Close()
			bufferPool.Put(buffer)
			buffer = nil
		}

	}()

}
