package tegenaria

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var engineLog *logrus.Entry = GetLogger("engine") // engineLog engine runtime logger

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

	Downloader Downloader

	// cache Request cache.
	// You can set your custom cache module,like redis
	cache CacheInterface

	// requestsChan *Request channel
	// sender is SpiderInterface.StartRequest
	// receiver is SpiderEngine.writeCache
	requestsChan chan *Context
	cacheChan    chan *Context
	// filterDuplicateReq flag if filter duplicate request fingerprint.
	// to filter duplicate request fingerprint set true or not set false.
	filterDuplicateReq bool
	// RFPDupeFilter request fingerprint BloomFilter
	// it will work if filterDuplicateReq is true
	RFPDupeFilter     RFPDupeFilterInterface
	startSpiderFinish bool

	isStop bool
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
	engineLog.Infof("Register %v priority pipeline success\n", pipeline)

}

// RegisterDownloadMiddlewares add a download middlewares
func (e *CrawlEngine) RegisterDownloadMiddlewares(middlewares MiddlewaresInterface) {
	e.downloaderMiddlewares = append(e.downloaderMiddlewares, middlewares)
	sort.Sort(e.downloaderMiddlewares)
}

// Start spider engine start.
// It will schedule all spider system
func (e *CrawlEngine) Start(spiderName string) {
	fmt.Printf("spider is ready to start")

	spider, ok := e.spiders.SpidersModules[spiderName]

	if !ok {
		panic(fmt.Sprintf("Spider %s not found", spiderName))
	}
	e.waitGroup.Add(1)

	go func(spiderName string) {
		defer func() {
			e.waitGroup.Done()
			e.startSpiderFinish = true
		}()
		e.startSpiderFinish = false

		spider.StartRequest(e.requestsChan)
	}(spiderName)

	e.waitGroup.Add(1)
	go e.recvRequest()
	e.waitGroup.Add(1)
	go e.Scheduler(spider)
	e.waitGroup.Wait()
}

func (e *CrawlEngine) Scheduler(spider SpiderInterface) {
	defer e.waitGroup.Done()
	for {
		if e.isStop{
			engineLog.Infof("调度已经结束")
			return 
		}
		req, err := e.cache.dequeue()
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		request := req.(*Context)
		engineLog.Infof("request %s is ready work", request.CtxId)
		e.waitGroup.Add(1)
		go e.worker(request)

	}
}
func (e *CrawlEngine) worker(ctx *Context) {
	defer func() {
		e.waitGroup.Done()
		if err := recover(); err != nil {
			err := NewError(ctx, fmt.Errorf("crawl error %s", err), ErrorWithRequest(ctx.Request))
			engineLog.Errorf("crawl error %s", err.Error())
		}
		if ctx.Error != nil {
			ctx.Spider.ErrorHandler(ctx, e.requestsChan)
		}
		ctx.Close()

	}()
	resp, err := e.doDownload(ctx)
	if err != nil {
		engineLog.Errorf("download error %s", err.Error())
		ctx.Error = err
		return
	}
	ctx.setResponse(resp)
	e.doParse(ctx)
	if ctx.Error != nil {
		return
	}
	pipeErr := e.doPipelinesHandlers(ctx)
	if pipeErr != nil {
		return
	}
}
func (e *CrawlEngine) recvRequest() {
	defer func() {
		e.waitGroup.Done()
	}()
	for {
		select {
		case req := <-e.requestsChan:
			e.writeCache(req)
		case <-time.After(time.Second * 3):
			if e.checkReadyDone() {
				e.isStop = true
				return
			}
		}
	}
}
func (e *CrawlEngine) checkReadyDone() bool {
	return e.startSpiderFinish && ctxManager.isEmpty()
}
func (e *CrawlEngine) writeCache(ctx *Context) {
	defer func() {
		// e.waitGroup.Done()
		if err := recover(); err != nil {
			err := NewError(ctx, fmt.Errorf("write cache error %s", err), ErrorWithRequest(ctx.Request))
			engineLog.Errorf("write cache error %s", err.Error())
			ctx.Error = err
		}
	}()
	var err error = nil
	if e.filterDuplicateReq {
		var ret bool = false
		ret, err = e.RFPDupeFilter.DoDupeFilter(ctx.Request)
		if err != nil {
			engineLog.WithField("request_id", ctx.CtxId).Errorf("request unique error %s", err.Error())
			return
		}
		if !ret {
			return
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

}

// doDownload handle request download
func (e *CrawlEngine) doDownload(ctx *Context) (*Response, *HandleError) {
	defer func() {
		// e.waitGroup.Done()
		if err := recover(); err != nil {
			err := NewError(ctx, fmt.Errorf("Download error %s", err), ErrorWithRequest(ctx.Request))
			engineLog.Errorf("Download error %s", err.Error())
		}
	}()
	// use download middleware to handle request object
	for _, middleware := range e.downloaderMiddlewares {
		err := middleware.ProcessRequest(ctx)
		if err != nil {
			engineLog.WithField("request_id", ctx.CtxId).Errorf("Middleware %s handle request error %s", middleware.GetName(), err.Error())
			ctx.Error = err
			return nil, NewError(ctx, err, ErrorWithRequest(ctx.Request))
		}
	}
	// incr request download number
	return e.Downloader.Download(ctx)

}

func (e *CrawlEngine) doParse(ctx *Context) error {
	defer func() {
		if err := recover(); err != nil {
			err := NewError(ctx, fmt.Errorf("parse error %s", err), ErrorWithRequest(ctx.Request))
			ctx.Error = err
			engineLog.Errorf("Parse error %s", err.Error())
		}
		close(ctx.Items)
	}()
	if ctx.Response == nil {
		return nil
	}
	return ctx.Spider.Parser(ctx, e.requestsChan)
}
func (e *CrawlEngine) doPipelinesHandlers(ctx *Context) error {
	defer func() {
		if err := recover(); err != nil {
			err := NewError(ctx, fmt.Errorf("pipeline error %s", err), ErrorWithRequest(ctx.Request))
			ctx.Error = err
			engineLog.Errorf("pipeline error %s", err.Error())
		}
	}()
	for item := range ctx.Items {
		engineLog.Infof("get item %v", item)
		for _, pipeline := range e.pipelines {
			err := pipeline.ProcessItem(ctx.Spider, item)
			if err != nil {
				engineLog.WithField("request_id", ctx.CtxId).Errorf("Pipeline %d handle item error %s", pipeline.GetPriority(), err.Error())
				ctx.Error = err
				return err
			}
		}
	}
	return nil
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

		filterDuplicateReq: true,
		RFPDupeFilter:      NewRFPDupeFilter(1024*4, 8),
		isStop: false,
	}
	for _, o := range opts {
		o(Engine)
	}
	return Engine
}
