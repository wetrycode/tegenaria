package metric

import (
	"context"
	"fmt"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/bsm/redislock"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/wetrycode/tegenaria"
)

var metricLog = tegenaria.GetLogger("metric")

// CrawlMetricCollector 数据指标采集器
type CrawlMetricCollector struct {
	influxdbWrite api.WriteAPIBlocking
	influxdbQuery api.QueryAPI
	engine        *tegenaria.CrawlEngine
	Locker        *redislock.Client
	currentSpider string
	bucket string
}

// NewInfluxdb 构建influxdb 客户端
func NewInfluxdb(serverURL string, token string, bucket string, org string) (api.WriteAPIBlocking, api.QueryAPI) {
	client := influxdb2.NewClientWithOptions(serverURL, token, influxdb2.DefaultOptions().SetUseGZip(true).SetMaxRetries(3))
	return client.WriteAPIBlocking(org, bucket), client.QueryAPI(org)
}

// NewCrawlMetricCollector 构建采集器
func NewCrawlMetricCollector(serverURL string, token string, bucket string, org string, engine *tegenaria.CrawlEngine, locker *redislock.Client) *CrawlMetricCollector {
	write, read := NewInfluxdb(serverURL, token, bucket, org)
	return &CrawlMetricCollector{
		influxdbWrite: write,
		influxdbQuery: read,
		engine:        engine,
		Locker:        locker,
	}
}

// Collect 搜集器
func (c *CrawlMetricCollector) Collect() {
	defer func() {
		if p := recover(); p != nil {
			metricLog.Errorf("采集数据错误:%s", p)
		}
	}()
	spider := c.engine.GetCurrentSpider().GetName()

	for key, value := range c.engine.GetStatic().GetAllStats() {
		metricLog.Infof("采集集到:%s的数据指标:%s:%d", spider, key, value)
		p := influxdb2.NewPointWithMeasurement(spider).
			AddField(key, atomic.LoadUint64(&value)).
			SetTime(time.Now())
		c.influxdbWrite.WritePoint(context.Background(), p)
	}
}

// Start 启动搜集器
func (c *CrawlMetricCollector) Start() {
	for {
		if c.engine.GetCurrentSpider() != nil {
			break
		}
		runtime.Gosched()
	}
	spider := c.engine.GetCurrentSpider().GetName()
	key := fmt.Sprintf("tegenaria:v1:metric:%s", spider)
	lock, err := c.Locker.Obtain(context.Background(), key, 90*time.Second, &redislock.Options{})
	if err != nil && err != redislock.ErrNotObtained {
		return
	}
	defer lock.Release(context.Background())

	ticker := time.NewTicker(time.Duration(time.Second * 10))
	for {
		<-ticker.C
		c.Collect()
		if c.engine.GetStatusOn() == tegenaria.ON_STOP {
			return
		}
	}
}

func(c *CrawlMetricCollector)GetAllStats() map[string]uint64{
	query := fmt.Sprintf(`from(bucket:"%s")
	|> filter(fn: (r) =>
		r._measurement == "%s"
	)
	|> group()  
	|> count()`, c.bucket, c.currentSpider)

	result, err := c.influxdbQuery.Query(context.TODO(), query)
	if err != nil {
		metricLog.Errorf("查询统计数据失败:%s", err.Error())
		return nil
	}
	ret :=map[string]uint64{}
	for result.Next() {
		if result.TableChanged() {
			metricLog.Infof("table: %s\n", result.TableMetadata().String())
		}
		r := result.Record().Values()
		metricLog.Infof("%v", r)
		for key, val:=range r{
			ret[key] = uint64(val.(float64))
		}

	}
	return ret
}

func(c *CrawlMetricCollector)setCurrentSpider(spider string){
	c.currentSpider = spider
}
func(c *CrawlMetricCollector)Incr(metric string){
	metricLog.Infof("采集集到:%s的数据指标:%s:%d", c.currentSpider, metric, 1)
	p := influxdb2.NewPointWithMeasurement(c.currentSpider).
		AddField(metric, 1).
		SetTime(time.Now())
	c.influxdbWrite.WritePoint(context.Background(), p)

}
func(c *CrawlMetricCollector)Get(metric string) uint64{
	query := fmt.Sprintf(`from(bucket:"%s")
	|> filter(fn: (r) =>
		r._measurement == "%s"
	)
	|> group(columns:["%s"])  
	|> count()`, c.bucket, c.currentSpider, metric)

	result, err := c.influxdbQuery.Query(context.TODO(), query)
	if err != nil {
		metricLog.Errorf("查询统计数据失败:%s", err.Error())
		return 0
	}
	for result.Next() {
		if result.TableChanged() {
			metricLog.Infof("table: %s\n", result.TableMetadata().String())
		}
		v := result.Record().Value()
		return uint64(v.(float64))

	}
	return 0
}
