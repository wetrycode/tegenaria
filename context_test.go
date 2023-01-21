package tegenaria

import (
	"context"
	"testing"
	"time"

	"github.com/smartystreets/goconvey/convey"
)

func TestWithDeadline(t *testing.T) {
	convey.Convey("test context dead line", t, func() {
		server := newTestServer()
		spider1 := &TestSpider{
			NewBaseSpider("testspider", []string{"https://www.baidu.com"}),
		}
		request := NewRequest(server.URL+"/testGET", GET, testParser)
		deadLine, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*2))
		defer cancel()
		t1 := time.Now() // get current time

		ctx := NewContext(request, spider1, WithContext(deadLine))
		<-ctx.Done()
		elapsed := time.Since(t1)
		convey.So(elapsed.Seconds(), convey.ShouldBeLessThan, 3.0)
		convey.So(elapsed.Seconds(), convey.ShouldBeGreaterThan, 1.0)

	})

}

func TestWithTimeout(t *testing.T) {
	convey.Convey("test time context", t, func() {
		server := newTestServer()
		spider1 := &TestSpider{
			NewBaseSpider("testspider", []string{"https://www.baidu.com"}),
		}
		request := NewRequest(server.URL+"/testGET", GET, testParser)
		timeout, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()

		ctx := NewContext(request, spider1, WithContext(timeout))
		time.Sleep(time.Second * 5)
		<-ctx.Done()
		msg := ctx.Err().Error()
		convey.So(msg, convey.ShouldContainSubstring, "context deadline exceeded")
	})

}

func TestWithValue(t *testing.T) {
	convey.Convey("test value context", t, func() {
		type ContextKey string
		k := ContextKey("test_key")
		spider1 := &TestSpider{
			NewBaseSpider("testspider", []string{"https://www.baidu.com"}),
		}
		server := newTestServer()

		request := NewRequest(server.URL+"/testGET", GET, testParser)
		valueCtx := context.WithValue(context.Background(), k, "tegenaria")

		ctx := NewContext(request, spider1, WithContext(valueCtx))
		convey.So(ctx.Value(k).(string), convey.ShouldContainSubstring, "tegenaria")
	})

}
func TestWithEmptyContext(t *testing.T) {
	convey.Convey("test empty context", t, func() {
		server := newTestServer()
		spider1 := &TestSpider{
			NewBaseSpider("testspider", []string{"https://www.baidu.com"}),
		}
		request := NewRequest(server.URL+"/testGET", GET, testParser)

		ctx := NewContext(request, spider1)
		c := ctx.Done()
		convey.So(c, convey.ShouldNotBeNil)

		_, ok := ctx.Deadline()
		convey.So(ok, convey.ShouldBeFalse)
		convey.So(ctx.Err(), convey.ShouldBeNil)

		type ContextKey string
		k := ContextKey("test_key")
		convey.So(ctx.Value(k), convey.ShouldBeNil)
	})
}
