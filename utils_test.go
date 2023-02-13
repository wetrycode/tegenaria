package tegenaria

import (
	"reflect"
	"testing"
)

func TestGetParserByName(t *testing.T) {
	server := newTestServer()
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
