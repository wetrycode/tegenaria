package tegenaria

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-kiss/monkey"
)

var onceServer sync.Once
var onceProxyServer sync.Once
var ts *httptest.Server
var proxyServer *httptest.Server

func testParser(resp *Context, item chan<- *ItemMeta, req chan<- *Context) error {
	newItem := &testItem{
		test:      "test",
		pipelines: make([]int, 0),
	}
	item <- NewItem(resp, newItem)
	return nil
}

// func doTest(request *Request)(Response, Error, context.CancelFunc){
// 	return nil, nil, nil

// }
func newTestProxyServer() *httptest.Server {
	onceProxyServer.Do(func() {
		gin.SetMode(gin.ReleaseMode)
		router := gin.New()
		router.GET("/:a", func(c *gin.Context) {
			reqUrl := c.Request.URL.String() // Get request url
			req, err := http.NewRequest(c.Request.Method, reqUrl, nil)
			if err != nil {
				c.AbortWithStatus(404)
				return
			}

			// Forwarding requests from client.
			cli := &http.Client{}
			resp, err := cli.Do(req)
			if err != nil {
				c.AbortWithStatus(404)
				return
			}
			defer resp.Body.Close()

			body, _ := ioutil.ReadAll(resp.Body)
			c.Data(200, "text/plain", body)        // write response body to response.
			c.String(200, "This is proxy Server.") // add proxy info.
		})
		proxyServer = httptest.NewServer(router)
	})
	return proxyServer
}
func newTestServer() *httptest.Server {
	onceServer.Do(func() {
		gin.SetMode(gin.ReleaseMode)
		router := gin.New()
		router.GET("/testGET", func(c *gin.Context) {
			c.String(200, "GET")
		})
		router.POST("/testPOST", func(c *gin.Context) {
			dataType, _ := json.Marshal(map[string]string{"key": "value"})
			data, err := c.GetRawData()
			if err != nil {
				c.AbortWithStatus(404)
				return
			}
			if string(data) == string(dataType) {

				c.String(200, "POST")
			} else {
				c.String(200, string(data))
			}
		})
		router.GET("/testGetCookie", func(c *gin.Context) {
			cookies := c.Request.Cookies()

			for _, cookie := range cookies {
				http.SetCookie(c.Writer, &http.Cookie{
					Name:    cookie.Name,
					Value:   cookie.Value,
					Path:    "/",
					Expires: time.Now().Add(30 * time.Second),
				})
			}
			c.String(200, "cookie")

		})
		router.GET("/testHeader", func(c *gin.Context) {
			header := c.GetHeader("key")
			c.String(200, header)
		})
		router.GET("/testTimeout", func(c *gin.Context) {
			time.Sleep(5 * time.Second)
			c.String(200, "timeout")
		})
		router.GET("/test403", func(c *gin.Context) {
			c.String(403, "forbidden")
		})
		router.GET("/testParams", func(c *gin.Context) {
			value := c.Query("key")
			c.String(200, value)
		})
		router.GET("/testRedirect1", func(c *gin.Context) {
			c.Redirect(http.StatusMovedPermanently, "/testRedirect2")
		})
		router.GET("/testRedirect2", func(c *gin.Context) {
			c.Redirect(http.StatusMovedPermanently, "/testRedirect3")
		})
		router.GET("/testRedirect3", func(c *gin.Context) {
			c.String(200, "/testRedirect3")
		})
		router.GET("/testJson", func(c *gin.Context) {
			data := gin.H{
				"name": "json",
				"msg":  "hello world",
				"age":  18,
			}
			c.JSON(200, data)
		})
		router.GET("/proxy", func(c *gin.Context) {
			c.String(200, "This is target website.")
		})
		router.GET("/testFile", func(c *gin.Context) {
			testString := make([]string, 200000)
			for i := 0; i < 200000; i++ {
				testString = append(testString, "This test files")

			}
			c.String(200, "", testString)
		})

		ts = httptest.NewServer(router)
	})
	return ts

}
func TestRequestGet(t *testing.T) {
	server := newTestServer()

	request := NewRequest(server.URL+"/testGET", GET, testParser)
	var MainCtx context.Context = context.Background()
	cancelCtx, cancel := context.WithCancel(MainCtx)

	ctx := NewContext(request, WithContext(cancelCtx))

	defer func() {
		cancel()
	}()
	resultChan := make(chan *Context, 1)

	downloader := NewDownloader(DownloadWithTlsConfig(&tls.Config{InsecureSkipVerify: true}))

	downloader.Download(ctx, resultChan)
	result := <-resultChan
	err := result.DownloadResult.Error
	resp := result.DownloadResult.Response
	if err != nil {
		t.Errorf("request error")

	}
	if resp.Status != 200 {
		t.Errorf("response status = %d; expected %d", resp.Status, 200)

	}
	if resp.String() != "GET" {
		t.Errorf("response text = %s; expected %s", resp.String(), "GET")

	}

}

func TestRequestPost(t *testing.T) {
	body := map[string]interface{}{
		"key": "value",
	}
	server := newTestServer()

	request := NewRequest(server.URL+"/testPOST", POST, testParser, RequestWithRequestBody(body))
	var MainCtx context.Context = context.Background()

	cancelCtx, cancel := context.WithCancel(MainCtx)

	ctx := NewContext(request, WithContext(cancelCtx))

	defer func() {
		cancel()
	}()
	resultChan := make(chan *Context, 1)
	downloader := NewDownloader()

	downloader.Download(ctx, resultChan)
	result := <-resultChan
	err := result.DownloadResult.Error
	resp := result.DownloadResult.Response
	if err != nil {
		t.Errorf("request error %s", err.Error())

	}
	if resp.Status != 200 {
		t.Errorf("response status = %d; expected %d", resp.Status, 200)

	}
	if resp.String() != "POST" {
		t.Errorf("response text = %s; expected %s", resp.String(), "POST")

	}

}

func TestRequestCookie(t *testing.T) {
	cookies := map[string]string{
		"test1": "test1",
		"test2": "test2",
	}
	server := newTestServer()
	// downloader := NewDownloader()

	request := NewRequest(server.URL+"/testGetCookie", GET, testParser, RequestWithRequestCookies(cookies))
	var MainCtx context.Context = context.Background()

	cancelCtx, cancel := context.WithCancel(MainCtx)

	ctx := NewContext(request, WithContext(cancelCtx))
	// var MainCtx context.Context = context.Background()

	defer func() {
		cancel()
	}()
	resultChan := make(chan *Context, 1)
	downloader := NewDownloader()

	downloader.Download(ctx, resultChan)
	result := <-resultChan
	err := result.DownloadResult.Error
	resp := result.DownloadResult.Response

	if err != nil {
		t.Errorf("request error with cookies")

	}
	if resp.Status != 200 {
		t.Errorf("response with cookies status = %d; expected %d", resp.Status, 200)

	}
	if len(resp.Header["Set-Cookie"]) != 2 {
		t.Errorf("request with cookies get = %d; expected %d", len(resp.Header["Set-Cookie"]), 2)
	}
}

func TestRequestQueryParams(t *testing.T) {
	params := map[string]string{
		"key": "value",
	}
	server := newTestServer()
	downloader := NewDownloader()

	request := NewRequest(server.URL+"/testParams", GET, testParser, RequestWithRequestParams(params))
	var MainCtx context.Context = context.Background()

	cancelCtx, cancel := context.WithCancel(MainCtx)

	ctx := NewContext(request, WithContext(cancelCtx))
	// var MainCtx context.Context = context.Background()

	defer func() {
		cancel()
	}()
	resultChan := make(chan *Context, 1)

	downloader.Download(ctx, resultChan)
	result := <-resultChan
	err := result.DownloadResult.Error
	resp := result.DownloadResult.Response
	if err != nil {
		t.Errorf("request error")

	}
	if resp.Status != 200 {
		t.Errorf("response with cookies status = %d; expected %d", resp.Status, 200)

	}
	if resp.String() != "value" {
		t.Errorf("request with params get = %s; expected %s", resp.String(), "value")

	}

}

func TestRequestProxy(t *testing.T) {
	server := newTestServer()
	proxyServer := newTestProxyServer()

	downloader := NewDownloader()
	proxy := Proxy{
		ProxyUrl: proxyServer.URL,
	}

	defer proxyServer.Close()
	request := NewRequest(server.URL+"/proxy", GET, testParser, RequestWithRequestProxy(proxy))
	var MainCtx context.Context = context.Background()

	cancelCtx, cancel := context.WithCancel(MainCtx)

	ctx := NewContext(request, WithContext(cancelCtx))
	// var MainCtx context.Context = context.Background()

	defer func() {
		cancel()
	}()
	resultChan := make(chan *Context, 1)

	downloader.Download(ctx, resultChan)
	result := <-resultChan
	err := result.DownloadResult.Error
	resp := result.DownloadResult.Response
	if err != nil {
		t.Errorf("request error")

	}
	if resp.Status != 200 {
		t.Errorf("response status = %d; expected %d", resp.Status, 200)

	}
	if resp.String() != "This is target website.This is proxy Server." {
		t.Errorf("response text = %s; expected %s", resp.String(), "This is target website.This is proxy Server.")

	}

}

func TestRequestHeaders(t *testing.T) {
	server := newTestServer()
	downloader := NewDownloader()

	headers := map[string]string{
		"KEY":        "value",
		"Intparams":  "1",
		"Boolparams": "false",
	}
	request := NewRequest(server.URL+"/testHeader", GET, testParser, RequestWithRequestHeader(headers))
	var MainCtx context.Context = context.Background()

	cancelCtx, cancel := context.WithCancel(MainCtx)

	ctx := NewContext(request, WithContext(cancelCtx))
	// var MainCtx context.Context = context.Background()

	defer func() {
		cancel()
	}()
	resultChan := make(chan *Context, 1)

	go downloader.Download(ctx, resultChan)
	result := <-resultChan
	err := result.DownloadResult.Error
	resp := result.DownloadResult.Response
	if err != nil {
		t.Errorf("request error")

	}
	if resp.Status != 200 {
		t.Errorf("response status = %d; expected %d", resp.Status, 200)

	}
	if resp.String() != "value" {
		t.Errorf("request with headers get = %s; expected %s", resp.String(), "value")

	}
}

func TestTimeout(t *testing.T) {
	server := newTestServer()
	downloader := NewDownloader(DownloadWithTimeout(1 * time.Second))

	request := NewRequest(server.URL+"/testTimeout", GET, testParser)
	var MainCtx context.Context = context.Background()

	cancelCtx, cancel := context.WithCancel(MainCtx)

	ctx := NewContext(request, WithContext(cancelCtx))
	// var MainCtx context.Context = context.Background()

	defer func() {
		cancel()
	}()
	resultChan := make(chan *Context, 1)

	downloader.Download(ctx, resultChan)
	result := <-resultChan
	err := result.DownloadResult.Error
	resp := result.DownloadResult.Response
	if err == nil {
		t.Errorf("request timeout expected error but no any errors")

	}
	if resp != nil {
		t.Errorf("response = %v; expected  nil", resp)

	}
}
func TestLargeFile(t *testing.T) {
	server := newTestServer()
	file, _ := os.Create("test.file")
	// writer := bufio.NewWriter(file)
	defer os.Remove("test.file")
	defer file.Close()
	downloader := NewDownloader(DownloadWithTimeout(1 * time.Second))
	request := NewRequest(server.URL+"/testFile", GET, testParser, RequestWithResponseWriter(file))
	var MainCtx context.Context = context.Background()

	cancelCtx, cancel := context.WithCancel(MainCtx)

	ctx := NewContext(request, WithContext(cancelCtx))
	// var MainCtx context.Context = context.Background()

	defer func() {
		cancel()
	}()
	resultChan := make(chan *Context, 1)

	downloader.Download(ctx, resultChan)
	result := <-resultChan
	err := result.DownloadResult.Error
	resp := result.DownloadResult.Response
	if err != nil || resp == nil {
		t.Errorf("request error %s", err.Error())

	}
	if resp.Status != 200 {
		t.Errorf("response status = %d; expected %d", resp.Status, 200)

	}
	fi, statErr := os.Stat("test.file")
	if statErr != nil {
		t.Errorf("get test.file info fail %s", err)

	}
	if fi.Size() == 0 {
		t.Errorf("get test.file size 0")

	}
}

func TestJsonResponse(t *testing.T) {
	server := newTestServer()
	downloader := NewDownloader()
	request := NewRequest(server.URL+"/testJson", GET, testParser)
	var MainCtx context.Context = context.Background()

	cancelCtx, cancel := context.WithCancel(MainCtx)

	ctx := NewContext(request, WithContext(cancelCtx))
	// var MainCtx context.Context = context.Background()

	defer func() {
		cancel()
	}()
	resultChan := make(chan *Context, 1)

	downloader.Download(ctx, resultChan)
	result := <-resultChan
	err := result.DownloadResult.Error
	resp := result.DownloadResult.Response
	if err != nil {
		t.Errorf("request json error %s", err.Error())

	}
	if resp.Status != 200 {
		t.Errorf("response status = %d; expected %d", resp.Status, 200)

	}
	if resp.Json()["name"] != "json" {
		t.Errorf("request with headers get = %s; expected %s", resp.String(), "json")

	}
}
func TestRedirectLimit(t *testing.T) {
	server := newTestServer()
	downloader := NewDownloader()
	request := NewRequest(server.URL+"/testRedirect1", GET, testParser, RequestWithMaxRedirects(1))
	var MainCtx context.Context = context.Background()

	cancelCtx, cancel := context.WithCancel(MainCtx)

	ctx := NewContext(request, WithContext(cancelCtx))

	defer func() {
		cancel()
	}()
	resultChan := make(chan *Context, 1)

	downloader.Download(ctx, resultChan)
	result := <-resultChan
	err := result.DownloadResult.Error
	resp := result.DownloadResult.Response
	if err == nil {
		t.Errorf("request redirect should be limit error\n")

	}
	if resp != nil {
		t.Errorf("response is not empty ")

	}
}

func TestNotAllowRedirect(t *testing.T) {
	server := newTestServer()
	downloader := NewDownloader()
	request := NewRequest(server.URL+"/testRedirect1", GET, testParser, RequestWithAllowRedirects(false))
	var MainCtx context.Context = context.Background()

	cancelCtx, cancel := context.WithCancel(MainCtx)

	ctx := NewContext(request, WithContext(cancelCtx))

	defer func() {
		cancel()
	}()
	resultChan := make(chan *Context, 1)

	downloader.Download(ctx, resultChan)
	result := <-resultChan
	err := result.DownloadResult.Error
	resp := result.DownloadResult.Response
	if !strings.Contains(err.Error(), "maximum number of redirects") {
		t.Errorf("Except error exceeded the maximum number of redirects,but get %s\n", err.Error())
	}

	if resp != nil {
		t.Errorf("response is not empty ")

	}
}

func TestInvalidURL(t *testing.T) {
	downloader := NewDownloader()
	request := NewRequest("error"+"/testRedirect1", GET, testParser)
	var MainCtx context.Context = context.Background()

	cancelCtx, cancel := context.WithCancel(MainCtx)

	ctx := NewContext(request, WithContext(cancelCtx))

	defer func() {
		cancel()
	}()
	resultChan := make(chan *Context, 1)

	downloader.Download(ctx, resultChan)
	result := <-resultChan
	err := result.DownloadResult.Error
	if err == nil {
		t.Errorf("request invlid url should get an error\n")

	}
}

func TestResponseReadError(t *testing.T) {
	server := newTestServer()

	request := NewRequest(server.URL+"/testGET", GET, testParser)
	var MainCtx context.Context = context.Background()
	cancelCtx, cancel := context.WithCancel(MainCtx)

	ctx := NewContext(request, WithContext(cancelCtx))

	defer func() {
		cancel()
	}()
	resultChan := make(chan *Context, 1)
	monkey.Patch(io.Copy, func(dst io.Writer, src io.Reader) (written int64, err error) {
		return 0, errors.New("empty buffer in CopyBuffer")
	})

	downloader := NewDownloader()

	downloader.Download(ctx, resultChan)
	result := <-resultChan
	err := result.DownloadResult.Error
	// resp := result.DownloadResult.Response
	msg := err.Error()
	if !strings.Contains(msg, "read response to buffer error") {
		t.Errorf("request should have error read response to buffer error,but get %s\n", err.Error())

	}
}

func TestProxyUrlError(t *testing.T) {
	server := newTestServer()
	proxyServer := newTestProxyServer()

	proxy := Proxy{
		ProxyUrl: "error",
	}

	monkey.Patch(urlParse, func(string) (url *url.URL, err error) {
		return nil, errors.New("proxy invail url")
	})
	defer proxyServer.Close()
	request := NewRequest(server.URL+"/proxy", GET, testParser, RequestWithRequestProxy(proxy))
	var MainCtx context.Context = context.Background()
	cancelCtx, cancel := context.WithCancel(MainCtx)

	ctx := NewContext(request, WithContext(cancelCtx))
	defer func() {
		cancel()
	}()
	resultChan := make(chan *Context, 1)

	downloader := NewDownloader()

	downloader.Download(ctx, resultChan)
	result := <-resultChan
	err := result.DownloadResult.Error
	if err.Error() == "proxy invail url" {
		t.Errorf("request should have error proxy invail url, but get %s\n", err.Error())

	}
}

func TestDownloaderRequestConextError(t *testing.T) {
	server := newTestServer()

	request := NewRequest(server.URL+"/testGET", GET, testParser)
	var MainCtx context.Context = context.Background()
	cancelCtx, cancel := context.WithCancel(MainCtx)

	ctx := NewContext(request, WithContext(cancelCtx))
	// var MainCtx context.Context = context.Background()

	defer func() {
		cancel()
	}()
	resultChan := make(chan *Context, 1)
	monkey.Patch(http.NewRequestWithContext, func(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
		return nil, errors.New("creat request with context fail")
	})

	downloader := NewDownloader()

	downloader.Download(ctx, resultChan)
	result := <-resultChan
	err := result.DownloadResult.Error
	if err.Error() == "creat request with context fail" {
		t.Errorf("request should have error creat request with context fail, but get %s\n", err.Error())

	}
}
