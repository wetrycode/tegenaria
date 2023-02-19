package main

import (
	"fmt"
	"sync"

	"github.com/bsm/redislock"
	"github.com/wetrycode/tegenaria"
	"github.com/wetrycode/tegenaria/example/quotes"
	"github.com/wetrycode/tegenaria/metric"
)

func main() {
	// 构造分布式组件
	config := tegenaria.NewDistributedWorkerConfig("", "", 0)
	woker := tegenaria.NewDistributedWorker("127.0.0.1:6379", config)
	// 分布式组件加入到引擎
	opts := []tegenaria.EngineOption{tegenaria.EngineWithUniqueReq(false), tegenaria.EngineWithDistributedWorker(woker)}
	engine := quotes.NewQuotesEngine(opts...)

	host := tegenaria.Config.GetString("influxdb.host")
	port := tegenaria.Config.GetInt("influxdb.port")
	bucket := tegenaria.Config.GetString("influxdb.bucket")
	token := tegenaria.Config.GetString("influxdb.token")
	org := tegenaria.Config.GetString("influxdb.org")
	metrics := metric.NewCrawlMetricCollector(fmt.Sprintf("http://%s:%d", host, port), token, bucket, org, engine, redislock.New(woker.GetRDB()))
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		metrics.Start()
	}()

	engine.Execute()
	wg.Wait()
}
