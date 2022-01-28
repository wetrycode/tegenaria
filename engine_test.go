package tegenaria

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/wxnacy/wgo/arrays"
)

func newTestEngine(spiderName string) *SpiderEngine {
	engine := NewSpiderEngine()
	server := newTestServer()

	// register test spider
	engine.RegisterSpider(&TestSpider{NewBaseSpider(spiderName, []string{server.URL + "/testGET"})})

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
func newTestRequest() *Context {
	server := newTestServer()
	request := NewRequest(server.URL+"/testGET", GET, testParser)
	var MainCtx context.Context = context.Background()
	ctx := NewContext(request, WithContext(MainCtx))
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
	engine := NewSpiderEngine(
		EngineWithContext(context.Background()),
		EngineWithTimeout(1*time.Second),
		EngineWithAllowStatusCode([]uint64{404}),
		EngineWithUniqueReq(false),
		EngineWithSchedulerNum(2),
		EngineWithRequestNum(32),
		EngineWithReadCacheNum(4),
		EngineWithDownloader(NewDownloader()),
	)
	if engine.DownloadTimeout != 1*time.Second {
		t.Errorf("spider engine set download timeout value error")
	}
	if arrays.ContainsUint(engine.allowStatusCode, uint64(404)) != 0 {
		t.Errorf("spider engine set allow status code arrays value error\n")
	}
	if engine.filterDuplicateReq != false {
		t.Errorf("spider engine set disable request unique value error\n")
	}
	if engine.schedulerNum != 2 {
		t.Errorf("spider engine set schedulers number value error\n")

	}
	if cap(engine.cacheChan) != 32 {
		t.Errorf("spider engine set request number value error except %d, get %d\n", 32, cap(engine.cacheChan))

	}
	if engine.cacheReadNum != 4 {
		t.Errorf("spider engine set cache reader number value error\n")

	}

}

func TestCache(t *testing.T) {
	server := newTestServer()

	engine := NewSpiderEngine()
	engine.waitGroup.Add(1)

	// write new request
	ctx := newTestRequest()
	go engine.writeCache(ctx)
	engine.waitGroup.Wait()

	// read a request from cache
	engine.mainWaitGroup.Add(1)
	go engine.readCache()
	engine.isDone = true
	engine.mainWaitGroup.Wait()
	if len(engine.cacheChan) == 0 {
		t.Errorf("Read request from cache error get empty cache channel buffer")

	}
	new_req := <-engine.cacheChan
	if new_req != nil && new_req.Request.Url != server.URL+"/testGET" {
		t.Errorf("Read request from cache error except url %s, but get url %s\n", server.URL+"/testGET", new_req.Request.Url)
	}

}

func TestDoFilter(t *testing.T) {
	engine := NewSpiderEngine(EngineWithUniqueReq(true))
	ctx := newTestRequest()

	for i := 0; i < 3; i++ {
		engine.waitGroup.Add(1)
		go engine.writeCache(ctx)
	}
	engine.waitGroup.Wait()

	if engine.cache.getSize() != 1 {
		t.Errorf("Cache filter duplicate request fail except %d requests, but get %d requests\n", 1, engine.cache.getSize())
	}

}

func TestDoDownload(t *testing.T) {
	engine := newTestEngine("testSpider2")
	server := newTestServer()

	ctx := newTestRequest()

	engine.waitGroup.Add(1)
	go engine.doDownload(ctx)
	engine.waitGroup.Wait()

	value, ok := ctx.Request.Header["priority-0"]
	if !ok {
		t.Errorf("Download middlerwares not work except priority-0 header, but get nil\n")

	}
	if value != "0" {
		t.Errorf("Download middlerwares not work except priority-0 header value 0, but get %s\n", value)

	}

	result := <-engine.requestResultChan

	if result.DownloadResult.Error != nil {
		t.Errorf("Download result error get %s\n", result.DownloadResult.Error)

	}
	// test parser
	spider := &TestSpider{NewBaseSpider("testSpider2", []string{server.URL + "/testGET"})}
	engine.waitGroup.Add(1)
	go engine.doParse(spider, result)
	engine.waitGroup.Wait()
	itemCtx := <-engine.itemsChan
	i := itemCtx.Item.(*testItem)
	if i.test != "test" {
		t.Errorf("Download result parser fail except %s get %s\n", "test", i.test)

	}
	// test pipelines
	engine.waitGroup.Add(1)
	go engine.doPipelinesHandlers(spider, itemCtx)
	engine.waitGroup.Wait()

}
func TestDoParse(t *testing.T) {

}
func TestDoRequestResult(t *testing.T) {
	engine := newTestEngine("testSpider4")
	ctx := newTestRequest()
	ctx.Request.Url = newTestServer().URL + "/testTimeout"
	engine.SetDownloadTimeout(1 * time.Second)
	// test timeout
	engine.waitGroup.Add(1)
	go engine.doDownload(ctx)
	engine.waitGroup.Wait()
	result := <-engine.requestResultChan
	engine.waitGroup.Add(1)
	go engine.doRequestResult(result)
	engine.waitGroup.Wait()
	err := <-engine.errorChan
	if err == nil {
		t.Errorf("Download should be timeout but not \n")
	}

}

func TestAllowedStatusCode(t *testing.T) {
	engine := newTestEngine("testSpider5")
	ctx := newTestRequest()
	ctx.Request.Url = newTestServer().URL + "/test404"
	engine.SetAllowedStatus([]uint64{404, 403})
	// test 404 status codes
	engine.waitGroup.Add(1)
	go engine.doDownload(ctx)
	engine.waitGroup.Wait()
	result := <-engine.requestResultChan
	engine.waitGroup.Add(1)
	go engine.doRequestResult(result)
	engine.waitGroup.Wait()
	resp := <-engine.respChan
	if resp.DownloadResult.Response == nil {
		t.Errorf("Download response not should be empty \n")

	}
	if resp.DownloadResult.Response.Status != 404 {
		t.Errorf("Download response status should be 404 but get %d \n", resp.DownloadResult.Response.Status)
	}

}

func TestNotAllowedStatus(t *testing.T) {
	engine := newTestEngine("testSpider11")
	ctx := newTestRequest()
	ctx.Request.Url = newTestServer().URL + "/test403"
	engine.SetAllowedStatus([]uint64{404})
	// test 403 status codes
	engine.waitGroup.Add(1)
	go engine.doDownload(ctx)
	engine.waitGroup.Wait()
	result := <-engine.requestResultChan
	engine.waitGroup.Add(1)
	go engine.doRequestResult(result)
	engine.waitGroup.Wait()
	err := <-engine.errorChan

	if err.Error() != fmt.Sprintf("%s %d with context id %s", ErrNotAllowStatusCode.Error(), 403, err.CtxId) {
		t.Errorf("Download response error get %s\n", err.Error())
	}

}

func TestSpiderNotFound(t *testing.T) {
	defer func() {
		if p := recover(); p != nil {
			t.Errorf("Spider spider not should be found \n")

		}
	}()
	engine := newTestEngine("testSpider6")
	engine.Start("spider")
}
func TestSpiderDuplicate(t *testing.T) {
	defer func() {
		if p := recover(); p == nil {
			t.Errorf("Spider register should be get duplicate error \n")

		}
	}()
	engine := newTestEngine("testSpider7")
	engine.RegisterSpider(&TestSpider{NewBaseSpider("testSpider7", []string{})})
}
func TestSystemSignalHandle(t *testing.T) {
	engine := newTestEngine("testSpider8")
	engine.quitSignal <- os.Interrupt
	engine.Start("testSpider8")

}

func TestEngineStart(t *testing.T) {
	engine := newTestEngine("testSpider9")
	engine.Start("testSpider9")
	if engine.Stats.ErrorCount != 0 {
		t.Errorf("Spider crawl get %d errors \n", engine.Stats.ErrorCount)
	}
	if engine.Stats.RequestDownloaded != 1 {
		t.Errorf("Spider crawl except 1 download, but get %d \n", engine.Stats.RequestDownloaded)

	}
	if engine.Stats.ItemScraped != 1 {
		t.Errorf("Spider crawl except 1 item, but get %d \n", engine.Stats.ItemScraped)

	}
}
func TestEngineErrorHandler(t *testing.T) {
	engine := newTestEngine("testSpider10")
	engine.spiders.SpidersModules["testSpider10"].(*TestSpider).FeedUrls = []string{"http://127.0.0.1:12345"}
	engine.Start("testSpider10")
	if engine.Stats.ErrorCount != 1 {
		t.Errorf("Spider crawl except 1 error,but get %d \n", engine.Stats.ErrorCount)
	}
}

func TestProcessRequestError(t *testing.T) {
	engine := newTestEngine("testSpider19")
	m := TestDownloadMiddler2{9, "test"}
	engine.RegisterDownloadMiddlewares(m)

	ctx := newTestRequest()

	engine.waitGroup.Add(1)
	go engine.doDownload(ctx)
	engine.waitGroup.Wait()
	err := <-engine.errorChan
	if !strings.Contains(err.Error(), "process request fail") {
		t.Errorf("request should have error process request fail, but get %s\n", err.Error())
	}

}
func TestProcessResponseError(t *testing.T) {
	engine := newTestEngine("testSpider79")
	m := TestDownloadMiddler2{9, "test"}
	engine.RegisterDownloadMiddlewares(m)

	ctx := newTestRequest()
	engine.processResponse(ctx)
	err := <-engine.errorChan
	if !strings.Contains(err.Error(), "process response fail") {
		t.Errorf("request should have error process response fail, but get %s\n", err.Error())
	}
}

func TestProcessItemError(t *testing.T) {
	engine := newTestEngine("testSpider99")
	engine.RegisterPipelines(&TestItemPipeline4{4})
	engine.waitGroup.Add(1)
	m := make(map[string]string)
	m["a"] = "b"
	ctx := newTestRequest()

	item := NewItem(ctx, &testItem{"TEST", make([]int, 0)})
	go engine.doPipelinesHandlers(&TestSpider{NewBaseSpider("testSpider7", []string{})}, item)
	engine.waitGroup.Wait()
	err := <-engine.errorChan
	if !strings.Contains(err.Error(), "process item fail") {
		t.Errorf("request should have error process item fail, but get %s\n", err.Error())
	}
}
func TestParseError(t *testing.T) {
	engine := newTestEngine("testSpiderParseError")
	server := newTestServer()
	request := NewRequest(server.URL+"/testGET", GET, func(resp *Context, item chan<- *ItemMeta, req chan<- *Context) error {
		return errors.New("parse response error")
	})
	var MainCtx context.Context = context.Background()
	ctx := NewContext(request, WithContext(MainCtx))
	// test 403 status codes
	engine.waitGroup.Add(1)
	go engine.doDownload(ctx)
	engine.waitGroup.Wait()
	result := <-engine.requestResultChan
	engine.waitGroup.Add(1)
	spider, err := engine.spiders.GetSpider("testSpiderParseError")
	if err != nil {
		t.Errorf("Get spider testSpiderParseError")
	}
	go engine.doParse(spider, result)
	engine.waitGroup.Wait()
	err = <-engine.errorChan
	if !strings.Contains(err.Error(), "parse response error") {
		t.Errorf("Except parse response error but get %s\n", err.Error())

	}
}
