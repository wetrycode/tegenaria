#### 组件接口

一个完整的组件接口的实现应当包含以下必要的模块:  

- [请求队列](queue.md) 用于缓存请求，控制请求的输入和输出  

- [去重模块](dupfilter.md) 用于请求在进入队列之前的去重处理  

- [限速器](limit.md) 用于控制请求的频率  

- [指标统计](stats.md) 用于记录运行过程中产生的指标  

- [事件监听器](events.md) 用于监听和处理运行过程中产生的事件,例如启动、停止、错误及引擎心跳  

- [爬虫运行前置动作](before.md) 用于控制爬虫正式运行前的动作

## 接口定义
- 定义 
```go
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
```
- 组件接口允许用户实现自定义的组件模块和组件本身，提供引擎的扩展能力  

- 通过组件接口可以实现自定义的请求队列及其他的组件模块，项目自带的分布式组件就是基于组件接口实现的  

#### 默认的组件定义如下  

```go
// DefaultComponents 默认的组件
type DefaultComponents struct {
	// dupefilter 默认的去重过滤模块
	dupefilter *DefaultRFPDupeFilter
	// queue 默认的请求队列
	queue      *DefaultQueue
	// limiter 默认限速器
	limiter    *DefaultLimiter
	// statistic 指标统计组件
	statistic  *DefaultStatistic
	// events 事件监听器
	events     *DefaultHooks
	// 当前运行的爬虫实例
	spider     SpiderInterface
}
```

- 默认的组件包含了全部的必要模块

- 如何使用自定义的组件:
	- 实现`ComponentInterface`接口  

	- 在创建引擎时通过可选参数```EngineWithComponents(components ComponentInterface) EngineOption```传入引擎进行设置

	- 示例
	```go
	engine := NewEngine(EngineWithComponents(NewDefaultComponents()))
	```