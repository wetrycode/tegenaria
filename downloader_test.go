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
	"sync"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/gin-gonic/gin"
	"github.com/smartystreets/goconvey/convey"
)

var onceServer sync.Once
var onceProxyServer sync.Once
var ts *httptest.Server
var proxyServer *httptest.Server
var testSpider *TestSpider
var onceTestSpider sync.Once

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
func newRequestDownloadCase(uri string, method RequestMethod, opts ...RequestOption) (*Response, error) {
	server := newTestServer()
	newTestSpider()
	request := NewRequest(server.URL+uri, method, testParser, opts...)
	ctx := NewContext(request, testSpider)
	downloader := NewDownloader(DownloadWithTlsConfig(&tls.Config{InsecureSkipVerify: true}))

	return downloader.Download(ctx)
}
func TestRequestGet(t *testing.T) {
	convey.Convey("test request get", t, func() {
		resp, errHandle := newRequestDownloadCase("/testGET", GET)
		convey.So(errHandle, convey.ShouldBeNil)
		convey.So(resp.Status, convey.ShouldAlmostEqual, 200)
		convey.So(resp.String(), convey.ShouldContainSubstring, "GET")
	})

}

func TestRequestPost(t *testing.T) {
	convey.Convey("test request post", t, func() {
		body := map[string]interface{}{
			"key1": "value1",
		}
		resp, err := newRequestDownloadCase("/testPOST", POST, RequestWithRequestBody(body))
		convey.So(err, convey.ShouldBeNil)
		convey.So(resp.Status, convey.ShouldAlmostEqual, 200)
		data, _ := resp.Json()
		convey.So(data["key1"].(string), convey.ShouldContainSubstring, "value1")
	})
}

func TestRequestCookie(t *testing.T) {
	convey.Convey("test request cookie", t, func() {
		cookies := map[string]string{
			"test1": "test1",
			"test2": "test2",
		}
		resp, err := newRequestDownloadCase("/testGETCookie", GET, RequestWithRequestCookies(cookies))
		convey.So(err, convey.ShouldBeNil)
		convey.So(resp.Status, convey.ShouldAlmostEqual, 200)
		convey.So(len(resp.Header["Set-Cookie"]), convey.ShouldAlmostEqual, 2)
	})

}

func TestRequestQueryParams(t *testing.T) {
	convey.Convey("test request with ext params", t, func() {
		params := map[string]string{
			"key": "value",
		}
		resp, err := newRequestDownloadCase("/testParams", GET, RequestWithRequestParams(params))
		convey.So(err, convey.ShouldBeNil)
		convey.So(resp.Status, convey.ShouldAlmostEqual, 200)
		convey.So(resp.String(), convey.ShouldContainSubstring, "value")
	})

}

func TestRequestProxyWithTimeOut(t *testing.T) {
	convey.Convey("test request with proxy", t, func() {
		proxyServer := newTestProxyServer()
		proxy := Proxy{
			ProxyUrl: proxyServer.URL,
		}
		defer proxyServer.Close()
		resp, err := newRequestDownloadCase("/testTimeout", GET, RequestWithRequestProxy(proxy),RequestWithTimeout(10 * time.Second))
		convey.So(err, convey.ShouldBeNil)
		convey.So(resp.Status, convey.ShouldAlmostEqual, 200)
		convey.So(resp.String(), convey.ShouldContainSubstring, "This is proxy Server.")
	})

	convey.Convey("test request with timeout", t, func() {
		proxyServer := newTestProxyServer()
		proxy := Proxy{
			ProxyUrl: proxyServer.URL,
		}
		defer proxyServer.Close()
		resp, err := newRequestDownloadCase("/testTimeout", GET, RequestWithRequestProxy(proxy),RequestWithTimeout(1 * time.Second))
		convey.So(err, convey.ShouldNotBeNil)
		convey.So(resp, convey.ShouldBeNil)
	})
}

func TestRequestHeaders(t *testing.T) {
	convey.Convey("test request with add headers", t, func() {
		headers := map[string]string{
			"key":        "value",
			"Intparams":  "1",
			"Boolparams": "false",
		}
		resp, err := newRequestDownloadCase("/testHeader", GET, RequestWithRequestHeader(headers))
		convey.So(err, convey.ShouldBeNil)
		convey.So(resp.Status, convey.ShouldAlmostEqual, 200)
		content:=resp.String()
		convey.So(content, convey.ShouldContainSubstring, "value")

	})
}

func TestTimeout(t *testing.T) {
	convey.Convey("test request timeout", t, func() {
		resp, err := newRequestDownloadCase("/testTimeout", GET, RequestWithTimeout(1 * time.Second))
		convey.So(err, convey.ShouldNotBeNil)
		convey.So(resp, convey.ShouldBeNil)

	})
}
func TestLargeFile(t *testing.T) {
	convey.Convey("test large file", t, func() {
		file, _ := os.Create("test.file")
		defer os.Remove("test.file")
		defer file.Close()
		resp, err := newRequestDownloadCase("/testFile", GET, RequestWithResponseWriter(file))
		fi, statErr := os.Stat("test.file")
		convey.So(err, convey.ShouldBeNil)
		convey.So(resp, convey.ShouldNotBeNil)
		convey.So(resp.Status, convey.ShouldAlmostEqual, 200)
		convey.So(statErr, convey.ShouldBeNil)
		convey.So(fi.Size(), convey.ShouldBeGreaterThan, 0)

	})
}

func TestJsonResponse(t *testing.T) {
	convey.Convey("test handle json response", t, func() {
		resp, err := newRequestDownloadCase("/testJson", GET)
		convey.So(err, convey.ShouldBeNil)
		convey.So(resp, convey.ShouldNotBeNil)
		convey.So(resp.Status, convey.ShouldAlmostEqual, 200)
		r, errJson := resp.Json()
		convey.So(errJson, convey.ShouldBeNil)
		convey.So(r["name"], convey.ShouldContainSubstring, "json")
		patch := gomonkey.ApplyFunc(json.Unmarshal, func(data []byte, v interface{}) error {
			return errors.New("Unmarshal json response")
		})
		defer patch.Reset()
		r, errJson = resp.Json()
		convey.So(errJson, convey.ShouldBeError, errors.New("Unmarshal json response"))
		convey.So(r, convey.ShouldBeNil)
	})
}
func TestRedirectLimit(t *testing.T) {
	convey.Convey("test redirect limit", t, func() {
		resp, err := newRequestDownloadCase("/testRedirect1", GET, RequestWithMaxRedirects(1))
		convey.So(err, convey.ShouldNotBeNil)
		convey.So(resp, convey.ShouldBeNil)
	})
}

func TestNotAllowRedirect(t *testing.T) {
	convey.Convey("test not allowed redirect", t, func() {
		resp, err := newRequestDownloadCase("/testRedirect1", GET, RequestWithAllowRedirects(false))
		convey.So(err.Error(), convey.ShouldContainSubstring, "maximum number of redirects")
		convey.So(resp, convey.ShouldBeNil)
	})
}

func TestInvalidURL(t *testing.T) {
	convey.Convey("test invlid url", t, func() {
		_, err := newRequestDownloadCase("error/testRedirect1", GET)
		convey.So(err, convey.ShouldNotBeNil)
	})
}

func TestResponseReadError(t *testing.T) {
	convey.Convey("test response read error", t, func() {
		patch := gomonkey.ApplyFunc(io.Copy, func(dst io.Writer, src io.Reader) (written int64, err error) {
			return 0, errors.New("read response to buffer error")
		})
		defer patch.Reset()
		_, err := newRequestDownloadCase("/testGET", GET)
		msg := err.Error()
		convey.So(msg, convey.ShouldContainSubstring, "read response to buffer error")
	})
}

func TestProxyUrlError(t *testing.T) {
	convey.Convey("test proxy url error", t, func() {
		proxyServer := newTestProxyServer()
		proxy := Proxy{
			ProxyUrl: "error",
		}
		patch := gomonkey.ApplyFunc(urlParse, func(string) (url *url.URL, err error) {
			return nil, errors.New("proxy invail url")
		})
		defer proxyServer.Close()
		defer patch.Reset()
		_, err := newRequestDownloadCase("/proxy", GET, RequestWithRequestProxy(proxy))
		convey.So(err.Error(), convey.ShouldContainSubstring, "proxy invail url")
	})
}

func TestDownloaderRequestConextError(t *testing.T) {
	convey.Convey("test downloader request conext error", t, func() {
		patch := gomonkey.ApplyFunc(http.NewRequestWithContext, func(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
			return nil, errors.New("create request with context fail")
		})
		defer patch.Reset()
		_, err := newRequestDownloadCase("/testGET", GET)
		convey.So(err.Error(), convey.ShouldContainSubstring, "create request with context fail")
	})

}
