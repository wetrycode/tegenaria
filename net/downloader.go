package net

import (
	"context"
	"crypto/tls"

	logger "github.com/geebytes/Tegenaria/logging"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
)

type Downloader interface {
	Do(ctx context.Context, request *Request) (*Response, error)
	Download(ctx context.Context, request *Request) (*Response, error)
}

type SpiderDownloader struct{}

type Result struct {
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

var log *logrus.Entry = logger.GetLogger("downloader")
var GoSpiderDownloader *SpiderDownloader = &SpiderDownloader{}

func (d *SpiderDownloader) Download(ctx context.Context, request *Request, result chan Result) {
	r := Result{}
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer func() {
		// Release response resource
		fasthttp.ReleaseResponse(resp)
		// Release request resource
		fasthttp.ReleaseRequest(req)
		close(result)
		if p := recover(); p != nil {
			log.Errorf("panic recover! p: %v", p)
		}

	}()

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
	for {
		select {
		case <-ctx.Done():
			log.Infof("request is done")
			return
		default:
			if err := c.DoTimeout(req, resp, request.Timeout); err != nil {
				// log.Errorf("request %s error %s:", request.Url, err.Error())
				//return nil, err
				r.Response = nil
				r.Error = err
				result <- r
				return
			}
			b := resp.Body()
			var header map[string][]byte = make(map[string][]byte)

			resp.Header.VisitAll(func(key, value []byte) {
				header[string(key)] = value
			})
			r.Response = &Response{
				Text:   resp.String(),
				Status: resp.StatusCode(),
				Body:   b,
				Header: header,
				Req:    request,
			}
			log.Infof("Request is successful")
			r.Error = nil
			result <- r
		}
	}
}

func (d *SpiderDownloader) Do(ctx context.Context, request *Request, result chan Result) {
	d.Download(ctx, request, result)
}
