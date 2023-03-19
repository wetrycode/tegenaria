#### 统计指标  

- tegenaria 提供了以下几个基础的指标字段  
```go
const (
	// RequestStats 发起的请求总数
	RequestStats string = "requests"
	// ItemsStats 获取到的items总数
	ItemsStats string = "items"
	// DownloadFailStats 请求失败总数
	DownloadFailStats string = "download_fail"
	// ErrorStats 错误总数
	ErrorStats string = "errors"
)
```
- stats接口定义如下:
```go
// StatisticInterface 数据统计组件接口
type StatisticInterface interface {
	// GetAllStats 获取所有的指标数据
	GetAllStats() map[string]uint64
	// Incr 指定的指标计数器自增1
	Incr(metric string)
	// Get 获取指标的数值
	Get(metric string) uint64
	// SetCurrentSpider 设置当前的爬虫实例
	SetCurrentSpider(spider SpiderInterface)
}
```

#### 默认的指标统计组件  
- 指标统计组件定义

```go
// Statistic 数据统计指标
type DefaultStatistic struct {

	// Metrics 指标-数值缓存
	Metrics  map[string]*uint64
	spider   SpiderInterface `json:"-"`
	// register 指标注册器,注册指标字段
	register sync.Map
}

// NewStatistic 默认统计数据组件构造函数
func NewDefaultStatistic() *DefaultStatistic {
	// 初始化指标
	m := map[string]*uint64{
		RequestStats:      new(uint64),
		DownloadFailStats: new(uint64),
		ItemsStats:        new(uint64),
		ErrorStats:        new(uint64),
	}
	// 指标初始化为0
	for _, status := range codeStatusName {
		min, max := status[0], status[1]
		for i := min; i <= max; i++ {
			m[strconv.Itoa(i)] = new(uint64)
		}
	}
	for _, v := range m {
		atomic.StoreUint64(v, 0)

	}
	s := &DefaultStatistic{
		Metrics:  m,
		register: sync.Map{},
	}
	return s
}
```

- tegenaria除了提供基本的指标外，还会动态统计请求状态码的数值  

- 用户可以根据需求实现该接口以满足指标统计的需求  

#### 示例

```go
	engine.Execute("example")
	engine.GetStatic().Get(tegenaria.DownloadFailStats)
	engine.GetStatic().Get(tegenaria.RequestStats)
	engine.GetStatic().Get(tegenaria.ItemsStats)
	engine.GetStatic().Get(tegenaria.ErrorStats)
	// 200 状态码总量
	engine.GetStatic().Get("200")
```
