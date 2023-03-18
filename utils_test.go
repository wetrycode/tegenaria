package tegenaria

import (
	"reflect"
	"testing"

	"github.com/smartystreets/goconvey/convey"
)

func TestGetParserByName(t *testing.T) {
	server := NewTestServer()
	spider := &TestSpider{NewBaseSpider("spiderParser", []string{"http://127.0.0.1" + "/testGET"})}
	request := NewRequest(server.URL+"/testGET", GET, testParser)
	ctx := NewContext(request, spider)
	ch := make(chan *Context, 1)
	f := GetParserByName(spider, "Parser")
	// f(ctx, ch)
	args := make([]reflect.Value, 2)
	args[0] = reflect.ValueOf(ctx)
	args[1] = reflect.ValueOf(ch)
	f.Call(args)
	item := <-ctx.Items
	it := item.Item.(*testItem)
	if it.test != "test" {
		t.Error("call parser func fail")
	}

	close(ch)

}

func TestGetMachineIP(t *testing.T) {
	convey.Convey("test get machine IP", t, func() {
		ip, err := GetMachineIP()
		convey.So(err, convey.ShouldBeNil)
		convey.So(ip, convey.ShouldNotContainSubstring, "127.0.0.1")

	})
}
func TestInterface2Uint(t *testing.T) {
	convey.Convey("test Interface2Uint", t, func() {
		v1 := Interface2Uint("2")
		v2 := Interface2Uint(2)
		v3 := Interface2Uint(2.0)
		v4 := Interface2Uint(uint64(2))
		v5 := Interface2Uint(int(2))
		v6 := Interface2Uint(nil)
		v7 := func() {
			Interface2Uint(false)
		}
		v8 := Interface2Uint(uint64(2))
		convey.So(v1, convey.ShouldEqual, 2)
		convey.So(v2, convey.ShouldEqual, 2)
		convey.So(v3, convey.ShouldEqual, 2)
		convey.So(v4, convey.ShouldEqual, 2)
		convey.So(v5, convey.ShouldEqual, 2)
		convey.So(v6, convey.ShouldEqual, 0)
		convey.So(v8, convey.ShouldEqual, 2)
		convey.So(v7, convey.ShouldPanic)
	})

}
