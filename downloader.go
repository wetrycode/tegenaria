package tegenaria

import (
	"context"
	"crypto/tls"
	"net/url"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
	"github.com/wxnacy/wgo/arrays"
)

type Downloader interface {
	Download(ctx context.Context, request *Request, result chan<- *RequestResult)
	CheckStatus(statusCode int64, allowStatus []int64) bool
}

type SpiderDownloader struct{}

type RequestResult struct {
	Error    error
	Response *Response
}

const (
	GET     string = "GET"
	POST    string = "POST"
	PUT     string = "PUT"
	DELETE  string = "DELETE"
	OPTIONS string = "OPTIONS"
)

var log *logrus.Entry = GetLogger("downloader")
var GoSpiderDownloader *SpiderDownloader = &SpiderDownloader{}

func checkUrlVaildate(requestUrl string) error {
	_, err := url.ParseRequestURI(requestUrl)
	return err

}
func (d *SpiderDownloader) CheckStatus(statusCode int64, allowStatus []int64) bool {
	if statusCode > 400 && -1 == arrays.ContainsInt(allowStatus, statusCode) {
		return false
	}
	return true
}
func (d *SpiderDownloader) Download(ctx context.Context, request *Request, result chan<- *RequestResult) {
	now := time.Now()
	_, cancel := context.WithCancel(ctx)
	r := &RequestResult{}
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer func() {
		// Release response resource
		fasthttp.ReleaseResponse(resp)
		// Release request resource
		fasthttp.ReleaseRequest(req)
		// close(result)
		if p := recover(); p != nil {
			log.Errorf("panic recover! p: %v", p)
		}
		cancel()

	}()
	if err := checkUrlVaildate(request.Url); err != nil {
		r.Response = nil
		r.Error = err
		result <- r
		return
	}
	req.Header.SetMethod(request.Method)
	req.SetRequestURI(request.Url)
	// Set request body
	if len(request.Body) != 0 {
		req.SetBody(request.Body)
	}
	// Set request headers
	for key, value := range request.Header {
		req.Header.Set(key, value)
	}
	// Set tls设置
	c := &fasthttp.Client{
		TLSConfig: &tls.Config{
			InsecureSkipVerify: request.TLS,
		},
		MaxConnDuration:    request.Timeout,
		ReadTimeout:        request.Timeout,
		MaxConnWaitTimeout: request.Timeout,
	}
	// set cookies
	if len(request.Cookies) != 0 {
		for key, value := range request.Cookies {
			req.Header.SetCookie(key, value)

		}
	}
	// Set request proxy without protocol
	if len(request.Proxy) != 0 {
		c.Dial = fasthttpproxy.FasthttpHTTPDialer(request.Proxy)
	}
	if request.AllowRedirects {
		if err := c.DoRedirects(req, resp, request.MaxRedirects); err != nil {
			r.Response = nil
			r.Error = err
			result <- r
			return
		}
	} else {
		if err := c.DoTimeout(req, resp, request.Timeout); err != nil {
			r.Response = nil
			r.Error = err
			result <- r
			return
		}
	}

	b := resp.Body()
	var header map[string][]byte = make(map[string][]byte)

	resp.Header.VisitAll(func(key, value []byte) {
		header[string(key)] = value
	})
	r.Response = &Response{
		Text:          resp.String(),
		Status:        resp.StatusCode(),
		Body:          b,
		Header:        header,
		Req:           request,
		Delay:         time.Since(now).Seconds(),
		ContentLength: resp.Header.ContentLength(),
	}
	log.Infof("Request %s is successful", request.Url)
	r.Error = nil
	result <- r

}

func (d *SpiderDownloader) Do(ctx context.Context, request *Request, result chan *RequestResult) {
	d.Download(ctx, request, result)
}
