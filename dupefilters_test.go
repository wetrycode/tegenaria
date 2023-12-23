package tegenaria

import (
	"errors"
	"io"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/smartystreets/goconvey/convey"
)

func TestDoDupeFilter(t *testing.T) {

	convey.Convey("test dupefilter", t, func() {
		server := NewTestServer()
		headers := map[string]string{
			"Params1":    "params1",
			"Intparams":  "1",
			"Boolparams": "false",
		}
		spider1 := &TestSpider{
			NewBaseSpider("testspider", []string{"https://m.s.weibo.com/ajax_topic/detail?q=%23%E9%9F%A9%E5%9B%BD%E9%98%B4%E5%8E%86%E6%96%B0%E5%B9%B4%E5%BC%95%E4%BA%89%E8%AE%AE%23"}),
		}
		request1 := NewRequest(server.URL+"/testHeader", GET, testParser, RequestWithRequestHeader(headers))
		ctx1 := NewContext(request1, spider1)

		request2 := NewRequest(server.URL+"/testHeader", GET, testParser, RequestWithRequestHeader(headers))
		ctx2 := NewContext(request2, spider1)

		request3 := NewRequest(server.URL+"/testHeader", GET, testParser, RequestWithRequestHeader(headers), RequestWithDoNotFilter(true))
		ctx3 := NewContext(request3, spider1)

		request4 := NewRequest(server.URL+"/testHeader2", GET, testParser, RequestWithRequestHeader(headers))
		ctx4 := NewContext(request4, spider1)

		duplicates := NewRFPDupeFilter(0.001, 1024*1024)
		r1, _ := duplicates.DoDupeFilter(ctx1)
		convey.So(r1, convey.ShouldBeFalse)
		r2, _ := duplicates.DoDupeFilter(ctx2)
		convey.So(r2, convey.ShouldBeTrue)
		r3, _ := duplicates.DoDupeFilter(ctx3)
		convey.So(r3, convey.ShouldBeFalse)

		r4, _ := duplicates.DoDupeFilter(ctx4)
		convey.So(r4, convey.ShouldBeFalse)
	})
}
func TestDoDupeFilterErr(t *testing.T) {
	convey.Convey("test dupefilter error", t, func() {
		patch := gomonkey.ApplyFunc(io.WriteString, func(_ io.Writer, _ string) (int, error) {
			return 0, errors.New("write string error")
		})
		defer patch.Reset()
		server := NewTestServer()
		// downloader := NewDownloader()
		headers := map[string]string{
			"Params1":    "params1",
			"Intparams":  "1",
			"Boolparams": "false",
		}
		body := make(map[string]interface{})
		body["test"] = "tegenaria"
		request1 := NewRequest(server.URL+"/testHeader", GET, testParser, RequestWithRequestHeader(headers), RequestWithRequestBody(body))
		spider1 := &TestSpider{
			NewBaseSpider("DoDupeFilterErrTestSpider", []string{"https://www.example.com"}),
		}
		ctx1 := NewContext(request1, spider1)
		duplicates := NewRFPDupeFilter(0.001, 1024*1024)
		ret, err := duplicates.DoDupeFilter(ctx1)
		convey.So(ret, convey.ShouldBeFalse)
		convey.So(err, convey.ShouldBeError, errors.New("write string error"))

	})
}
func TestDoBodyDupeFilter(t *testing.T) {
	convey.Convey("test body dupefilter", t, func() {
		server := NewTestServer()
		// downloader := NewDownloader()
		headers := map[string]string{
			"Params1":    "params1",
			"Intparams":  "1",
			"Boolparams": "false",
		}
		body := make(map[string]interface{})
		body["test"] = "tegenaria"
		body1 := make(map[string]interface{})
		body1["test"] = "tegenaria2"
		spider1 := &TestSpider{
			NewBaseSpider("testspider", []string{"https://www.example.com"}),
		}
		request1 := NewRequest(server.URL+"/testHeader", GET, testParser, RequestWithRequestHeader(headers), RequestWithRequestBody(body))
		ctx1 := NewContext(request1, spider1)

		request2 := NewRequest(server.URL+"/testHeader", GET, testParser, RequestWithRequestHeader(headers), RequestWithRequestBody(body))
		ctx2 := NewContext(request2, spider1)

		request3 := NewRequest(server.URL+"/testHeader2", GET, testParser, RequestWithRequestHeader(headers))
		ctx3 := NewContext(request3, spider1)

		request4 := NewRequest(server.URL+"/testHeader", GET, testParser, RequestWithRequestHeader(headers), RequestWithRequestBody(body))
		ctx4 := NewContext(request4, spider1)

		duplicates := NewRFPDupeFilter(0.001, 1024*1024)

		r1, err := duplicates.DoDupeFilter(ctx1)
		convey.So(err, convey.ShouldBeNil)
		convey.So(r1, convey.ShouldBeFalse)

		r2, err := duplicates.DoDupeFilter(ctx2)
		convey.So(err, convey.ShouldBeNil)
		convey.So(r2, convey.ShouldBeTrue)

		r3, err := duplicates.DoDupeFilter(ctx3)
		convey.So(err, convey.ShouldBeNil)
		convey.So(r3, convey.ShouldBeFalse)

		r4, err := duplicates.DoDupeFilter(ctx4)
		convey.So(err, convey.ShouldBeNil)
		convey.So(r4, convey.ShouldBeTrue)
	})

}
