package tegenaria

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	bloom "github.com/bits-and-blooms/bloom/v3"
	"github.com/gin-gonic/gin"
)

func parser(resp *Response, item chan<- ItemInterface, req chan<- *Request) {}
func newTestProxyServer() *httptest.Server {
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

		body, err := ioutil.ReadAll(resp.Body)
		c.Data(200, "text/plain", body)        // write response body to response.
		c.String(200, "This is proxy Server.") // add proxy info.
	})
	ts := httptest.NewServer(router)
	return ts
}
func newTestServer() *httptest.Server {
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
		http.SetCookie(c.Writer, &http.Cookie{
			Name:    "key",
			Value:   "value",
			Path:    "/",
			Expires: time.Now().Add(30 * time.Second),
		})
		for _, cookie := range cookies {
			c.String(200, cookie.Name+"="+cookie.Value)
		}
	})
	router.GET("/testHeader", func(c *gin.Context) {
		header := c.GetHeader("key")
		c.String(200, header)
	})
	router.GET("/testTimeout", func(c *gin.Context) {
		time.Sleep(5 * time.Second)
		c.String(200, "timeout")
	})
	router.GET("/testParams", func(c *gin.Context) {
		value := c.Query("key")
		c.String(200, value)
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

	ts := httptest.NewServer(router)
	return ts
}
func TestRequestGet(t *testing.T) {
	server := newTestServer()
	defer server.Close()
	request := NewRequest(server.URL+"/testGET", GET, parser)
	var MainCtx context.Context = context.Background()

	ctx, cancel := context.WithCancel(MainCtx)
	defer func() {
		cancel()
	}()
	resultChan := make(chan *RequestResult, 1)
	downloader := NewDownloader(DownloadWithTlsConfig(tls.Config{InsecureSkipVerify: true}))

	go downloader.Download(ctx, request, resultChan)
	result := <-resultChan
	err := result.Error
	resp := result.Response
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
	defer server.Close()
	request := NewRequest(server.URL+"/testPOST", POST, parser, RequestWithRequestBody(body))
	var MainCtx context.Context = context.Background()

	ctx, cancel := context.WithCancel(MainCtx)
	defer func() {
		cancel()
	}()
	resultChan := make(chan *RequestResult, 1)
	downloader := NewDownloader()

	go downloader.Download(ctx, request, resultChan)
	result := <-resultChan
	err := result.Error
	resp := result.Response
	if err != nil {
		t.Errorf("request error")

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
	downloader := NewDownloader()

	defer server.Close()
	request := NewRequest(server.URL+"/testGetCookie", GET, parser, RequestWithRequestCookies(cookies))
	var MainCtx context.Context = context.Background()

	ctx, cancel := context.WithCancel(MainCtx)
	defer func() {
		cancel()
	}()
	resultChan := make(chan *RequestResult, 1)
	go downloader.Download(ctx, request, resultChan)
	result := <-resultChan
	err := result.Error
	resp := result.Response

	if err != nil {
		t.Errorf("request error with cookies")

	}
	if resp.Status != 200 {
		t.Errorf("response with cookies status = %d; expected %d", resp.Status, 200)

	}
	if resp.String() != "test1=test1test2=test2" {
		t.Errorf("request with cookies get = %s; expected %s", resp.String(), "test1=test1test2=test2")

	}
}

func TestRequestQueryParams(t *testing.T) {
	params := map[string]string{
		"key": "value",
	}
	server := newTestServer()
	downloader := NewDownloader()

	defer server.Close()
	request := NewRequest(server.URL+"/testParams", GET, parser, RequestWithRequestParams(params))
	var MainCtx context.Context = context.Background()

	ctx, cancel := context.WithCancel(MainCtx)
	defer func() {
		cancel()
	}()
	resultChan := make(chan *RequestResult, 1)
	go downloader.Download(ctx, request, resultChan)
	result := <-resultChan
	err := result.Error
	resp := result.Response
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
		HTTP:  proxyServer.URL,
		HTTPS: proxyServer.URL,
		SOCKS: "",
	}
	defer server.Close()
	defer proxyServer.Close()
	request := NewRequest(server.URL+"/proxy", GET, parser, RequestWithRequestProxy(proxy))
	var MainCtx context.Context = context.Background()

	ctx, cancel := context.WithCancel(MainCtx)
	defer func() {
		cancel()
	}()
	resultChan := make(chan *RequestResult, 1)
	go downloader.Download(ctx, request, resultChan)
	result := <-resultChan
	err := result.Error
	resp := result.Response
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
	request := NewRequest(server.URL+"/testHeader", GET, parser, RequestWithRequestHeader(headers))
	var MainCtx context.Context = context.Background()

	ctx, cancel := context.WithCancel(MainCtx)
	defer func() {
		cancel()
	}()
	resultChan := make(chan *RequestResult, 1)
	go downloader.Download(ctx, request, resultChan)
	result := <-resultChan
	err := result.Error
	resp := result.Response
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
func TestFingerprint(t *testing.T) {
	server := newTestServer()
	// downloader := NewDownloader()
	headers := map[string]string{
		"Params1":    "params1",
		"Intparams":  "1",
		"Boolparams": "false",
	}
	request1 := NewRequest(server.URL+"/testHeader", GET, parser, RequestWithRequestHeader(headers))
	request2 := NewRequest(server.URL+"/testHeader", GET, parser, RequestWithRequestHeader(headers))
	request3 := NewRequest(server.URL+"/testHeader2", GET, parser, RequestWithRequestHeader(headers))

	bloomFilter := bloom.New(1024*1024, 5)
	if request1.doUnique(bloomFilter) {
		t.Errorf("Request1 igerprint sum error expected=%v, get=%v", false, true)
	}
	if !request2.doUnique(bloomFilter) {
		t.Errorf("Request2 igerprint sum error expected=%v, get=%v", true, false)

	}
	if request3.doUnique(bloomFilter) {
		t.Errorf("Request3 igerprint sum error expected=%v, get=%v", false, true)

	}
}
func TestTimeout(t *testing.T) {
	server := newTestServer()
	downloader := NewDownloader(DownloadWithTimeout(1 * time.Second))

	request := NewRequest(server.URL+"/testTimeout", GET, parser)
	var MainCtx context.Context = context.Background()

	ctx, cancel := context.WithCancel(MainCtx)
	defer func() {
		cancel()
	}()
	resultChan := make(chan *RequestResult, 1)
	go downloader.Download(ctx, request, resultChan)
	result := <-resultChan
	err := result.Error
	resp := result.Response
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
	req := NewRequest(server.URL+"/testFile", GET, parser, RequestWithResponseWriter(file))
	var MainCtx context.Context = context.Background()

	ctx, cancel := context.WithCancel(MainCtx)
	defer func() {
		cancel()
	}()
	resultChan := make(chan *RequestResult, 1)
	go downloader.Download(ctx, req, resultChan)
	result := <-resultChan
	err := result.Error
	resp := result.Response
	if err != nil || resp == nil {
		t.Errorf("request error %s", err.Error())

	}
	if resp.Status != 200 {
		t.Errorf("response status = %d; expected %d", resp.Status, 200)

	}
	fi,err:=os.Stat("test.file")
	if err !=nil{
		t.Errorf("get test.file info fail %s", err)

	}
	if fi.Size()==0{
		t.Errorf("get test.file size 0")

	}
}
