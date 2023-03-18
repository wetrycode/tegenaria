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
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wetrycode/tegenaria"
)

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

func (d *DistributedComponents) GetDupefilter() tegenaria.RFPDupeFilterInterface {
	return d.dupefilter
}
func (d *DistributedComponents) GetQueue() tegenaria.CacheInterface {
	return d.queue
}
func (d *DistributedComponents) GetLimiter() tegenaria.LimitInterface {
	return d.limiter
}
func (d *DistributedComponents) GetStats() tegenaria.StatisticInterface {
	return d.statistic
}
func (d *DistributedComponents) GetEventHooks() tegenaria.EventHooksInterface {
	return d.events
}
func (d *DistributedComponents) CheckWorkersStop() bool {
	stopped, _ := d.worker.CheckAllNodesStop()
	return d.queue.IsEmpty() && stopped
}
func (d *DistributedComponents) SetCurrentSpider(spider tegenaria.SpiderInterface) {
	d.spider = spider
	d.worker.SetCurrentSpider(spider)
}
func (d *DistributedComponents) SpiderBeforeStart(engine *tegenaria.CrawlEngine, spider tegenaria.SpiderInterface) error {
	if !d.worker.IsMaster() {
		// 分布式模式下的启动流程
		// 如果节点角色不是master则检查是否有主节点在线
		// 若没有主节点在线则不启动爬虫
		start := time.Now()
		for {
			time.Sleep(3 * time.Second)
			live, err := d.worker.CheckMasterLive()
			logger.Infof("有主节点存在:%v", live)
			if (!live) || (err != nil) {
				if err != nil {
					panic(fmt.Sprintf("check master nodes status error %s", err.Error()))
				}
				logger.Warnf("正在等待主节点上线")
				// 超过30s直接panic
				if time.Since(start) > 30*time.Second {
					panic(tegenaria.ErrNoMaterNodeLive)
				}
				continue
			}
			break
		}
		// 从节点不需要启动StartRequest
		return errors.New("No need to start this node 'StartRequest'")
	}
	return nil
}
