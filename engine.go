// MIT License

// Copyright (c) 2023 wetrycode

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package tegenaria

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"sort"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var engineLog *logrus.Entry = GetLogger("engine") // engineLog engine runtime logger
// wokerUnit context处理单元
type wokerUnit func(c *Context) error

// EventsWatcher 事件监听器
type EventsWatcher func(ch chan EventType) error

// CheckMasterLive 检查所有的master节点是否都在线
type CheckMasterLive func() (bool, error)

// CrawlEngine 引擎是整个框架数据流调度核心
type CrawlEngine struct {
	// spiders 已经注册的spider
	// 引擎调度的spider实例从此处根据爬虫名获取
	spiders *Spiders

	// pipelines 处理item的pipelines
	pipelines ItemPipelines

	// middlewares 注册的下载中间件
	downloaderMiddlewares Middlewares

	// waitGroup 引擎调度任务协程控制器
	waitGroup *sync.WaitGroup
	// downloader 下载器
	downloader Downloader
	// cache request队列缓存
	// 可以自定义实现cache接口，例如通过redis实现缓存
	cache CacheInterface
	// limiter 限速器，用于并发控制
	limiter LimitInterface
	// checkMasterLive 检查所有的master是否都在线
	checkMasterLive CheckMasterLive

	// requestsChan *Request channel
	// 整个系统产生的request对象都应该通过该channel
	// 提交到引擎
	requestsChan chan *Context
	// cacheChan 缓存Context数据数据流管道
	// 主要用于与cache交互
	cacheChan chan *Context
	// eventsChan 事件管道
	// 数据抓取过程中发起的事件都需要通过该channel与引擎进行交互
	eventsChan chan EventType
	// filterDuplicateReq 是否进行去重处理
	filterDuplicateReq bool
	// RFPDupeFilter 去重组件
	rfpDupeFilter RFPDupeFilterInterface
	// startSpiderFinish spider.StartRequest是否已经执行结束
	startSpiderFinish bool
	// statistic 数据统计组件
	statistic StatisticInterface
	// hooker 事件监听和处理组件
	hooker EventHooksInterface
	// isStop 当前的引擎是否已经停止处理数据
	// 引擎会每三秒种调用checkReadyDone检查所有的任务
	// 是否都已经结束，是则将isStop设置为true
	isStop bool
	// useDistributed 是否启用分布式年模式
	useDistributed bool
	// isMaster是否是master节点
	isMaster bool
	// mutex spider 启动锁
	// 同一进程下只能启动一个spider
	mutex sync.Mutex
	// interval 定时任务执行时间间隔
	interval time.Duration
}

// RegisterSpider 将spider实例注册到引擎的 spiders
func (e *CrawlEngine) RegisterSpiders(spider SpiderInterface) {
	err := e.spiders.Register(spider)
	if err != nil {
		panic(err)
	}
	engineLog.Infof("Register %s spider success\n", spider.GetName())
}

// RegisterPipelines 注册pipelines到引擎
func (e *CrawlEngine) RegisterPipelines(pipeline PipelinesInterface) {
	e.pipelines = append(e.pipelines, pipeline)
	sort.Sort(e.pipelines)
	engineLog.Debugf("Register %v priority pipeline success\n", pipeline)

}

// RegisterDownloadMiddlewares 注册下载中间件到引擎
func (e *CrawlEngine) RegisterDownloadMiddlewares(middlewares MiddlewaresInterface) {
	e.downloaderMiddlewares = append(e.downloaderMiddlewares, middlewares)
	sort.Sort(e.downloaderMiddlewares)
}

// startSpider 通过爬虫名启动爬虫
func (e *CrawlEngine) startSpider(spiderName string) GoFunc {
	_spiderName := spiderName
	return func() error {
		spider, ok := e.spiders.SpidersModules[_spiderName]

		if !ok {
			panic(fmt.Sprintf("Spider %s not found", _spiderName))
		}
		// 对相关组件设置当前的spider
		e.statistic.setCurrentSpider(spiderName)
		e.cache.setCurrentSpider(spiderName)
		if e.useDistributed {
			// 分布式模式下非master组件不启动StartRequest
			if e.isMaster {
				spider.StartRequest(e.requestsChan)
			} else {
				live, err := e.checkMasterLive()
				if !live || err != nil {
					if err != nil {
						engineLog.Errorf("check master nodes status error %s", err.Error())
					}
					// 没有在线的master节点
					// 不能启动slove节点
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

// Execute 通过命令行启动spider
func (e *CrawlEngine) Execute() {
	ExecuteCmd(e)
	defer rootCmd.ResetCommands()

}

// start spider 爬虫启动器
func (e *CrawlEngine) start(spiderName string) StatisticInterface {
	e.mutex.Lock()
	defer func() {
		if p := recover(); p != nil {
			e.mutex.Unlock()
			panic(p)
		}
		e.mutex.Unlock()
	}()
	// 引入引擎所有的组件，并通过协程执行
	tasks := []GoFunc{e.startSpider(spiderName), e.recvRequest, e.Scheduler}
	wg := &sync.WaitGroup{}
	// 事件监听器，通过协程执行
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
	e.startSpiderFinish = false
	e.isStop = false
	return e.statistic
}
func (e *CrawlEngine) startWithTicker(spiderName string) {
	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()
	for {
		engineLog.Infof("进入下一轮")
		e.start(spiderName)
		engineLog.Infof("完成一轮抓取")
		<-ticker.C
	}

}

// EventsWatcherRunner 事件监听器运行组件
func (e *CrawlEngine) EventsWatcherRunner() error {
	err := e.hooker.EventsWatcher(e.eventsChan)
	if err != nil {
		return fmt.Errorf("events watcher task execution error %s", err.Error())
	}
	return nil
}

// Scheduler 调度器
func (e *CrawlEngine) Scheduler() error {
	for {
		if e.isStop {
			return nil
		}
		// 从缓存队列中读取请求对象
		req, err := e.cache.dequeue()
		if err != nil {
			runtime.Gosched()
			continue
		}
		request := req.(*Context)
		// 对request进行处理
		f := []GoFunc{e.worker(request)}
		AddGo(e.waitGroup, f...)

	}
}

// worker request处理器，单个context生成周期都基于此
// 该函数包含了整个request和context的处理过程
func (e *CrawlEngine) worker(ctx *Context) GoFunc {
	c := ctx
	return func() error {
		defer func() {
			if c.Error != nil {
				// 新增一个错误
				e.statistic.IncrErrorCount()
				// 提交一个错误事件
				e.eventsChan <- ERROR
				// 将错误交给自定义的spider错误处理函数
				c.Spider.ErrorHandler(c, e.requestsChan)
			}
			// 处理结束回收context
			c.Close()
		}()
		// 发起心跳检查
		e.eventsChan <- HEARTBEAT
		// item启用新的协程进行处理
		wg := &sync.WaitGroup{}
		funcs := []GoFunc{func() error {
			err := e.doPipelinesHandlers(c)
			if err != nil {
				c.Error = NewError(c, err)
			}
			return err
		},
		}
		AddGo(wg, funcs...)
		// 处理request的所有工作单元
		units := []wokerUnit{e.doDownload, e.doHandleResponse, e.doParse}
		// 依次执行工作单元
		for _, unit := range units {
			err := unit(c)
			if err != nil || c.Error != nil {
				if c.Error == nil {
					c.Error = NewError(c, err)
				}
				break
			}
		}
		close(c.Items)
		wg.Wait()
		return nil
	}
}

// recvRequest 从requestsChan读取context对象
func (e *CrawlEngine) recvRequest() error {
	for {
		select {
		case req := <-e.requestsChan:
			e.writeCache(req)
		// 每三秒钟检查一次所有的任务是否都已经结束
		case <-time.After(time.Second * 3):
			if e.checkReadyDone() {
				e.isStop = true
				return nil
			}

		}
		runtime.Gosched()
	}
}

// checkReadyDone 检查任务状态
// 主要检查spider.StartRequest是否已经执行完成
// 所有的context是否都已经关闭
// 队列是否为空
func (e *CrawlEngine) checkReadyDone() bool {
	return e.startSpiderFinish && ctxManager.isEmpty() && e.cache.isEmpty()
}

// writeCache 将Context 写入缓存
func (e *CrawlEngine) writeCache(ctx *Context) error {
	var isDuplicated bool = false
	defer func() {
		if err := recover(); err != nil {
			ctx.setError(fmt.Sprintf("write cache error %s", err), string(debug.Stack()))
		}
		// 写入分布式组件后主动删除
		if e.useDistributed || isDuplicated {
			ctx.Close()
		}
	}()
	var err error = nil
	// 是否进入去重流程
	if e.filterDuplicateReq && !ctx.Request.DoNotFilter {
		ret, err := e.rfpDupeFilter.DoDupeFilter(ctx)
		if err != nil {
			isDuplicated = true
			engineLog.WithField("request_id", ctx.CtxId).Errorf("request unique error %s", err.Error())
			return err
		}
		// request重复则直接返回
		if ret {
			isDuplicated = true
			return nil
		}
	}
	// ctx进入缓存队列
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

// doDownload 对context执行下载操作
func (e *CrawlEngine) doDownload(ctx *Context) error {
	var err error = nil
	defer func() {
		if p := recover(); p != nil {
			ctx.setError(fmt.Sprintf("Download error %s", p), string(debug.Stack()))
		}
		if err != nil || ctx.Err() != nil {
			e.statistic.IncrDownloadFail()
		}
	}()
	// 通过下载中间件对request进行处理
	// 按优先级调用ProcessRequest
	for _, middleware := range e.downloaderMiddlewares {
		err := middleware.ProcessRequest(ctx)
		if err != nil {
			engineLog.WithField("request_id", ctx.CtxId).Errorf("Middleware %s handle request error %s", middleware.GetName(), err.Error())
			return err
		}
	}
	// 执行限速器，速率过高则等待
	err = e.limiter.checkAndWaitLimiterPass()
	if err != nil {
		return err
	}
	// 增加请求发送量
	e.statistic.IncrRequestSent()
	engineLog.WithField("request_id", ctx.CtxId).Infof("%s request ready to download", ctx.CtxId)
	rsp, err := e.downloader.Download(ctx)
	if err != nil {
		return err
	}
	ctx.setResponse(rsp)
	return nil

}

// doHandleResponse 处理请求响应
func (e *CrawlEngine) doHandleResponse(ctx *Context) error {
	defer func() {
		if p := recover(); p != nil {
			ctx.setError(fmt.Sprintf("handle response error %s", p), string(debug.Stack()))
		}
	}()
	if ctx.Response == nil {
		err := fmt.Errorf("response is nil")
		engineLog.WithField("request_id", ctx.CtxId).Errorf("Request is fail with error %s", err.Error())
		return err
	}
	// 检查状态码是否合法
	if !e.downloader.CheckStatus(uint64(ctx.Response.Status), ctx.Request.AllowStatusCode) {
		err := fmt.Errorf("%s %d", ErrNotAllowStatusCode.Error(), ctx.Response.Status)
		engineLog.WithField("request_id", ctx.CtxId).Errorf("Request is fail with error %s", err.Error())
		return err
	}

	if len(e.downloaderMiddlewares) == 0 {
		return nil
	}
	// 逆优先级调用ProcessResponse
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

// doParse 调用解析逻辑
func (e *CrawlEngine) doParse(ctx *Context) error {
	defer func() {
		if err := recover(); err != nil {
			ctx.setError(fmt.Sprintf("parse error %s", err), string(debug.Stack()))
		}
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

// doPipelinesHandlers 通过pipeline 处理item
func (e *CrawlEngine) doPipelinesHandlers(ctx *Context) error {
	defer func() {
		if err := recover(); err != nil {
			ctx.setError(fmt.Sprintf("pipeline error %s", err), string(debug.Stack()))
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

// GetSpiders 获取所有的已经注册到引擎的spider实例
func (e *CrawlEngine) GetSpiders() *Spiders {
	return e.spiders
}

// Close 关闭引擎
func (e *CrawlEngine) Close() {
	close(e.requestsChan)
	close(e.cacheChan)
	err := e.statistic.Reset()
	if err != nil {
		engineLog.Errorf("reset statistic error %s", err.Error())

	}
}

// 构建新的引擎
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
		rfpDupeFilter:         NewRFPDupeFilter(0.001, 1024*4),
		isStop:                false,
		useDistributed:        false,
		isMaster:              true,
		checkMasterLive:       func() (bool, error) { return true, nil },
		limiter:               NewDefaultLimiter(32),
		downloader:            NewDownloader(),
		hooker:                NewDefaultHooks(),
		interval:              -1 * time.Second,
	}
	for _, o := range opts {
		o(Engine)
	}
	return Engine
}
