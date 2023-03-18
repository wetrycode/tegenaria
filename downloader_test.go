package tegenaria

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/smartystreets/goconvey/convey"
)

func newRequestDownloadCase(uri string, method RequestMethod, opts ...RequestOption) (*Response, error) {
	server := NewTestServer()
	newTestSpider()
	request := NewRequest(server.URL+uri, method, testParser, opts...)
	ctx := NewContext(request, testSpider)
	downloader := NewDownloader(DownloadWithTLSConfig(&tls.Config{InsecureSkipVerify: true}), DownloadWithH2((true)))

	return downloader.Download(ctx)
}
func TestRequestGet(t *testing.T) {
	convey.Convey("test request get", t, func() {
		resp, errHandle := newRequestDownloadCase("/testGET", GET, RequestWithMaxConnsPerHost(3))
		convey.So(errHandle, convey.ShouldBeNil)
		convey.So(resp.Status, convey.ShouldAlmostEqual, 200)
		str, _ := resp.String()
		convey.So(str, convey.ShouldContainSubstring, "GET")
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

	convey.Convey("test request bytes body", t, func() {
		body := `{
			"key1": "value1"
		}`
		resp, err := newRequestDownloadCase("/testPOST", POST, RequestWithRequestBytesBody([]byte(body)))
		convey.So(err, convey.ShouldBeNil)
		convey.So(resp.Status, convey.ShouldAlmostEqual, 200)
		data, _ := resp.Json()
		convey.So(data["key1"].(string), convey.ShouldContainSubstring, "value1")
	})

	convey.Convey("test request bytes body", t, func() {
		body := `{
			"key1": "value1"
		}`
		resp, err := newRequestDownloadCase("/testPOST", POST, RequestWithRequestBytesBody([]byte(body)))
		convey.So(err, convey.ShouldBeNil)
		convey.So(resp.Status, convey.ShouldAlmostEqual, 200)
		data, _ := resp.Json()
		convey.So(data["key1"].(string), convey.ShouldContainSubstring, "value1")
	})

	convey.Convey("test request meta", t, func() {
		meta := map[string]interface{}{
			"key1": "value1",
		}
		resp, err := newRequestDownloadCase("/testPOST", POST, RequestWithRequestMeta(meta))
		convey.So(err, convey.ShouldBeNil)
		convey.So(resp.Status, convey.ShouldAlmostEqual, 200)
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
		str, _ := resp.String()
		convey.So(str, convey.ShouldContainSubstring, "value")
	})

}

func TestRequestProxyWithTimeOut(t *testing.T) {
	convey.Convey("test request with proxy timeout", t, func() {
		proxyServer := NewTestProxyServer()
		proxy := Proxy{
			ProxyUrl: proxyServer.URL,
		}
		defer proxyServer.Close()
		resp, err := newRequestDownloadCase("/testTimeout", GET, RequestWithRequestProxy(proxy), RequestWithTimeout(10*time.Second))
		convey.So(err, convey.ShouldBeNil)
		convey.So(resp.Status, convey.ShouldAlmostEqual, 200)
		str, _ := resp.String()
		convey.So(str, convey.ShouldContainSubstring, "timeout")
	})

	convey.Convey("test request with timeout", t, func() {
		proxyServer := NewTestProxyServer()
		proxy := Proxy{
			ProxyUrl: proxyServer.URL,
		}
		defer proxyServer.Close()
		resp, err := newRequestDownloadCase("/testTimeout", GET, RequestWithRequestProxy(proxy), RequestWithTimeout(1*time.Second))
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
		content, _ := resp.String()
		convey.So(content, convey.ShouldContainSubstring, "value")

	})
}

func TestTimeout(t *testing.T) {
	convey.Convey("test request timeout", t, func() {
		resp, err := newRequestDownloadCase("/testTimeout", GET, RequestWithTimeout(1*time.Second))
		convey.So(err, convey.ShouldNotBeNil)
		convey.So(resp, convey.ShouldBeNil)

	})
}
func TestLargeFile(t *testing.T) {
	convey.Convey("test large file", t, func() {
		file, _ := os.Create("test.file")
		defer os.Remove("test.file")
		defer file.Close()
		resp, err := newRequestDownloadCase("/testFile", GET)
		convey.So(err, convey.ShouldBeNil)
		_, err = resp.WriteTo(file)
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

	convey.Convey("test redirect limit is not allowed", t, func() {
		resp, err := newRequestDownloadCase("/testRedirect1", GET, RequestWithMaxRedirects(-1))
		convey.So(err.Error(), convey.ShouldContainSubstring, "maximum number of redirects")
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
		resp, _ := newRequestDownloadCase("/testGET", GET)
		_, err := resp.String()
		msg := err.Error()
		convey.So(msg, convey.ShouldContainSubstring, "read response to buffer error")
	})
}

func TestProxyUrlError(t *testing.T) {
	convey.Convey("test proxy url error", t, func() {
		proxyServer := NewTestProxyServer()
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

func TestRequestBodyReader(t *testing.T) {
	convey.Convey("test request body reader", t, func() {
		body := map[string]string{
			"key": "value",
		}
		buff := new(bytes.Buffer)
		err := json.NewEncoder(buff).Encode(body)
		convey.So(err, convey.ShouldBeNil)

		resp, err := newRequestDownloadCase("/testPOST", POST, RequestWithBodyReader(buff))
		convey.So(err, convey.ShouldBeNil)
		str, err := resp.String()
		convey.So(err, convey.ShouldBeNil)
		convey.So(str, convey.ShouldContainSubstring, "value")
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
