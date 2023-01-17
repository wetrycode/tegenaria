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
		request := NewRequest("http://www.baidu.com", GET, testParser)
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
