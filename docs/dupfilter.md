#### 去重组件  

- 去重组件用于将请求推入队列前对```Request```对象进行去重过滤  

- 去重逻辑由三个部分组成:
    - 引擎参数,全局控制是否进入去重流程,在构建引擎是传入该参数
    ```go
    // EngineWithUniqueReq 是否进行去重处理,
    // true则进行去重处理，默认值为true
    func EngineWithUniqueReq(uniqueReq bool) EngineOption {
	return func(r *CrawlEngine) {
		r.filterDuplicateReq = uniqueReq
	    }
    }
    ```  
    - 请求对象参数，优先级高于引擎参数，控制单条请求是否进入去重流程，可以在构建`Request`对象时传入该参数  
    ```go
    // RequestWithDoNotFilter 设置当前请求是否进行过滤处理
    // true则认为该条请求无需进入去重流程,默认值为false
    func RequestWithDoNotFilter(doNotFilter bool) RequestOption {
    	return func(r *Request) {
    		r.DoNotFilter = doNotFilter
    	}
    }
    ```

    - 去重接口，负责实现去重逻辑，```DoDupeFilter```为去重逻辑，引擎根据返回值判断`Request`对象是否重复  
    若返回值为true则认为该请求重复否则为新的请求
    ```go
    // RFPDupeFilterInterface request 对象指纹计算和布隆过滤器去重
    type RFPDupeFilterInterface interface {
    	// Fingerprint request指纹计算
    	Fingerprint(ctx *Context) ([]byte, error)
    
    	// DoDupeFilter request去重
    	DoDupeFilter(ctx *Context) (bool, error)
    
    	SetCurrentSpider(spider SpiderInterface)
    }
    ```

#### 默认的去重组件

- 项目提供的默认的去重组件由[布隆过滤器](https://pkg.go.dev/github.com/bits-and-blooms/bloom/v3@v3.3.1)实现  

```go
// RFPDupeFilter 去重组件
type DefaultRFPDupeFilter struct {
	bloomFilter *bloom.BloomFilter
	spider      SpiderInterface
}

// NewRFPDupeFilter 新建去重组件
// bloomP容错率
// bloomN数据规模
func NewRFPDupeFilter(bloomP float64, bloomN int) *DefaultRFPDupeFilter {
	// 计算最佳的bit set大小
	bloomM := OptimalNumOfBits(bloomN, bloomP)
	// 计算最佳的哈希函数大小
	bloomK := OptimalNumOfHashFunctions(bloomN, bloomM)
	return &DefaultRFPDupeFilter{
		bloomFilter: bloom.New(uint(bloomM), uint(bloomK)),
	}
}
```
- 关键参数:
    - bloomP 主要控制的是容错率，莫默认值0.001  

    - bloomN 为预估的数据规模,默认值1024*1024  

    - bloomM bit set的大小由```OptimalNumOfBits```计算出最优值

    - bloomK 哈希函数大小,由```OptimalNumOfHashFunctions```计算出最优值

- 如何去重？
    - 先将请求参数，包括请求头、请求体、请求url参数进行统一格式的序列化  
    - 通过`sha128`算法计算出请求指纹  
    - 指纹传入布隆过滤器对其进行探测并判断该条请求是否重复  

- 如何修改默认参数？  
在构造```DefaultComponents```是传入参数```DefaultComponentsWithDupefilter(NewRFPDupeFilter(0.001，1024*1024))```即可  