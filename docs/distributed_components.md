#### 分布式组件
- Tegenaria通过```ComponentInterface```接口实现了一组分布式部署所需的模块，其定义如下:
```go
type DistributedComponents struct {
	dupefilter *DistributedDupefilter
	queue      *DistributedQueue
	limiter    *LeakyBucketLimiterWithRdb
	statistic  *CrawlMetricCollector
	events     *DistributedHooks
	worker     tegenaria.DistributedWorkerInterface
	spider     tegenaria.SpiderInterface
}
func NewDistributedComponents(config *DistributedWorkerConfig, worker tegenaria.DistributedWorkerInterface, rdb redis.Cmdable) *DistributedComponents {
	worker.SetMaster(config.isMaster)
	d := &DistributedComponents{
		dupefilter: NewDistributedDupefilter(config.bloomN, config.bloomP, rdb, config.getBloomFilterKey),
		queue:      NewDistributedQueue(rdb, config.getQueueKey),
		limiter:    NewLeakyBucketLimiterWithRdb(config.LimiterRate, rdb, config.getLimitKey),
		statistic:  NewCrawlMetricCollector(config.influxdb.influxdbServer, config.influxdb.influxdbToken, config.influxdb.influxdbBucket, config.influxdb.influxdbOrg),
		events:     NewDistributedHooks(worker),
		worker:     worker,
	}

	return d
}
```

- 分布式组件引入了两个外部的中间件:
    - redis:
        - 基于List实现请求队列  

        - 为分布式限速器提供数据缓存  

        - 分布式节点状态维护

        - 用于实现布隆过滤器
    
    - infulxdb2:
        - 存储按时序存储统计指标

#### 说明

- ```dupefilter``` 基于redis实现的布隆过滤器，用于请求对象的去重处理，是[`RFPDupeFilterInterface`接口](dupfilter.md)的实现实例对象

- ```queue``` 是基于redis list数据结构实现的```Context```对象消息队列,实现了[`CacheInterface`接口](queue.md)

- ```limiter``` 基于redis实现的漏斗算法限速器,控制分布式场景下的请求并发量，实现了[`LimitInterface`接口](limit.md)

- ```statistic``` 基于influxdb2实现的指标数据采集器,实现了[`StatisticInterface`接口](stats.md)  

- ```events``` 基于redis实现的事件监听器，主要是监听和控制节点的状态,实现了[`EventHooksInterface`接口](event.md)

- ```worker``` 基于redis实现的分布式节点维护组件，主要用于控制节点的注册、删除和状态查询，实现了[`DistributedWorkerInterface`接口](worker.md)

#### 配置组件

- 分布式组件引入了额外的配置管理模块```DistributedWorkerConfig```,用于管理分布式模式下的配置项，其定义如下:
```go
type DistributedWorkerConfig struct {
	redisConfig *RedisConfig
	// LimiterRate 限速大小
	LimiterRate int
	// bloomP 布隆过滤器的容错率
	bloomP float64
	// bloomN 数据规模，比如1024 * 1024
	bloomN int
	// isMaster 是否是主节点
	isMaster bool
	influxdb *InfluxdbConfig
	// getLimitKey 获取限速器redis key前缀
	getLimitKey GetRDBKey
	// getBloomFilterKey 布隆过滤器redis key前缀
	getBloomFilterKey GetRDBKey
	// getQueueKey 消息队列的key
	getQueueKey GetRDBKey
}
// NewDistributedWorkerConfig 新建分布式组件的配置
func NewDistributedWorkerConfig(rdbConfig *RedisConfig, influxdbConfig *InfluxdbConfig, opts ...DistributeOptions) *DistributedWorkerConfig {
	config := &DistributedWorkerConfig{
		redisConfig:       rdbConfig,
		influxdb:          influxdbConfig,
		LimiterRate:       32,
		bloomP:            0.001,
		bloomN:            1024 * 256,
		isMaster:          true,
		getLimitKey:       getLimiterDefaultKey,
		getBloomFilterKey: getBloomFilterKey,
		getQueueKey:       getQueueKey,
	}
	for _, opt := range opts {
		opt(config)
	}
	return config
}
```

- ```redisConfig``` redis实例的配置参数,
    ```go
    type RedisConfig struct {
    	// RedisAddr redis 地址
    	RedisAddr string
    	// RedisPasswd redis 密码
    	RedisPasswd string
    	// RedisUsername redis 用户名
    	RedisUsername string
    	// RedisDB redis 数据库索引 index
    	RedisDB uint32
    	// RdbConnectionsSize 连接池大小
    	RdbConnectionsSize uint64
    	// RdbTimeout redis 超时时间
    	RdbTimeout time.Duration
    	// RdbMaxRetry redis操作失败后的重试次数
    	RdbMaxRetry int
    }
    ```

- ```LimiterRate``` 限速器的速率   

- ```bloomP``` 布隆过滤器容错率，默认值为0.001，可以通过```DistributedWithBloomP(bloomP float64) DistributeOptions```进行设置

- ```bloomN``` 布隆过滤器数据规模,默认值为1024*1024，可以通过```DistributedWithBloomN(bloomN int) DistributeOptions```进行设置

- ```isMaster```节点角色标记,默认值为true,当前节点的角色为master,通过```DistributedWithSlave() DistributeOptions```将节点角色修改为slave 

- ```influxdb``` influxdb2客户端配置参数  
    ```go
    // InfluxdbConfig influxdb配置
    type InfluxdbConfig struct {
    	// influxdbServer influxdb服务链接
    	influxdbServer string
    	// influxdbToken influxdb api token
    	influxdbToken string
    	// influxdbBucket influxdb bucket 名称
    	influxdbBucket string
    	// influxdbOrg influxdb org 名称
    	influxdbOrg string
    }
    ```

- ```getLimitKey``` 限速器相关redis key前缀的生成函数

    - 默认采用如下的方式生成  

    ```go
    // getLimiterDefaultKey 限速器默认的key
    // 返回前缀和过期时间
    func getLimiterDefaultKey() (string, time.Duration) {
    	return "tegenaria:v1:limiter", 0 * time.Second
    }
    ```

    - 自定义生成方式可以通过可选参数```DistributedWithGetLimitKey(keyFunc GetRDBKey) DistributeOptions```传入

    - 完整的限速器缓存key结构为```{prefix}:{spider_name}```,例如`tegenaria:v1:limiter:example`

- ```getBloomFilterKey``` 布隆过滤器缓存key前缀生成器

    - 默认采用如下的方式生成  

    ```go
    // getBloomFilterKey 自定义的布隆过滤器key生成函数
    func getBloomFilterKey() (string, time.Duration) {
    	return "tegenaria:v1:bf", 0 * time.Second
    }
    ```
    - 自定义生成方式通过可选参数```DistributedWithGetBFKey(keyFunc GetRDBKey) DistributeOptions```传入    

    - 完整的限速器缓存key结构为```{prefix}:{spider_name}```,例如`tegenaria:v1:bf:example`

- ```getQueueKey``` 请求对象的消息队列key前缀生成函数

    - 默认采用如下的方式生成前缀 

    ```go
    // getQueueKey 自定义的request缓存队列key生成函数
    func getQueueKey() (string, time.Duration) {
    	return "tegenaria:v1:request", 0 * time.Second
    
    }
    ```
    - 自定义生成方式通过可选参数```DistributedWithGetQueueKey(keyFunc GetRDBKey) DistributeOptions```传入    

    - 完整的限速器缓存key结构为```{prefix}:{spider_name}```,例如`tegenaria:v1:request:example`

####  其他配置可选参数

- ```DistributedWithConnectionsSize(size int) DistributeOptions``` 控制redis连接池的大小  

- ```DistributedWithRdbTimeout(timeout time.Duration) DistributeOptions``` 控制redis超时时间 

- ```DistributedWithRdbMaxRetry(retry int) DistributeOptions``` 控制redis操作的重试次数  

- ```DistributedWithLimiterRate(rate int) DistributeOptions``` 控制并发速率

#### 示例

分布式组件使用方法可以参照[分布式模式](distributed.md)