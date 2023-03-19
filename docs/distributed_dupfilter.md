#### 分布式去重组件
tegenaria 提供了基于redis实现的布隆过滤器用于`Request`对象的去重，该过滤器实现了`RFPDupeFilterInterface`[接口](dupfilter.md)  
其定义如下:
```go
type DistributedDupefilter struct {
	// rdb redis客户端支持redis单机实例和redis cluster集群模式
	rdb redis.Cmdable
	// getBloomFilterKey 布隆过滤器对应的生成key的函数，允许用户自定义
	getBloomFilterKey GetRDBKey
	// bloomP 布隆过滤器的容错率
	bloomP float64
	// bloomN 数据规模，比如1024 * 1024
	bloomN int
	// bloomM bitset 大小
	bloomM int
	// bloomK hash函数个数
	bloomK int
	// currentSpider 当前的spider
	currentSpider string

	// dupeFilter 去重组件
	dupeFilter *tegenaria.DefaultRFPDupeFilter
}
```

#### 说明

- 相关参数说明见[组件说明](distributed_components.md)  

- 去重流程和[默认的单机去重组件](dupfilter.md)基本一致，差别在于分布式组件将指纹hash存入redis