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

package distributed

import (
	"context"
	"fmt"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/wetrycode/tegenaria"
)

var metricLog = tegenaria.GetLogger("metric")

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

// GetAllStats 获取所有字段的总量数据
func (c *CrawlMetricCollector) GetAllStats() map[string]uint64 {
	query := fmt.Sprintf(`from(bucket:"%s")
	|> range(start: -100y)
	|> filter(fn: (r) =>
		r._measurement == "%s"
	)
	|> group(columns:["_field"])  
	|> count(column:"_value")
	|> yield(name:"count")`, c.bucket, c.currentSpider)

	result, err := c.influxdbQuery.Query(context.TODO(), query)
	if err != nil {
		metricLog.Errorf("查询统计数据失败:%s", err.Error())
		return nil
	}
	ret := map[string]uint64{}

	for result.Next() {
		if result.TableChanged() {
			metricLog.Infof("table: %s\n", result.TableMetadata().String())
		}
		field := result.Record().ValueByKey("_field").(string)
		value := result.Record().ValueByKey("_value")
		logger.Infof("%v,%v", field, value)
		ret[field] = uint64(tegenaria.Interface2Uint(value))

	}
	return ret
}

func (c *CrawlMetricCollector) SetCurrentSpider(spider tegenaria.SpiderInterface) {
	c.currentSpider = spider.GetName()
}
func (c *CrawlMetricCollector) Incr(metric string) {
	metricLog.Infof("采集集到:%s的数据指标:%s:%d", c.currentSpider, metric, 1)
	p := influxdb2.NewPointWithMeasurement(c.currentSpider).
		AddField(metric, 1).
		SetTime(time.Now())
	c.influxdbWrite.WritePoint(p)

}

// Get 提取某一个指标的统计数据的总量
func (c *CrawlMetricCollector) Get(metric string) uint64 {
	query := fmt.Sprintf(`from(bucket:"%s")
	|> range(start: -100y)
	|> filter(fn: (r) =>
		r._measurement == "%s" and
		r._field == "%s"
	)
	|> count(column:"_value")`, c.bucket, c.currentSpider, metric)

	result, err := c.influxdbQuery.Query(context.TODO(), query)
	if err != nil {
		metricLog.Errorf("查询统计数据失败:%s", err.Error())
		return 0
	}
	result.Next()
	if result.TableChanged() {
		metricLog.Infof("table: %s\n", result.TableMetadata().String())
	}
	table := result.Record()
	if table != nil {
		return uint64(tegenaria.Interface2Uint(table.Value()))
	}
	return 0
}
