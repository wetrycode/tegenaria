package tegenaria

import (
	"errors"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	jsoniter "github.com/json-iterator/go"
	"github.com/smartystreets/goconvey/convey"
)

func TestRequestWithBodyError(t *testing.T) {
	convey.Convey("request body read error,should be panic", t, func() {
		body := map[string]interface{}{}
		body["key"] = "value"
		request := NewRequest("http://www.example.com", GET, testParser)
		opt := RequestWithRequestBody(body)
		patch := gomonkey.ApplyFunc(jsoniter.Marshal, func(_ interface{}) ([]byte, error) {
			return nil, errors.New("marshal error")
		})
		defer patch.Reset()
		f := func() {
			opt(request)
		}
		convey.So(f, convey.ShouldPanic)
	})
}

func TestRequestToMap(t *testing.T) {
	convey.Convey("request to map", t, func() {

		body := map[string]interface{}{}
		body["key"] = "value"
		bytesBody := `{
			"key":"value"
		}`
		request := NewRequest("http://www.example.com", GET, testParser)

		r, err := request.ToMap()
		convey.So(err, convey.ShouldBeNil)
		convey.So(r["url"], convey.ShouldContainSubstring, "http://www.example.com")

		newReq := RequestFromMap(r, RequestWithRequestBytesBody([]byte(bytesBody)))
		convey.So(newReq.Url, convey.ShouldContainSubstring, "http://www.example.com")

	})

}

func TestRequestWithParser(t *testing.T) {
	convey.Convey("RequestWithParser", t, func() {

		testSpider = &TestSpider{
			NewBaseSpider("tWithParserSpider", []string{"https://www.example.com"}),
		}
		request := NewRequest("http://www.example.com", GET, testSpider.Parser, RequestWithParser(testSpider.Parser))
		convey.So(request.Parser, convey.ShouldContainSubstring, GetFunctionName(testSpider.Parser))

	})

}
