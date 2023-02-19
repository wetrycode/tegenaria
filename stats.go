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

package tegenaria

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sourcegraph/conc"
)

// StatsFieldType 统计指标的数据类型
type StatsFieldType string

// var (
//
//	REQUEST uint64 =
//
// )
var codeStatusName = [][]int{{100, 101}, {200, 206}, {300, 307}, {400, 417}, {500, 505}}

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

// StatisticInterface 数据统计组件接口
type StatisticInterface interface {
	GetAllStats() map[string]uint64

	setCurrentSpider(spider string)
	Incr(metric string)
	Get(metric string) uint64
}

// Statistic 数据统计指标
type Statistic struct {

	// spider 当前正在运行的spider名
	Metrics  map[string]*uint64
	spider   string `json:"-"`
	register sync.Map
}

// DistributeStatistic 分布式统计组件
type DistributeStatistic struct {
	// keyPrefix 缓存key前缀，默认tegenaria:v1:nodes
	keyPrefix string
	// nodesKey 节点池的key
	nodesKey string
	// rdb redis客户端实例
	rdb redis.Cmdable
	// 调度该组件的wg
	wg *conc.WaitGroup
	// afterResetTTL 重置数据之前缓存多久
	// 默认不缓存这些统计数据
	afterResetTTL time.Duration
	// spider 当前在运行的spider名
	spider string
	// fields 所有参与统计的指标
	fields   []string
	register sync.Map
}

// DistributeStatisticOption 分布式组件可选参数定义
type DistributeStatisticOption func(d *DistributeStatistic)

// NewStatistic 默认统计数据组件构造函数
func NewStatistic() *Statistic {
	m := map[string]*uint64{
		RequestStats:      new(uint64),
		DownloadFailStats: new(uint64),
		ItemsStats:        new(uint64),
		ErrorStats:        new(uint64),
	}
	for _, status := range codeStatusName {
		min, max := status[0], status[1]
		for i := min; i <= max; i++ {
			m[strconv.Itoa(i)] = new(uint64)
		}
	}
	for _, v := range m {
		atomic.StoreUint64(v, 0)

	}
	s := &Statistic{
		Metrics:  m,
		register: sync.Map{},
	}
	return s
}

// setCurrentSpider 设置当前的spider
func (s *Statistic) setCurrentSpider(spider string) {
	s.spider = spider
}

// Incr 新增一个指标值
func (s *Statistic) Incr(metrics string) {
	atomic.AddUint64(s.Metrics[metrics], 1)
	s.register.Store(metrics, true)
}

// Get 获取某个指标的数值
func (s *Statistic) Get(metric string) uint64 {
	return atomic.LoadUint64(s.Metrics[metric])
}

// GetAllStats 格式化统计数据
func (s *Statistic) GetAllStats() map[string]uint64 {
	result := make(map[string]uint64)
	s.register.Range(func(key any, value any) bool {
		k := key.(string)
		result[k] = s.Get(k)
		return true

	})
	return result
}

// setCurrentSpider 设置当前的spider名
func (s *DistributeStatistic) setCurrentSpider(spider string) {
	s.spider = spider
}

// NewDistributeStatistic 分布式数据统计组件构造函数
func NewDistributeStatistic(statsPrefixKey string, rdb redis.Cmdable, wg *conc.WaitGroup, opts ...DistributeStatisticOption) *DistributeStatistic {
	d := &DistributeStatistic{
		keyPrefix:     statsPrefixKey,
		nodesKey:      "tegenaria:v1:nodes",
		rdb:           rdb,
		wg:            wg,
		afterResetTTL: -1 * time.Second,
		fields:        []string{ItemsStats, RequestStats, DownloadFailStats, ErrorStats},
	}
	for _, o := range opts {
		o(d)
	}
	return d
}

// Incr 新增一个指标值
func (s *DistributeStatistic) Incr(metric string) {
	f := func() error {
		return s.rdb.Incr(context.TODO(), fmt.Sprintf("%s:%s:%s", s.keyPrefix, s.spider, metric)).Err()
	}
	funcs := []GoFunc{f}
	s.register.Store(metric, true)
	GoRunner(context.TODO(), s.wg, funcs...)
}

// Get 获取某一个指标
func (s *DistributeStatistic) Get(field string) uint64 {
	val, err := s.rdb.Get(context.TODO(), fmt.Sprintf("%s:%s:%s", s.keyPrefix, s.spider, field)).Int64()
	if err != nil {
		engineLog.Errorf("get %s stats error %s", field, err.Error())
		return 0
	}
	return uint64(val)
}

// Reset 重置各项指标
// 若afterResetTTL>0则为每一项指标设置ttl否则直接删除指标
func (s *DistributeStatistic) Reset() error {
	nodesKey := fmt.Sprintf("%s:%s", s.nodesKey, s.spider)
	members := s.rdb.SCard(context.TODO(), nodesKey).Val()
	if members <= 0 {
		return nil
	}
	// fields := []string{"items", "requests", "download_fail", "errors"}
	pipe := s.rdb.Pipeline()
	for _, field := range s.fields {
		key := fmt.Sprintf("%s:%s:%s", s.keyPrefix, s.spider, field)
		// ttl >-0则先设置ttl不直接删除
		if s.afterResetTTL > 0 {
			pipe.Expire(context.TODO(), key, s.afterResetTTL)
			continue
		}
		pipe.Del(context.TODO(), key)
	}
	_, err := pipe.Exec(context.TODO())
	return err
}

// DistributeStatisticAfterResetTTL 为分布式计数器设置重置之前的ttl
func DistributeStatisticAfterResetTTL(ttl time.Duration) DistributeStatisticOption {
	return func(d *DistributeStatistic) {
		d.afterResetTTL = ttl
	}
}

// GetAllStats 获取所有的数据指标
func (s *DistributeStatistic) GetAllStats() map[string]uint64 {

	// fields := []string{"items", "requests", "download_fail", "errors"}
	pipe := s.rdb.Pipeline()
	result := []*redis.StringCmd{}
	s.register.Range(func(key any, value any) bool {
		field := key.(string)
		k := fmt.Sprintf("%s:%s:%s", s.keyPrefix, s.spider, field)
		result = append(result, pipe.Get(context.TODO(), k))
		return true

	})
	_, err := pipe.Exec(context.TODO())
	if err != nil {
		engineLog.Errorf("output stats error %s", err.Error())
		return map[string]uint64{}
	}
	stats := map[string]uint64{}
	for index, r := range result {
		val, err := r.Result()
		if err != nil {
			engineLog.Errorf("get stats error %s", err.Error())
		}
		v, _ := strconv.ParseInt(val, 10, 64)
		stats[string(s.fields[index])] = uint64(v)
	}

	return stats
}
