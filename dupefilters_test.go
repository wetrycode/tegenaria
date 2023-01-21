package tegenaria

import (
	"testing"

	"github.com/smartystreets/goconvey/convey"
)

func TestDoDupeFilter(t *testing.T) {

	convey.Convey("test dupefilter", t, func() {
		server := newTestServer()
		headers := map[string]string{
			"Params1":    "params1",
			"Intparams":  "1",
			"Boolparams": "false",
		}
		spider1 := &TestSpider{
			NewBaseSpider("testspider", []string{"https://www.baidu.com"}),
		}
		request1 := NewRequest(server.URL+"/testHeader", GET, testParser, RequestWithRequestHeader(headers))
		ctx1 := NewContext(request1, spider1)

		request2 := NewRequest(server.URL+"/testHeader", GET, testParser, RequestWithRequestHeader(headers))
		ctx2 := NewContext(request2, spider1)

		request3 := NewRequest(server.URL+"/testHeader2", GET, testParser, RequestWithRequestHeader(headers))
		ctx3 := NewContext(request3, spider1)

		duplicates := NewRFPDupeFilter(0.001, 1024*1024)
		r1, _ := duplicates.DoDupeFilter(ctx1)
		convey.So(r1, convey.ShouldBeFalse)
		r2, _ := duplicates.DoDupeFilter(ctx2)
		convey.So(r2, convey.ShouldBeTrue)
		r3, _ := duplicates.DoDupeFilter(ctx3)
		convey.So(r3, convey.ShouldBeFalse)
	})
}

func TestDoBodyDupeFilter(t *testing.T) {
	convey.Convey("test body dupefilter", t, func() {
		server := newTestServer()
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
			NewBaseSpider("testspider", []string{"https://www.baidu.com"}),
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
