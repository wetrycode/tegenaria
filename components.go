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

// EventsWatcher 事件监听器
type EventsWatcher func(ch chan EventType) error

// CheckMasterLive 检查所有的master节点是否都在线
type CheckMasterLive func() (bool, error)

// ComponentInterface 系统组件接口
// 包含了爬虫系统运行的必要组件
type ComponentInterface interface {
	// GetDupefilter 获取过滤器组件
	GetDupefilter() RFPDupeFilterInterface
	// GetQueue 获取请求队列接口
	GetQueue() CacheInterface
	// GetLimiter 限速器组件
	GetLimiter() LimitInterface
	// GetStats 指标统计组件
	GetStats() StatisticInterface
	// GetEventHooks 事件监控组件
	GetEventHooks() EventHooksInterface
	// CheckWorkersStop 爬虫停止的条件
	CheckWorkersStop() bool
	// SetCurrentSpider 当前正在运行的爬虫实例
	SetCurrentSpider(spider SpiderInterface)
	// SpiderBeforeStart 启动StartRequest之前的动作
	SpiderBeforeStart(engine *CrawlEngine, spider SpiderInterface) error
}

type DefaultComponents struct {
	dupefilter *DefaultRFPDupeFilter
	queue      *DefaultQueue
	limiter    *DefaultLimiter
	statistic  *DefaultStatistic
	events     *DefaultHooks
	spider     SpiderInterface
}
type DefaultComponentsOption func(d *DefaultComponents)

func NewDefaultComponents(opts ...DefaultComponentsOption) *DefaultComponents {
	d := &DefaultComponents{
		dupefilter: NewRFPDupeFilter(0.001, 1024*1024),
		queue:      NewDefaultQueue(1024 * 1024),
		limiter:    NewDefaultLimiter(16),
		statistic:  NewDefaultStatistic(),
		events:     NewDefaultHooks(),
	}
	for _, o := range opts {
		o(d)
	}
	return d
}

func (d *DefaultComponents) GetDupefilter() RFPDupeFilterInterface {
	return d.dupefilter
}
func (d *DefaultComponents) GetQueue() CacheInterface {
	return d.queue
}
func (d *DefaultComponents) GetLimiter() LimitInterface {
	return d.limiter
}
func (d *DefaultComponents) GetStats() StatisticInterface {
	return d.statistic
}
func (d *DefaultComponents) GetEventHooks() EventHooksInterface {
	return d.events
}
func (d *DefaultComponents) CheckWorkersStop() bool {
	return d.queue.IsEmpty()
}
func (d *DefaultComponents) SetCurrentSpider(spider SpiderInterface) {
	d.spider = spider
}
func (d *DefaultComponents) SpiderBeforeStart(engine *CrawlEngine, spider SpiderInterface) error {
	return nil
}
func DefaultComponentsWithDupefilter(dupefilter *DefaultRFPDupeFilter) DefaultComponentsOption {
	return func(r *DefaultComponents) {
		r.dupefilter = dupefilter

	}
}

func DefaultComponentsWithDefaultQueue(queue *DefaultQueue) DefaultComponentsOption {
	return func(r *DefaultComponents) {
		r.queue = queue

	}
}
func DefaultComponentsWithDefaultLimiter(limiter *DefaultLimiter) DefaultComponentsOption {
	return func(r *DefaultComponents) {
		r.limiter = limiter

	}
}
func DefaultComponentsWithDefaultStatistic(statistic *DefaultStatistic) DefaultComponentsOption {
	return func(r *DefaultComponents) {
		r.statistic = statistic

	}
}
func DefaultComponentsWithDefaultHooks(events *DefaultHooks) DefaultComponentsOption {
	return func(r *DefaultComponents) {
		r.events = events

	}
}
