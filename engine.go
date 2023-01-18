package tegenaria

import (
	"fmt"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var engineLog *logrus.Entry = GetLogger("engine") // engineLog engine runtime logger
type wokerUnit func(c *Context) error
type EventsWatcher func(ch chan EventType) error
type CheckMasterLive func() (bool, error)

// Engine is the main struct of tegenaria
// it is used to start a spider crawl
// and sechdule the spiders
type CrawlEngine struct {
	// spiders is the spider that will be used to crawl
	spiders *Spiders

	// pipelines items process chan.
	// Items should be handled by these pipenlines
	// Such as save item into databases
	pipelines ItemPipelines

	// middlewares are handle request object such as add proxy or header
	downloaderMiddlewares Middlewares

	// waitGroup the engineScheduler inner goroutine task wait group
	waitGroup *sync.WaitGroup

	downloader Downloader

	// cache Request cache.
	// You can set your custom cache module,like redis
	cache           CacheInterface
	limiter         LimitInterface
	checkMasterLive CheckMasterLive

	// requestsChan *Request channel
	// sender is SpiderInterface.StartRequest
	// receiver is SpiderEngine.writeCache
	requestsChan chan *Context
	cacheChan    chan *Context
	eventsChan   chan EventType
	// filterDuplicateReq flag if filter duplicate request fingerprint.
	// to filter duplicate request fingerprint set true or not set false.
	filterDuplicateReq bool
	// RFPDupeFilter request fingerprint BloomFilter
	// it will work if filterDuplicateReq is true
	RFPDupeFilter     RFPDupeFilterInterface
	startSpiderFinish bool
	statistic         StatisticInterface
	hooker            EventHooksInterface
	isStop            bool
	useDistributed    bool
	isMaster          bool
	mutex             sync.Mutex
}

// RegisterSpider register a spider to engine
func (e *CrawlEngine) RegisterSpiders(spider SpiderInterface) {
	err := e.spiders.Register(spider)
	if err != nil {
		panic(err)
	}
	engineLog.Infof("Register %s spider success\n", spider.GetName())
}

// RegisterPipelines add items handle pipelines
func (e *CrawlEngine) RegisterPipelines(pipeline PipelinesInterface) {
	e.pipelines = append(e.pipelines, pipeline)
	sort.Sort(e.pipelines)
	engineLog.Debugf("Register %v priority pipeline success\n", pipeline)

}

// RegisterDownloadMiddlewares add a download middlewares
func (e *CrawlEngine) RegisterDownloadMiddlewares(middlewares MiddlewaresInterface) {
	e.downloaderMiddlewares = append(e.downloaderMiddlewares, middlewares)
	sort.Sort(e.downloaderMiddlewares)
}
func (e *CrawlEngine) startSpider(spiderName string) GoFunc {
	_spiderName := spiderName
	return func() error {
		spider, ok := e.spiders.SpidersModules[_spiderName]

		if !ok {
			panic(fmt.Sprintf("Spider %s not found", _spiderName))
		}
		e.statistic.setCurrentSpider(spiderName)
		e.cache.setCurrentSpider(spiderName)
		if e.useDistributed {
			if e.isMaster {
				spider.StartRequest(e.requestsChan)
			} else {
				live, err := e.checkMasterLive()
				if !live || err != nil {
					if err != nil {
						engineLog.Errorf("check master nodes status error %s", err.Error())
					}
					panic("master nodes not live")
				}
			}
		} else {
			spider.StartRequest(e.requestsChan)
		}

		e.startSpiderFinish = true
		return nil
	}

}
func (e *CrawlEngine) Execute() {
	ExecuteCmd(e)
}

// Start spider engine start.
// It will schedule all spider system
func (e *CrawlEngine) start(spiderName string) StatisticInterface {
	e.mutex.Lock()
	defer func() {
		if p := recover(); p != nil {
			e.mutex.Unlock()
			panic(p)
		}
		e.mutex.Unlock()
	}()
	tasks := []GoFunc{e.startSpider(spiderName), e.recvRequest, e.Scheduler}
	wg := &sync.WaitGroup{}
	hookTasks := []GoFunc{e.EventsWatcherRunner}
	AddGo(wg, hookTasks...)
	AddGo(e.waitGroup, tasks...)
	e.eventsChan <- START
	e.waitGroup.Wait()
	e.eventsChan <- EXIT
	engineLog.Infof("Wating engine to stop...")
	wg.Wait()
	stats := e.statistic.OutputStats()
	s := Map2String(stats)
	engineLog.Infof(s)
	return e.statistic
}

func (e *CrawlEngine) EventsWatcherRunner() error {
	err := e.hooker.EventsWatcher(e.eventsChan)
	if err != nil {
		return fmt.Errorf("events watcher task execution error %s", err.Error())
	}
	return nil
}
func (e *CrawlEngine) Scheduler() error {
	for {
		if e.isStop {
			return nil
		}
		req, err := e.cache.dequeue()
		if err != nil {
			runtime.Gosched()
			continue
		}
		request := req.(*Context)
		f := []GoFunc{e.worker(request)}
		AddGo(e.waitGroup, f...)

	}
}
func (e *CrawlEngine) worker(ctx *Context) GoFunc {
	c := ctx
	return func() error {
		defer func() {
			if err := recover(); err != nil {
				c.setError(fmt.Sprintf("crawl error %s", err))
			}
			if c.Error != nil {
				e.statistic.IncrErrorCount()
				e.eventsChan <- ERROR
				c.Spider.ErrorHandler(c, e.requestsChan)
			}
			c.Close()
		}()
		e.eventsChan <- HEARTBEAT
		units := []wokerUnit{e.doDownload, e.doHandleResponse, e.doParse, e.doPipelinesHandlers}
		for _, unit := range units {
			err := unit(c)
			if err != nil {
				c.Error = NewError(c, err)
				return nil
			}
		}
		return nil
	}
}
func (e *CrawlEngine) recvRequest() error {
	for {
		select {
		case req := <-e.requestsChan:
			e.writeCache(req)
		case <-time.After(time.Second * 3):
			if e.checkReadyDone() {
				e.isStop = true
				return nil
			}

		}
		runtime.Gosched()
	}
}
func (e *CrawlEngine) checkReadyDone() bool {
	return e.startSpiderFinish && ctxManager.isEmpty() && e.cache.isEmpty()
}
func (e *CrawlEngine) writeCache(ctx *Context) error {
	defer func() {
		// e.waitGroup.Done()
		if err := recover(); err != nil {
			ctx.setError(fmt.Sprintf("write cache error %s", err))
		}
		// 写入分布式组件后主动删除
		if e.useDistributed {
			ctx.Close()
		}
	}()
	var err error = nil
	if e.filterDuplicateReq {
		var ret bool = false
		ret, err = e.RFPDupeFilter.DoDupeFilter(ctx)
		if err != nil {
			engineLog.WithField("request_id", ctx.CtxId).Errorf("request unique error %s", err.Error())
			return err
		}
		if ret {
			return nil
		}
	}
	err = e.cache.enqueue(ctx)
	if err != nil {
		engineLog.WithField("request_id", ctx.CtxId).Warnf("Cache enqueue error %s", err.Error())
		time.Sleep(time.Second)
		e.cacheChan <- ctx
	}
	err = nil
	ctx.Error = err
	return nil

}

// doDownload handle request download
func (e *CrawlEngine) doDownload(ctx *Context) error {
	var err error = nil
	defer func() {
		if p := recover(); p != nil {
			ctx.setError(fmt.Sprintf("Download error %s", p))
		}
		if err != nil || ctx.Err() != nil {
			e.statistic.IncrDownloadFail()
		}
	}()
	// use download middleware to handle request object
	for _, middleware := range e.downloaderMiddlewares {
		err := middleware.ProcessRequest(ctx)
		if err != nil {
			engineLog.WithField("request_id", ctx.CtxId).Errorf("Middleware %s handle request error %s", middleware.GetName(), err.Error())
			return err
		}
	}
	// incr request download number
	err = e.limiter.checkAndWaitLimiterPass()
	if err != nil {
		return err
	}
	e.statistic.IncrRequestSent()
	var rsp *Response = nil
	engineLog.WithField("request_id", ctx.CtxId).Infof("%s request ready to download", ctx.CtxId)
	rsp, err = e.downloader.Download(ctx)
	if err != nil {
		return err
	}
	ctx.setResponse(rsp)
	return nil

}

// doRequestResult handle download respose result
func (e *CrawlEngine) doHandleResponse(ctx *Context) error {
	defer func() {
		if p := recover(); p != nil {
			ctx.setError(fmt.Sprintf("handle response error %s", p))
		}
	}()
	if ctx.Response == nil {
		err := fmt.Errorf("response is nil")
		engineLog.WithField("request_id", ctx.CtxId).Errorf("Request is fail with error %s", err.Error())
		return err
	}
	if !e.downloader.CheckStatus(uint64(ctx.Response.Status), ctx.Request.AllowStatusCode) {
		err := fmt.Errorf("%s %d", ErrNotAllowStatusCode.Error(), ctx.Response.Status)
		engineLog.WithField("request_id", ctx.CtxId).Errorf("Request is fail with error %s", err.Error())
		return err
	}

	if len(e.downloaderMiddlewares) == 0 {
		return nil
	}
	for index := range e.downloaderMiddlewares {
		middleware := e.downloaderMiddlewares[len(e.downloaderMiddlewares)-index-1]
		err := middleware.ProcessResponse(ctx, e.requestsChan)
		if err != nil {
			engineLog.WithField("request_id", ctx.CtxId).Errorf("Middleware %s handle response error %s", middleware.GetName(), err.Error())
			return err
		}
	}
	return nil

}
func (e *CrawlEngine) doParse(ctx *Context) error {
	defer func() {
		if err := recover(); err != nil {
			ctx.setError(fmt.Sprintf("parse error %s", err))
		}
		close(ctx.Items)
	}()
	if ctx.Response == nil {
		return nil
	}
	engineLog.WithField("request_id", ctx.CtxId).Infof("%s request response ready to parse", ctx.CtxId)

	parserErr := ctx.Request.Parser(ctx, e.requestsChan)
	if parserErr != nil {
		engineLog.Errorf("%s", parserErr.Error())
	}
	return parserErr

}
func (e *CrawlEngine) doPipelinesHandlers(ctx *Context) error {
	defer func() {
		if err := recover(); err != nil {
			ctx.setError(fmt.Sprintf("pipeline error %s", err))
		}
	}()
	for item := range ctx.Items {
		e.statistic.IncrItemScraped()
		engineLog.WithField("request_id", ctx.CtxId).Infof("%s item is scraped", item.CtxId)
		for _, pipeline := range e.pipelines {
			err := pipeline.ProcessItem(ctx.Spider, item)
			if err != nil {
				engineLog.WithField("request_id", ctx.CtxId).Errorf("Pipeline %d handle item error %s", pipeline.GetPriority(), err.Error())
				// ctx.Error = err
				return err
			}
		}
	}
	return nil
}
func (e *CrawlEngine) GetSpiders() *Spiders {
	return e.spiders
}
func (e *CrawlEngine) Close() {
	close(e.requestsChan)
	close(e.cacheChan)
	err := e.statistic.Reset()
	if err != nil {
		engineLog.Errorf("reset statistic error %s", err.Error())

	}
}
func NewEngine(opts ...EngineOption) *CrawlEngine {
	Engine := &CrawlEngine{
		waitGroup:             &sync.WaitGroup{},
		spiders:               NewSpiders(),
		requestsChan:          make(chan *Context, 1024),
		pipelines:             make(ItemPipelines, 0),
		downloaderMiddlewares: make(Middlewares, 0),
		cache:                 NewRequestCache(),
		cacheChan:             make(chan *Context, 512),
		eventsChan:            make(chan EventType, 16),
		statistic:             NewStatistic(),
		filterDuplicateReq:    true,
		RFPDupeFilter:         NewRFPDupeFilter(0.001, 1024*4),
		isStop:                false,
		useDistributed:        false,
		isMaster:              true,
		checkMasterLive:       func() (bool, error) { return true, nil },
		limiter:               NewDefaultLimiter(32),
		downloader:            NewDownloader(),
		hooker:                NewDefualtHooks(),
	}
	for _, o := range opts {
		o(Engine)
	}
	return Engine
}
