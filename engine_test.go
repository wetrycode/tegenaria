package tegenaria

import (
	"errors"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/alicebob/miniredis/v2"
	"github.com/go-kiss/monkey"
	"github.com/smartystreets/goconvey/convey"
	queue "github.com/yireyun/go-queue"
)

func TestEngineRegister(t *testing.T) {
	convey.Convey("EngineRegister", t, func() {
		engine := NewTestEngine("testSpider1")
		for index, pipeline := range engine.pipelines {
			convey.So(pipeline.GetPriority(), convey.ShouldAlmostEqual, index)
		}
		convey.So(engine.spiders.SpidersModules, convey.ShouldContainKey, "testSpider1")

	})

}

func TestEngineOptions(t *testing.T) {
	convey.Convey("Add EngineOptions to engine when new an engine", t, func() {
		components := NewDefaultComponents(
			DefaultComponentsWithDefaultHooks(NewDefaultHooks()),
			DefaultComponentsWithDefaultLimiter(NewDefaultLimiter(16)),
			DefaultComponentsWithDefaultStatistic(NewDefaultStatistic()),
			DefaultComponentsWithDupefilter(NewRFPDupeFilter(0.001, 1024*1024)),
			DefaultComponentsWithDefaultQueue(NewDefaultQueue(1024*1024)),
		)
		engine := NewEngine(
			EngineWithDownloader(NewDownloader()),
			EngineWithUniqueReq(true),
			EngineWithComponents(components),
		)
		convey.So(engine.downloader, convey.ShouldHaveSameTypeAs, NewDownloader())
		convey.So(engine.components.GetQueue(), convey.ShouldPointTo, components.GetQueue())
		convey.So(engine.components.GetLimiter(), convey.ShouldPointTo, components.GetLimiter())
		convey.So(engine.components.GetDupefilter(), convey.ShouldPointTo, components.GetDupefilter())
		convey.So(engine.components.GetEventHooks(), convey.ShouldPointTo, components.GetEventHooks())
		convey.So(engine.components.GetStats(), convey.ShouldPointTo, components.GetStats())
		convey.So(engine.filterDuplicateReq, convey.ShouldBeTrue)
	})

}

func TestCache(t *testing.T) {
	convey.Convey("request write to memory cache", t, func() {
		engine := NewTestEngine("testWriteMemoryCache")
		ctx := NewTestRequest(engine.GetSpiders().SpidersModules["testWriteMemoryCache"])
		err := engine.writeCache(ctx)
		convey.So(err, convey.ShouldBeNil)

		convey.So(engine.components.GetQueue().GetSize(), convey.ShouldAlmostEqual, 1)
	})
	convey.Convey("test empty request write to memory cache", t, func() {
		engine := NewTestEngine("testEmptyRequest")
		ctx := NewTestRequest(engine.GetSpiders().SpidersModules["testEmptyRequest"])
		ctx.Request = nil
		err := engine.components.GetQueue().Enqueue(ctx)
		convey.So(err, convey.ShouldBeError, errors.New("context or request cannot be nil"))
	})

}
func TestCacheError(t *testing.T) {
	convey.Convey("request write to cache error", t, func() {
		engine := NewTestEngine("testWriteCacheError", EngineWithUniqueReq(false), EngineWithComponents(NewDefaultComponents(DefaultComponentsWithDefaultQueue(NewDefaultQueue(3)))))

		ctx := NewTestRequest(engine.GetSpiders().SpidersModules["testWriteCacheError"])
		err := engine.writeCache(ctx)
		convey.So(err, convey.ShouldBeNil)

		patch := gomonkey.ApplyFunc((*queue.EsQueue).Put, func(_ *queue.EsQueue, _ interface{}) (bool, uint32) { return false, 0 })
		err = engine.writeCache(ctx)
		convey.So(err, convey.ShouldNotBeNil)

		patch.Reset()
		req, err := engine.components.GetQueue().Dequeue()
		convey.So(err, convey.ShouldBeNil)
		convey.So(req, convey.ShouldNotBeNil)

		size := engine.components.GetQueue().GetSize()
		convey.So(size, convey.ShouldAlmostEqual, 0)
		engine.filterDuplicateReq = true
		err = engine.writeCache(ctx)
		convey.So(err, convey.ShouldBeNil)

		convey.So(ctx.CtxID, convey.ShouldNotBeEmpty)

		engine.components.GetQueue().Close()

	})
}
func TestDupeFilterError(t *testing.T) {
	convey.Convey("request write to DupeFilter error", t, func() {
		engine := NewTestEngine("testWriteDupeFilterError")

		ctx := NewTestRequest(engine.GetSpiders().SpidersModules["testWriteDupeFilterError"])
		patch := gomonkey.ApplyFunc((*DefaultRFPDupeFilter).DoDupeFilter, func(_ *DefaultRFPDupeFilter, _ *Context) (bool, error) {
			return false, errors.New("DefaultRFPDupeFilter ERROR")
		})
		defer patch.Reset()
		err := engine.writeCache(ctx)
		convey.So(err, convey.ShouldNotBeNil)
	})
}
func TestStartRequestError(t *testing.T) {
	convey.Convey("request start requests error", t, func() {
		engine := NewTestEngine("testStartRequestError")

		patch := gomonkey.ApplyFunc((*TestSpider).StartRequest, func(_ *TestSpider, _ chan<- *Context) {
			panic("start requests ERROR")
		})
		defer patch.Reset()
		f := func() {
			engine.startSpider(engine.GetSpiders().SpidersModules["testStartRequestError"])
		}
		convey.So(f, convey.ShouldPanic)
	})
}
func TestDoDownload(t *testing.T) {
	convey.Convey("engine download request and parse response and exec pipenline ", t, func() {
		monkey.UnpatchAll()
		engine := NewTestEngine("testSpider2")
		ctx := NewTestRequest(engine.GetSpiders().SpidersModules["testSpider2"])
		err := engine.doDownload(ctx)
		convey.So(err, convey.ShouldBeNil)
		convey.So(ctx.Response.Status, convey.ShouldAlmostEqual, 200)
		convey.So(ctx.Request.Headers, convey.ShouldContainKey, "priority-0")
		err = engine.doParse(ctx)
		convey.So(err, convey.ShouldBeNil)
		close(ctx.Items)
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
		engine := NewTestEngine("testErrResponse", EngineWithDownloader(downloader))
		ctx := NewTestRequest(engine.GetSpiders().SpidersModules["testErrResponse"])
		ctx.Request.Url = NewTestServer().URL + "/testTimeout"
		err := engine.doDownload(ctx)
		convey.So(err, convey.ShouldNotBeNil)

	})

}

func TestAllowedStatusCode(t *testing.T) {
	convey.Convey("test allowed status code", t, func() {
		engine := NewTestEngine("testAllowedStatusCode")
		ctx := NewTestRequest(engine.GetSpiders().SpidersModules["testAllowedStatusCode"], RequestWithAllowedStatusCode([]uint64{404, 403}))
		ctx.Request.Url = NewTestServer().URL + "/test404"

		err := engine.doDownload(ctx)
		convey.So(err, convey.ShouldBeNil)
		err = engine.doHandleResponse(ctx)
		convey.So(err, convey.ShouldBeNil)
	})

}

func TestNotAllowedStatus(t *testing.T) {
	convey.Convey("test not allowed status", t, func() {
		engine := NewTestEngine("testNotAllowedStatus")
		ctx := NewTestRequest(engine.GetSpiders().SpidersModules["testNotAllowedStatus"], RequestWithAllowedStatusCode([]uint64{404}))
		ctx.Request.Url = NewTestServer().URL + "/test403"
		err := engine.doDownload(ctx)
		convey.So(err, convey.ShouldBeNil)
		err = engine.doHandleResponse(ctx)
		convey.So(err.Error(), convey.ShouldContainSubstring, "not allow handle status code")
	})

}
func TestGetSpiders(t *testing.T) {
	convey.Convey("test get spiders", t, func() {
		NewTestEngine("GetSpiders")
		convey.So(SpidersList.GetAllSpidersName(), convey.ShouldContain, "GetSpiders")
	})
}
func TestSpiderNotFound(t *testing.T) {
	convey.Convey("test spider not found", t, func() {
		engine := NewTestEngine("testSpiderNotFound")
		f := func() {
			engine.start("spiderNotFound")
		}
		convey.So(func() {
			f()
		}, convey.ShouldPanic)
	})

}
func TestSpiderDuplicate(t *testing.T) {
	convey.Convey("test spider duplicate", t, func() {
		engine := NewTestEngine("testSpider7")
		convey.So(func() {
			engine.RegisterSpiders(&TestSpider{NewBaseSpider("testSpider7", []string{})})

		}, convey.ShouldPanic)
	})
}

func TestEngineStart(t *testing.T) {
	convey.Convey("engine start", t, func() {

		engine := NewTestEngine("testSpider9", EngineWithReqChannelSize(1024))

		engine.Execute("testSpider9")
		convey.So(GetEngineID(), convey.ShouldContainSubstring, engineID)
		convey.So(engine.GetStatic().Get(DownloadFailStats), convey.ShouldAlmostEqual, 0)
		convey.So(engine.GetStatic().Get(RequestStats), convey.ShouldAlmostEqual, 1)
		convey.So(engine.GetStatic().Get(ItemsStats), convey.ShouldAlmostEqual, 1)
		convey.So(engine.GetStatic().Get("200"), convey.ShouldAlmostEqual, 1)
		convey.So(engine.GetStatic().Get(ErrorStats), convey.ShouldAlmostEqual, 0)
		r := engine.GetRuntimeStatus()
		convey.So(r.GetStartAt(), convey.ShouldBeLessThan, time.Now().Unix()+1)
		convey.So(r.GetDuration(), convey.ShouldBeLessThan, 1)
		convey.So(r.GetRestartAt(), convey.ShouldBeLessThan, time.Now().Unix()+1)
		convey.So(r.GetStatusOn().GetTypeName(), convey.ShouldContainSubstring, ON_STOP.GetTypeName())
		convey.So(r.GetStopAt(), convey.ShouldBeGreaterThan, 0)

	})
	convey.Convey("status control", t, func() {
		engine := NewTestEngine("controlSpider9")
		patch := gomonkey.ApplyFunc((*CrawlEngine).checkReadyDone, func(_ *CrawlEngine) bool {
			return false

		})
		defer patch.Reset()

		go engine.Execute("controlSpider9")
		time.Sleep(time.Second)

		r := engine.GetRuntimeStatus()
		t.Log(r.GetStatusOn().GetTypeName())
		convey.So(r.GetStatusOn().GetTypeName(), convey.ShouldContainSubstring, ON_START.GetTypeName())
		convey.So(engine.GetCurrentSpider().GetName(), convey.ShouldContainSubstring, "controlSpider9")
		r.SetStatus(ON_PAUSE)
		convey.So(r.GetStatusOn().GetTypeName(), convey.ShouldContainSubstring, ON_PAUSE.GetTypeName())
		time.Sleep(time.Second * 4)
		r.SetStatus(ON_STOP)

		const unknown StatusType = 5
		convey.So(unknown.GetTypeName(), convey.ShouldContainSubstring, "unknown")
	})

}
func TestBeforeStartError(t *testing.T) {
	convey.Convey("engine start before error", t, func() {

		engine := NewTestEngine("testStartBeforeErrorSpider")
		patch := gomonkey.ApplyFunc((*DefaultComponents).SpiderBeforeStart, func(_ *DefaultComponents, _ *CrawlEngine, _ SpiderInterface) error {
			return errors.New("ERROR Before start")

		})
		defer patch.Reset()
		engine.Execute("testStartBeforeErrorSpider")
		convey.So(engine.GetRuntimeStatus().StartAt, convey.ShouldAlmostEqual, 0)
	})
}
func TestEngineStartPanic(t *testing.T) {
	convey.Convey("engine start panic", t, func() {

		engine := NewTestEngine("testStartPanicSpider")
		patch := gomonkey.ApplyFunc((*CrawlEngine).Execute, func(_ *CrawlEngine, _ string) StatisticInterface {
			panic("output panic")

		})
		defer patch.Reset()
		f := func() { engine.Execute("testStartPanicSpider") }
		convey.So(f, convey.ShouldPanic)
		convey.So(engine.mutex.TryLock(), convey.ShouldBeTrue)
	})

}

func TestEngineErrorHandler(t *testing.T) {
	convey.Convey("test error handler", t, func() {
		engine := NewTestEngine("testErrorHandlerSpider")
		engine.spiders.SpidersModules["testErrorHandlerSpider"].(*TestSpider).FeedUrls = []string{"http://127.0.0.1:12345"}
		engine.Execute("testErrorHandlerSpider")
		convey.So(engine.components.GetStats().Get(RequestStats), convey.ShouldAlmostEqual, 1)
		convey.So(engine.components.GetStats().Get(ErrorStats), convey.ShouldAlmostEqual, 1)

	})

}

func TestProcessRequestError(t *testing.T) {
	convey.Convey("test process request error", t, func() {
		engine := NewTestEngine("testProcessRequestErrorSpider")
		m := TestDownloadMiddler2{9, "test"}
		engine.RegisterDownloadMiddlewares(m)
		ctx := NewTestRequest(engine.GetSpiders().SpidersModules["testProcessRequestErrorSpider"])
		err := engine.doDownload(ctx)
		convey.So(err.Error(), convey.ShouldContainSubstring, "process request fail")
	})

}

func TestProcessResponseError(t *testing.T) {
	convey.Convey("test process response error", t, func() {

		engine := NewTestEngine("testProcessResposeErrorSpider")
		m := TestDownloadMiddler2{9, "test"}
		patch := gomonkey.ApplyFunc(TestDownloadMiddler2.ProcessRequest, func(_ TestDownloadMiddler2, _ *Context) error {
			return nil
		})
		defer patch.Reset()
		engine.RegisterDownloadMiddlewares(m)
		ctx := NewTestRequest(engine.GetSpiders().SpidersModules["testProcessResposeErrorSpider"])
		err := engine.doDownload(ctx)
		convey.So(err, convey.ShouldBeNil)
		err = engine.doHandleResponse(ctx)
		convey.So(err.Error(), convey.ShouldContainSubstring, "process response fail")
		ctx.Response = nil
		err = engine.doHandleResponse(ctx)
		convey.So(err, convey.ShouldBeError, errors.New("response is nil"))

	})

}

func TestProcessItemError(t *testing.T) {

	convey.Convey("test process item error", t, func() {
		engine := NewTestEngine("testProcessItemErrorSpider")
		engine.RegisterPipelines(&TestItemPipeline4{4})
		engine.Execute("testProcessItemErrorSpider")
		convey.So(engine.components.GetStats().Get(ErrorStats), convey.ShouldAlmostEqual, 1)
	})
}
func TestParseError(t *testing.T) {

	convey.Convey("test parser error", t, func() {
		engine := NewTestEngine("testSpiderParseError")
		server := NewTestServer()
		spider, err := engine.spiders.GetSpider("testSpiderParseError")
		convey.So(err, convey.ShouldBeNil)
		patch := gomonkey.ApplyFunc((*TestSpider).Parser, func(_ *TestSpider, _ *Context, _ chan<- *Context) error {
			return errors.New("parse response error")

		})
		defer patch.Reset()
		request := NewRequest(server.URL+"/testGET", GET, testSpider.Parser)
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
		ctxPatch := gomonkey.ApplyFunc((*Context).Close, func(_ *Context) {})
		defer func() {
			patch.Reset()
			ctxPatch.Reset()
			restContext(ctx, url)
		}()
		ctx.Items = make(chan *ItemMeta, 16)
		f := engine.worker(ctx)
		err := f()
		convey.So(err, convey.ShouldBeNil)

		convey.So(ctx.Error.Error(), convey.ShouldContainSubstring, errMsg)
	})
}
func restContext(ctx *Context, url string) {
	ctx.Error = nil
	ctx.Request = NewRequest(url, GET, ctx.Spider.Parser)
	ctx.Response = nil

}
func TestWorkerErr(t *testing.T) {
	engine := NewTestEngine("wokerSpider")

	ctx := NewTestRequest(engine.GetSpiders().SpidersModules["wokerSpider"])
	url := ctx.Request.Url
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
	patch = gomonkey.ApplyFunc((*SpiderDownloader).Download, func(_ *SpiderDownloader, _ *Context) (*Response, error) {
		panic("call download panic")
	})
	wokerError(ctx, url, "call download panic", t, patch, engine)

}

func TestPostForm(t *testing.T) {
	convey.Convey("test tegenaria.RequestWithPostForm", t, func() {
		tServer := NewTestServer()
		mockRedis := miniredis.RunT(t)
		defer mockRedis.Close()
		engine := NewTestEngine("RequestWithPostFormSpider")
		spider1, _ := engine.GetSpiders().GetSpider("RequestWithPostFormSpider")

		components := engine.GetComponents()
		urlReq := fmt.Sprintf("%s/testForm", tServer.URL)
		form := url.Values{}
		form.Set("key", "form data")
		header := make(map[string]string)
		header["Content-Type"] = "application/x-www-form-urlencoded"
		request := NewRequest(urlReq, POST, spider1.Parser, RequestWithPostForm(form), RequestWithRequestHeader(header))
		ctx := NewContext(request, spider1)
		queue := components.GetQueue()
		queue.SetCurrentSpider(spider1)
		err := queue.Enqueue(ctx)
		convey.So(err, convey.ShouldBeNil)
		var c interface{}
		c, err = queue.Dequeue()
		convey.So(err, convey.ShouldBeNil)
		newCtx := c.(*Context)
		downloader := NewDownloader()
		resp, err := downloader.Download(newCtx)
		content, _ := resp.String()
		convey.So(err, convey.ShouldBeNil)
		convey.So(content, convey.ShouldContainSubstring, "form data")
	})
}
