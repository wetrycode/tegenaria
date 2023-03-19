#### 数据指标采集器

tegenaria 提供了基于influxdb2数据指标采集器，该组件实现了`StatisticInterface`[接口](stats.md)  

##### 定义
```go
// CrawlMetricCollector 数据指标采集器
type CrawlMetricCollector struct {
	influxdbWrite api.WriteAPI
	influxdbQuery api.QueryAPI
	currentSpider string
	bucket        string
	influxClient  influxdb2.Client
}
// NewCrawlMetricCollector 构建采集器
func NewCrawlMetricCollector(serverURL string, token string, bucket string, org string) *CrawlMetricCollector {
	client := influxdb2.NewClientWithOptions(serverURL, token, influxdb2.DefaultOptions().SetMaxRetries(3))
	writer, reader := client.WriteAPI(org, bucket), client.QueryAPI(org)
	return &CrawlMetricCollector{
		influxdbWrite: writer,
		influxdbQuery: reader,
		bucket:        bucket,
		influxClient:  client,
	}
}
```

##### 参数配置

- 在构造```DistributedWorkerConfig```配置管理器时传入```InfluxdbConfig```进行参数设置

##### 示例

```go
	rdbConfig := distributed.NewRedisConfig("127.0.0.1:6379", "", "", 0)
	host := tegenaria.Config.GetString("influxdb.host")
	port := tegenaria.Config.GetInt("influxdb.port")
	bucket := tegenaria.Config.GetString("influxdb.bucket")
	token := tegenaria.Config.GetString("influxdb.token")
	org := tegenaria.Config.GetString("influxdb.org")
	influxdbConfig := distributed.NewInfluxdbConfig(fmt.Sprintf("http://%s:%d", host, port), token, bucket, org)
	config := distributed.NewDistributedWorkerConfig(rdbConfig, influxdbConfig)
	// 工作组件
	worker := distributed.NewDistributedWorker(config)
	components := distributed.NewDistributedComponents(config, worker, worker.GetRDB())
	// 分布式组件加入到引擎
	opts := []tegenaria.EngineOption{tegenaria.EngineWithUniqueReq(false), tegenaria.EngineWithComponents(components)}
	engine := quotes.NewQuotesEngine(opts...)

	engine.Execute("example")
```

##### 数据说明

- 组件会在引擎中进行埋点并实时记录指标数据，构成时序数据

- 该组件的查询接口获取的所有指标的值都是所有时段的`count`值，即全时全量求和

##### 其他

- 若要搭建数据看板或监控告警建议配合`prometheus`与`grafana `一起使用

- 用户可以通过实现`DistributedComponents`和`StatisticInterface`接口使用其他中间件来替换influxdb

