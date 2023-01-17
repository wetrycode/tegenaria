package tegenaria

import (
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/alicebob/miniredis/v2"
	"github.com/go-kiss/monkey"
	"github.com/smartystreets/goconvey/convey"
	queue "github.com/yireyun/go-queue"
)

type TestDownloadMiddler struct {
	Priority int
	Name     string
}

func (m TestDownloadMiddler) GetPriority() int {
	return m.Priority
}
func (m TestDownloadMiddler) ProcessRequest(ctx *Context) error {
	header := fmt.Sprintf("priority-%d", m.Priority)
	ctx.Request.Header[header] = strconv.Itoa(m.Priority)
	return nil
}

func (m TestDownloadMiddler) ProcessResponse(ctx *Context, req chan<- *Context) error {
	return nil

}
func (m TestDownloadMiddler) GetName() string {
	return m.Name
}

type TestDownloadMiddler2 struct {
	Priority int
	Name     string
}

func (m TestDownloadMiddler2) GetPriority() int {
	return m.Priority
}
func (m TestDownloadMiddler2) ProcessRequest(ctx *Context) error {
	return errors.New("process request fail")
}

func (m TestDownloadMiddler2) ProcessResponse(ctx *Context, req chan<- *Context) error {
	return errors.New("process response fail")

}
func (m TestDownloadMiddler2) GetName() string {
	return m.Name
}
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
	convey.Convey("EngineRegister", t, func() {
		engine := newTestEngine("testSpider1")
		for index, pipeline := range engine.pipelines {
			convey.So(pipeline.GetPriority(), convey.ShouldAlmostEqual, index)
		}
		convey.So(engine.spiders.SpidersModules, convey.ShouldContainKey, "testSpider1")

	})

}

func TestEngineOptions(t *testing.T) {
	convey.Convey("Add EngineOptions to engine when new an engine", t, func() {
		engine := NewEngine(
			EngineWithCache(NewRequestCache()),
			EngineWithDownloader(NewDownloader()),
			EngineWithFilter(NewRFPDupeFilter(0.001, 1024*1024)),
			EngineWithUniqueReq(true),
			EngineWithLimiter(NewDefaultLimiter(32)),
		)
		convey.So(engine.cache, convey.ShouldHaveSameTypeAs, NewRequestCache())
		convey.So(engine.downloader, convey.ShouldHaveSameTypeAs, NewDownloader())
		convey.So(engine.RFPDupeFilter, convey.ShouldHaveSameTypeAs, NewRFPDupeFilter(0.001, 1024*1024))
		convey.So(engine.filterDuplicateReq, convey.ShouldBeTrue)
		convey.So(engine.limiter, convey.ShouldHaveSameTypeAs, NewDefaultLimiter(32))
	})

}

func TestCache(t *testing.T) {
	convey.Convey("request write to memory cache", t, func() {
		engine := NewEngine()
		ctx := newTestRequest()
		engine.writeCache(ctx)
		convey.So(engine.cache.getSize(), convey.ShouldAlmostEqual, 1)
	})

	convey.Convey("request write to memory cache dupefilters", t, func() {
		config := NewDistributedWorkerConfig("", "", 0)
		mockRedis := miniredis.RunT(t)
		defer mockRedis.Close()
		worker := NewDistributedWorker(mockRedis.Addr(), config)
		worker.setCurrentSpider("testCacheSpider")
		opts := []EngineOption{}
		opts = append(opts, EngineWithUniqueReq(true))
		opts = append(opts, EngineWithDistributedWorker(worker))
		engine := newTestEngine("testCacheSpider", opts...)
		ctx := newTestRequest()
		body := map[string]interface{}{}
		body["test"] = "test"
		ctx1 := newTestRequest(RequestWithRequestBody(body))
		engine.writeCache(ctx1)
		engine.writeCache(ctx)
		err := engine.writeCache(ctx)
		convey.So(err.Error(), convey.ShouldContainSubstring, "request is nil,maybe it had been free")
		convey.So(engine.cache.getSize(), convey.ShouldAlmostEqual, 2)
	})
}
func TestCacheError(t *testing.T) {
	convey.Convey("request write to cache error", t, func() {
		engine := NewEngine(EngineWithUniqueReq(false))
		q := NewRequestCache()
		q.queue = queue.NewQueue(3)
		engine.cache = q

		ctx := newTestRequest()
		engine.writeCache(ctx)
		patch := gomonkey.ApplyFunc((*queue.EsQueue).Put, func(_ *queue.EsQueue, _ interface{}) (bool, uint32) { return false, 0 })
		engine.writeCache(ctx)
		patch.Reset()
		newCtx := <-engine.cacheChan
		convey.So(newCtx, convey.ShouldNotBeNil)
		size := engine.cache.getSize()
		convey.So(size, convey.ShouldAlmostEqual, 1)
		engine.cache.close()

	})
}

func TestDoDownload(t *testing.T) {
	convey.Convey("engine download request and parse response and exec pipenline ", t, func() {
		monkey.UnpatchAll()
		engine := newTestEngine("testSpider2")
		ctx := newTestRequest()
		err := engine.doDownload(ctx)
		convey.So(err, convey.ShouldBeNil)
		convey.So(ctx.Response.Status, convey.ShouldAlmostEqual, 200)
		convey.So(ctx.Request.Header, convey.ShouldContainKey, "priority-0")
		engine.doParse(ctx)
		for item := range ctx.Items {
			i := item.Item.(*testItem)
			convey.So(i.test, convey.ShouldContainSubstring, "test")
		}
		errHanler := engine.doPipelinesHandlers(ctx)
		convey.So(errHanler, convey.ShouldBeNil)
	})

}

func TestErrResponse(t *testing.T) {
	convey.Convey("test error response handle", t, func() {
		downloader := NewDownloader(DownloadWithTimeout(1 * time.Second))
		engine := newTestEngine("testSpider4", EngineWithDownloader(downloader))
		ctx := newTestRequest()
		ctx.Request.Url = newTestServer().URL + "/testTimeout"
		err := engine.doDownload(ctx)
		convey.So(err, convey.ShouldNotBeNil)

	})

}

func TestAllowedStatusCode(t *testing.T) {
	convey.Convey("test allowed status code", t, func() {
		engine := newTestEngine("testSpider5")
		ctx := newTestRequest(RequestWithAllowedStatusCode([]uint64{404, 403}))
		ctx.Request.Url = newTestServer().URL + "/test404"

		err := engine.doDownload(ctx)
		convey.So(err, convey.ShouldBeNil)
		err = engine.doHandleResponse(ctx)
		convey.So(err, convey.ShouldBeNil)
	})

}

func TestNotAllowedStatus(t *testing.T) {
	convey.Convey("test not allowed status", t, func() {
		engine := newTestEngine("testSpider6")
		ctx := newTestRequest(RequestWithAllowedStatusCode([]uint64{404}))
		ctx.Request.Url = newTestServer().URL + "/test403"
		err := engine.doDownload(ctx)
		convey.So(err, convey.ShouldBeNil)
		err = engine.doHandleResponse(ctx)
		convey.So(err.Error(), convey.ShouldContainSubstring, "not allow handle status code")
	})

}

func TestSpiderNotFound(t *testing.T) {
	convey.Convey("test spider not found", t, func() {
		engine := newTestEngine("testSpiderNotFound")
		f := engine.startSpider("spiderNotFound")
		convey.So(func() {
			f()
		}, convey.ShouldPanic)
	})

}
func TestSpiderDuplicate(t *testing.T) {
	convey.Convey("test spider duplicate", t, func() {
		engine := newTestEngine("testSpider7")
		convey.So(func() {
			engine.RegisterSpiders(&TestSpider{NewBaseSpider("testSpider7", []string{})})

		}, convey.ShouldPanic)
	})
}

func TestEngineStart(t *testing.T) {
	convey.Convey("engine start", t, func() {
		if ctxManager != nil {
			ctxManager.Clear()
		}
		engine := newTestEngine("testSpider9")
		stats := engine.Start("testSpider9")
		convey.So(engine.statistic.GetDownloadFail(), convey.ShouldAlmostEqual, 0)
		convey.So(engine.statistic.GetRequestSent(), convey.ShouldAlmostEqual, 1)
		convey.So(engine.statistic.GetItemScraped(), convey.ShouldAlmostEqual, 1)
		convey.So(engine.statistic.GetErrorCount(), convey.ShouldAlmostEqual, 0)
		stats.Reset()
		engine.Close()
	})

}
func TestEngineStartWithDistributed(t *testing.T) {
	convey.Convey("engine start with distributed", t, func() {
		if ctxManager != nil {
			ctxManager.Clear()
		}
		mockRedis, err := miniredis.Run()
		if err != nil {
			panic(err)
		}
		config := NewDistributedWorkerConfig("", "", 0)
		defer mockRedis.Close()
		worker := NewDistributedWorker(mockRedis.Addr(), config)
		worker.setCurrentSpider("testDistributedSpider9")
		engine := newTestEngine("testDistributedSpider9", EngineWithDistributedWorker(worker))
		go func() {
			for range time.Tick(1 * time.Second) {
				mockRedis.FastForward(1 * time.Second)
			}
		}()
		engine.Start("testDistributedSpider9")
		convey.So(engine.statistic.GetDownloadFail(), convey.ShouldAlmostEqual, 0)
		convey.So(engine.statistic.GetRequestSent(), convey.ShouldAlmostEqual, 1)
		convey.So(engine.statistic.GetItemScraped(), convey.ShouldAlmostEqual, 1)
		convey.So(engine.statistic.GetErrorCount(), convey.ShouldAlmostEqual, 0)
		engine.statistic.Reset()
	})

}
func TestEngineErrorHandler(t *testing.T) {
	convey.Convey("test error handler", t, func() {
		engine := newTestEngine("testErrorHandlerSpider")
		engine.spiders.SpidersModules["testErrorHandlerSpider"].(*TestSpider).FeedUrls = []string{"http://127.0.0.1:12345"}
		engine.Start("testErrorHandlerSpider")
		convey.So(engine.statistic.GetRequestSent(), convey.ShouldAlmostEqual, 1)
		convey.So(engine.statistic.GetErrorCount(), convey.ShouldAlmostEqual, 1)

	})

}

func TestProcessRequestError(t *testing.T) {
	convey.Convey("test process request error", t, func() {
		engine := newTestEngine("testProcessRequestErrorSpider")
		m := TestDownloadMiddler2{9, "test"}
		engine.RegisterDownloadMiddlewares(m)
		ctx := newTestRequest()
		err := engine.doDownload(ctx)
		convey.So(err.Error(), convey.ShouldContainSubstring, "process request fail")
	})

}

func TestProcessResponseError(t *testing.T) {
	convey.Convey("test process response error", t, func() {

		engine := newTestEngine("testProcessResposeErrorSpider")
		m := TestDownloadMiddler2{9, "test"}
		patch := gomonkey.ApplyFunc(TestDownloadMiddler2.ProcessRequest, func(_ TestDownloadMiddler2, _ *Context) error {
			return nil
		})
		defer patch.Reset()
		engine.RegisterDownloadMiddlewares(m)
		ctx := newTestRequest()
		engine.doDownload(ctx)
		err := engine.doHandleResponse(ctx)
		convey.So(err.Error(), convey.ShouldContainSubstring, "process response fail")
	})

}

func TestProcessItemError(t *testing.T) {

	convey.Convey("test process item error", t, func() {
		engine := newTestEngine("testProcessItemErrorSpider")
		engine.RegisterPipelines(&TestItemPipeline4{4})
		m := make(map[string]string)
		m["a"] = "b"
		ctx := newTestRequest()
		item := NewItem(ctx, &testItem{"TEST", make([]int, 0)})
		ctx.Items <- item
		err := engine.doPipelinesHandlers(ctx)
		convey.So(err.Error(), convey.ShouldContainSubstring, "process item fail")
	})
}
func TestParseError(t *testing.T) {

	convey.Convey("test parser error", t, func() {
		engine := newTestEngine("testSpiderParseError")
		server := newTestServer()
		spider, err := engine.spiders.GetSpider("testSpiderParseError")
		convey.So(err, convey.ShouldBeNil)

		request := NewRequest(server.URL+"/testGET", GET, func(resp *Context, req chan<- *Context) error {
			return errors.New("parse response error")
		})
		ctx := NewContext(request, spider)
		err = engine.doDownload(ctx)
		convey.So(err, convey.ShouldBeNil)
		err = engine.doParse(ctx)
		convey.So(err, convey.ShouldNotBeNil)
		convey.So(err.Error(), convey.ShouldContainSubstring, "parse response error")

	})
}
func wokerError(ctx *Context, url string, errMsg string, t *testing.T, patch *gomonkey.Patches, engine *CrawlEngine) {
	convey.Convey(fmt.Sprintf("test %s", errMsg), t, func() {
		defer func() {
			patch.Reset()
			restContext(ctx, url)
		}()
		f := engine.worker(ctx)
		f()
		convey.So(ctx.Error.Error(), convey.ShouldContainSubstring, errMsg)
	})
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
	patch := gomonkey.ApplyFunc(
		(*CrawlEngine).doDownload,
		func(_ *CrawlEngine, _ *Context) error {
			return fmt.Errorf("download error")
		})
	wokerError(ctx, url, "download error", t, patch, engine)
	patch = gomonkey.ApplyFunc((*CrawlEngine).doHandleResponse, func(_ *CrawlEngine, _ *Context) error { return fmt.Errorf("call handleResponse error") })
	wokerError(ctx, url, "call handleResponse error", t, patch, engine)

	patch = gomonkey.ApplyFunc((*CrawlEngine).doParse, func(_ *CrawlEngine, _ *Context) error { return fmt.Errorf("call parser error") })
	wokerError(ctx, url, "call parser error", t, patch, engine)

	patch = gomonkey.ApplyFunc((*CrawlEngine).doPipelinesHandlers, func(_ *CrawlEngine, _ *Context) error {
		return fmt.Errorf("call PipelinesHandlers error")
	})
	wokerError(ctx, url, "call PipelinesHandlers error", t, patch, engine)
	patch = gomonkey.ApplyFunc((*CrawlEngine).doDownload, func(_ *CrawlEngine, _ *Context) error {
		panic("call doDownload panic")
	})
	wokerError(ctx, url, "call doDownload panic", t, patch, engine)

}
