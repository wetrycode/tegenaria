package main

import (
	"fmt"

	"github.com/wetrycode/tegenaria"
	"github.com/wetrycode/tegenaria/distributed"
	"github.com/wetrycode/tegenaria/example/quotes"
)

func main() {
	// 构造分布式组件
	rdbConfig := distributed.NewRedisConfig("127.0.0.1:6379", "", "", 0)
	host := tegenaria.Config.GetString("influxdb.host")
	port := tegenaria.Config.GetInt("influxdb.port")
	bucket := tegenaria.Config.GetString("influxdb.bucket")
	token := tegenaria.Config.GetString("influxdb.token")
	org := tegenaria.Config.GetString("influxdb.org")
	influxdbConfig := distributed.NewInfluxdbConfig(fmt.Sprintf("http://%s:%d", host, port), token, bucket, org)
	config := distributed.NewDistributedWorkerConfig(rdbConfig, influxdbConfig)
	worker := distributed.NewDistributedWorker(config)
	components := distributed.NewDistributedComponents(config, worker, worker.GetRDB())
	// 分布式组件加入到引擎
	opts := []tegenaria.EngineOption{tegenaria.EngineWithUniqueReq(false), tegenaria.EngineWithComponents(components)}
	engine := quotes.NewQuotesEngine(opts...)

	engine.Execute("example")
}
