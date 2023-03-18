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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type TestSpider struct {
	*BaseSpider
}

func (s *TestSpider) StartRequest(req chan<- *Context) {

	for _, url := range s.FeedUrls {
		request := NewRequest(url, GET, s.Parser)
		ctx := NewContext(request, s)
		req <- ctx
	}
}
func (s *TestSpider) Parser(resp *Context, req chan<- *Context) error {
	return testParser(resp, req)
}

func (s *TestSpider) ErrorHandler(err *Context, req chan<- *Context) {

}
func (s *TestSpider) GetName() string {
	return s.Name
}
func (s *TestSpider) GetFeedUrls() []string {
	return s.FeedUrls
}

var onceServer sync.Once
var onceProxyServer sync.Once
var testServer *httptest.Server
var proxyServer *httptest.Server
var testSpider *TestSpider
var onceTestSpider sync.Once

type testItem struct {
	test      string
	pipelines []int
}
type TestItemPipeline struct {
	Priority int
}
type TestItemPipeline2 struct {
	Priority int
}
type TestItemPipeline3 struct {
	Priority int
}

type TestItemPipeline4 struct {
	Priority int
}

func (p *TestItemPipeline) ProcessItem(spider SpiderInterface, item *ItemMeta) error {
	i := item.Item.(*testItem)
	i.pipelines = append(i.pipelines, p.Priority)
	return nil

}
func (p *TestItemPipeline) GetPriority() int {
	return p.Priority
}
func (p *TestItemPipeline2) ProcessItem(spider SpiderInterface, item *ItemMeta) error {
	i := item.Item.(*testItem)
	i.pipelines = append(i.pipelines, p.Priority)
	return nil
}
func (p *TestItemPipeline2) GetPriority() int {
	return p.Priority
}

func (p *TestItemPipeline3) ProcessItem(spider SpiderInterface, item *ItemMeta) error {
	i := item.Item.(*testItem)
	i.pipelines = append(i.pipelines, p.Priority)
	return nil

}
func (p *TestItemPipeline3) GetPriority() int {
	return p.Priority
}
func (p *TestItemPipeline4) ProcessItem(spider SpiderInterface, item *ItemMeta) error {
	return errors.New("process item fail")

}
func (p *TestItemPipeline4) GetPriority() int {
	return p.Priority
}
func testParser(resp *Context, req chan<- *Context) error {
	newItem := &testItem{
		test:      "test",
		pipelines: make([]int, 0),
	}
	resp.Items <- NewItem(resp, newItem)
	return nil
}

func newTestSpider() {
	onceTestSpider.Do(func() {
		testSpider = &TestSpider{
			NewBaseSpider("testspider", []string{"https://www.baidu.com"}),
		}
	})
}

func NewTestServer() *httptest.Server {
	onceServer.Do(func() {
		gin.SetMode(gin.DebugMode)
		router := gin.Default()
		// router := gin.New()
		router.GET("/testGET", func(c *gin.Context) {
			c.String(200, "GET")
		})
		router.POST("/testPOST", func(c *gin.Context) {
			dataType, _ := json.Marshal(map[string]string{"key": "value"})
			data, _ := c.GetRawData()
			if string(data) == string(dataType) {

				c.String(200, "POST")
			} else {
				s := string(data)
				c.String(200, s)
			}
		})
		router.GET("/testGETCookie", func(c *gin.Context) {
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
		router.POST("/testForm", func(c *gin.Context) {
			value := c.PostForm("key")
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
		router.GET("/testFile", func(c *gin.Context) {
			testString := make([]string, 200000)
			for i := 0; i < 200000; i++ {
				testString = append(testString, "This test files")
			}
			c.String(200, "", testString)
		})
		testServer = httptest.NewServer(router)
	})
	return testServer

}

func NewTestProxyServer() *httptest.Server {
	onceProxyServer.Do(func() {
		gin.SetMode(gin.ReleaseMode)
		router := gin.New()
		f := func(c *gin.Context) {
			host := c.Request.URL.Host // Get request url
			director := func(req *http.Request) {
				req.URL.Scheme = "http"
				req.URL.Host = host
				req.Host = host
			}
			proxy := &httputil.ReverseProxy{Director: director}
			proxy.ServeHTTP(c.Writer, c.Request)

		}
		router.POST("/:t", f)
		router.GET("/:t", f)

		proxyServer = httptest.NewServer(router)
	})
	return proxyServer
}

type TestDownloadMiddler struct {
	Priority int
	Name     string
}

func (m TestDownloadMiddler) GetPriority() int {
	return m.Priority
}
func (m TestDownloadMiddler) ProcessRequest(ctx *Context) error {
	header := fmt.Sprintf("priority-%d", m.Priority)
	ctx.Request.Header[header] = strconv.Itoa(m.Priority)
	return nil
}

func (m TestDownloadMiddler) ProcessResponse(ctx *Context, req chan<- *Context) error {
	return nil

}
func (m TestDownloadMiddler) GetName() string {
	return m.Name
}

type TestDownloadMiddler2 struct {
	Priority int
	Name     string
}

func (m TestDownloadMiddler2) GetPriority() int {
	return m.Priority
}
func (m TestDownloadMiddler2) ProcessRequest(ctx *Context) error {
	return errors.New("process request fail")
}

func (m TestDownloadMiddler2) ProcessResponse(ctx *Context, req chan<- *Context) error {
	return errors.New("process response fail")

}
func (m TestDownloadMiddler2) GetName() string {
	return m.Name
}

func NewTestRequest(spider SpiderInterface, opts ...RequestOption) *Context {
	server := NewTestServer()
	// spider := &TestSpider{NewBaseSpider("spiderRequest", []string{server.URL + "/testGET"})}
	request := NewRequest(server.URL+"/testGET", GET, spider.Parser, opts...)
	ctx := NewContext(request, spider, WithItemChannelSize(16))
	return ctx
}
func NewTestEngine(spiderName string, opts ...EngineOption) *CrawlEngine {
	engine := NewEngine(opts...)
	server := NewTestServer()

	// register test spider
	engine.RegisterSpiders(&TestSpider{NewBaseSpider(spiderName, []string{server.URL + "/testGET"})})

	// register test pipelines
	engine.RegisterPipelines(&TestItemPipeline{0})
	engine.RegisterPipelines(&TestItemPipeline3{2})
	engine.RegisterPipelines(&TestItemPipeline2{1})

	// register download middlerware
	engine.RegisterDownloadMiddlewares(TestDownloadMiddler{0, "1"})
	engine.RegisterDownloadMiddlewares(TestDownloadMiddler{2, "3"})
	engine.RegisterDownloadMiddlewares(TestDownloadMiddler{1, "2"})

	return engine
}
