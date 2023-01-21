package tegenaria

import "testing"

func TestGetParserByName(t *testing.T) {
	server := newTestServer()
	spider := &TestSpider{NewBaseSpider("spiderParser", []string{"http://127.0.0.1" + "/testGET"})}
	request := NewRequest(server.URL+"/testGET", GET, testParser)
	ctx := NewContext(request, spider)
	ch := make(chan *Context, 1)
	f := GetParserByName(spider, "Parser")
	f(ctx, ch)
	item := <-ctx.Items
	it := item.Item.(*testItem)
	if it.test != "test" {
		t.Error("call parser func fail")
	}

	close(ch)

}
