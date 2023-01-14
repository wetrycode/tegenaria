package tegenaria

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-kiss/monkey"
)

func newTestEngine(spiderName string, opts ...EngineOption) *CrawlEngine {
	engine := NewEngine(opts...)
	server := newTestServer()

	// register test spider
	engine.RegisterSpiders(&TestSpider{NewBaseSpider(spiderName, []string{server.URL + "/testGET"})})

	// register test pipelines
	engine.RegisterPipelines(&TestItemPipeline{0})
	engine.RegisterPipelines(&TestItemPipeline3{2})
	engine.RegisterPipelines(&TestItemPipeline2{1})

	// register download middlerware
	engine.RegisterDownloadMiddlewares(TestDownloadMiddler{0, "1"})
	engine.RegisterDownloadMiddlewares(TestDownloadMiddler{2, "3"})
	engine.RegisterDownloadMiddlewares(TestDownloadMiddler{1, "2"})

	return engine
}
func newTestRequest(opts ...RequestOption) *Context {
	server := newTestServer()
	request := NewRequest(server.URL+"/testGET", GET, testParser, opts...)
	ctx := NewContext(request, &TestSpider{NewBaseSpider("spiderRequest", []string{server.URL + "/testGET"})})
	return ctx
}
func TestEngineRegister(t *testing.T) {
	engine := newTestEngine("testSpider1")

	for index, pipeline := range engine.pipelines {
		if pipeline.GetPriority() != index {
			t.Errorf("pipeline priority %d error", pipeline.GetPriority())
		}
	}
	_, ok := engine.spiders.SpidersModules["testSpider1"]
	if !ok {
		t.Errorf("spider register error not found spider %s", "testSpider1")
	}
}

func TestEngineOptions(t *testing.T) {
	NewEngine(
		EngineWithCache(NewRequestCache()),
		EngineWithDownloader(NewDownloader()),
		EngineWithFilter(NewRFPDupeFilter(0.001, 1024*1024)),
		EngineWithUniqueReq(true),
	)

}

func TestCache(t *testing.T) {
	engine := NewEngine()
	// write new request
	ctx := newTestRequest()
	engine.writeCache(ctx)
	if engine.cache.getSize() != 1 {
		t.Errorf("request write to cache fail")
	}

}

func TestDoFilter(t *testing.T) {
	engine := NewEngine(EngineWithUniqueReq(true))
	ctx := newTestRequest()

	for i := 0; i < 3; i++ {
		engine.writeCache(ctx)
	}

	if engine.cache.getSize() != 1 {
		t.Errorf("Cache filter duplicate request fail except %d requests, but get %d requests\n", 1, engine.cache.getSize())
	}

}

func TestDoDownload(t *testing.T) {
	monkey.UnpatchAll()

	engine := newTestEngine("testSpider2")
	// server := newTestServer()

	ctx := newTestRequest()

	err := engine.doDownload(ctx)
	if err != nil {
		t.Errorf("download err:%s", err.Error())
	}
	if ctx.Response.Status != 200 {
		t.Errorf("downloader except get 200 status code ,but get %d", ctx.Response.Status)
	}
	value, ok := ctx.Request.Header["priority-0"]
	if !ok {
		t.Errorf("Download middlerwares not work except priority-0 header, but get nil\n")

	}
	if value != "0" {
		t.Errorf("Download middlerwares not work except priority-0 header value 0, but get %s\n", value)

	}
	engine.doParse(ctx)
	for item := range ctx.Items {
		i := item.Item.(*testItem)
		if i.test != "test" {
			t.Errorf("Download result parser fail except %s get %s\n", "test", i.test)

		}
	}
	// test pipelines
	errHanler := engine.doPipelinesHandlers(ctx)
	if errHanler != nil {
		t.Errorf("item into pipeline err %s", errHanler.Error())
	}

}

func TestErrResponse(t *testing.T) {
	downloader := NewDownloader(DownloadWithTimeout(1 * time.Second))
	engine := newTestEngine("testSpider4", EngineWithDownloader(downloader))
	ctx := newTestRequest()
	ctx.Request.Url = newTestServer().URL + "/testTimeout"
	err := engine.doDownload(ctx)
	if err == nil {
		t.Errorf("Download should be timeout but not \n")

	}

}

func TestAllowedStatusCode(t *testing.T) {
	engine := newTestEngine("testSpider5")
	ctx := newTestRequest(RequestWithAllowedStatusCode([]uint64{404, 403}))
	ctx.Request.Url = newTestServer().URL + "/test404"

	err := engine.doDownload(ctx)
	if err != nil {
		t.Errorf("download request error %s", err.Error())
	}

	err = engine.doHandleResponse(ctx)
	if err != nil {
		t.Errorf("handle response error %s", err.Error())
	}

}

func TestNotAllowedStatus(t *testing.T) {
	engine := newTestEngine("testSpider6")
	ctx := newTestRequest(RequestWithAllowedStatusCode([]uint64{404}))
	ctx.Request.Url = newTestServer().URL + "/test403"

	err := engine.doDownload(ctx)
	if err != nil {
		t.Errorf("download request error %s", err.Error())
	}

	err = engine.doHandleResponse(ctx)
	if err == nil || !strings.Contains(err.Error(), "not allow handle status code") {
		t.Errorf("handle response error should not be nil or %s", err.Error())
	}

}

func TestSpiderNotFound(t *testing.T) {
	defer func() {
		_ = recover()
	}()
	engine := newTestEngine("testSpider6")
	f := engine.startSpider("spiderNotFound")
	f()
	t.Errorf("spider should be not found!")
}
func TestSpiderDuplicate(t *testing.T) {
	defer func() {
		if p := recover(); p == nil {
			t.Errorf("Spider register should be get duplicate error \n")

		}
	}()
	engine := newTestEngine("testSpider7")
	engine.RegisterSpiders(&TestSpider{NewBaseSpider("testSpider7", []string{})})
}

func TestEngineStart(t *testing.T) {
	if ctxManager != nil {
		ctxManager.Clear()
	}
	engine := newTestEngine("testSpider9")
	engine.Start("testSpider9")

	if engine.statistic.DownloadFail != 0 {
		t.Errorf("Spider crawl get %d errors \n", engine.statistic.DownloadFail)
	}
	if engine.statistic.RequestSent != 1 {
		t.Errorf("Spider crawl except 1 download, but get %d \n", engine.statistic.RequestSent)

	}
	if engine.statistic.ItemScraped != 1 {
		t.Errorf("Spider crawl except 1 item, but get %d \n", engine.statistic.ItemScraped)

	}
}
func TestEngineErrorHandler(t *testing.T) {
	engine := newTestEngine("testSpider10")
	engine.spiders.SpidersModules["testSpider10"].(*TestSpider).FeedUrls = []string{"http://127.0.0.1:12345"}
	engine.Start("testSpider10")
	if engine.statistic.ErrorCount != 1 {
		t.Errorf("Spider crawl except 1 error,but get %d \n", engine.statistic.ErrorCount)
	}
}

func TestProcessRequestError(t *testing.T) {
	engine := newTestEngine("testSpider19")
	m := TestDownloadMiddler2{9, "test"}
	engine.RegisterDownloadMiddlewares(m)

	ctx := newTestRequest()

	err := engine.doDownload(ctx)
	if !strings.Contains(err.Error(), "process request fail") {
		t.Errorf("request should have error process request fail, but get %s\n", err.Error())
	}

}

func TestProcessItemError(t *testing.T) {
	engine := newTestEngine("testSpider99")
	engine.RegisterPipelines(&TestItemPipeline4{4})
	m := make(map[string]string)
	m["a"] = "b"
	ctx := newTestRequest()

	item := NewItem(ctx, &testItem{"TEST", make([]int, 0)})
	ctx.Items <- item
	err := engine.doPipelinesHandlers(ctx)
	if !strings.Contains(err.Error(), "process item fail") {
		t.Errorf("request should have error process item fail, but get %s\n", err.Error())
	}
}
func TestParseError(t *testing.T) {
	engine := newTestEngine("testSpiderParseError")
	server := newTestServer()
	spider, err := engine.spiders.GetSpider("testSpiderParseError")
	if err != nil {
		t.Errorf("Get spider testSpiderParseError")
	}
	request := NewRequest(server.URL+"/testGET", GET, func(resp *Context, req chan<- *Context) error {
		return errors.New("parse response error")
	})
	ctx := NewContext(request, spider)
	err = engine.doDownload(ctx)
	if err != nil {
		t.Errorf("download error %s", err.Error())
	}
	err = engine.doParse(ctx)
	if err == nil {
		t.Errorf("Except parse response error but get nil")

	}
	if err != nil && !strings.Contains(err.Error(), "parse response error") {
		t.Errorf("Except parse response error but get %s\n", err.Error())

	}
}
func wokerError(ctx *Context, errMsg string, t *testing.T) {
	if ctx.Error == nil {
		t.Errorf("error should not be nil %s", errMsg)
	} else if !strings.Contains(ctx.Error.Error(), errMsg) {
		t.Errorf("download should get '%s',but get %s", errMsg, ctx.Error.Error())
	}
}
func restContext(ctx *Context, url string) {
	ctx.Error = nil
	ctx.Request = NewRequest(url, GET, testParser)
	ctx.Response = nil

}
func TestWorkerErr(t *testing.T) {
	ctx := newTestRequest()
	url := ctx.Request.Url
	engine := newTestEngine("wokerSpider")
	monkey.Patch((*CrawlEngine).doDownload, func(_ *CrawlEngine, _ *Context) (error) { return fmt.Errorf("call download error") })

	f := engine.worker(ctx)
	f()
	wokerError(ctx, "call download error", t)
	monkey.Unpatch((*CrawlEngine).doDownload)

	restContext(ctx, url)
	monkey.Patch((*CrawlEngine).doHandleResponse, func(_ *CrawlEngine, _ *Context) error { return fmt.Errorf("call handleResponse error") })
	f = engine.worker(ctx)
	f()
	wokerError(ctx, "call handleResponse error", t)
	monkey.Unpatch((*CrawlEngine).doHandleResponse)

	restContext(ctx, url)
	monkey.Patch((*CrawlEngine).doParse, func(_ *CrawlEngine, _ *Context) error { return fmt.Errorf("call parser error") })
	f = engine.worker(ctx)
	f()
	wokerError(ctx, "call parser error", t)
	monkey.Unpatch((*CrawlEngine).doParse)

	restContext(ctx, url)
	monkey.Patch((*CrawlEngine).doPipelinesHandlers, func(_ *CrawlEngine, _ *Context) error {
		return fmt.Errorf("call PipelinesHandlers error")
	})
	f = engine.worker(ctx)
	f()
	wokerError(ctx, "call PipelinesHandlers error", t)
	monkey.Unpatch((*CrawlEngine).doPipelinesHandlers)

	restContext(ctx, url)
	monkey.Patch((*CrawlEngine).doDownload, func(_ *CrawlEngine, _ *Context) (error) {
		panic("call doDownload panic")
	})
	f = engine.worker(ctx)
	f()
	wokerError(ctx, "call doDownload panic", t)
	monkey.Unpatch((*CrawlEngine).doDownload)

}
