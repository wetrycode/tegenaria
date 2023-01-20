package tegenaria

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
)
// StatsFieldType 统计指标的数据类型
type StatsFieldType string
const (
	// RequestStats 发起的请求总数
	RequestStats StatsFieldType = "requests"
	// ItemsStats 获取到的items总数
	ItemsStats StatsFieldType = "items"
	// DownloadFailStats 请求失败总数
	DownloadFailStats StatsFieldType = "download_fail"
	// ErrorStats 错误总数
	ErrorStats StatsFieldType = "errors"
)
// StatisticInterface 数据统计组件接口
type StatisticInterface interface {
	// IncrItemScraped 累加一条item
	IncrItemScraped()
	// IncrRequestSent 累加一条request
	IncrRequestSent()
	// IncrDownloadFail 累加一条下载失败数据
	IncrDownloadFail()
	// IncrErrorCount 累加一条错误数据
	IncrErrorCount()
	// GetItemScraped 获取拿到的item总数
	GetItemScraped() uint64
	// GetRequestSent 获取发送request的总数
	GetRequestSent() uint64
	// GetErrorCount 获取处理过程中生成的error总数
	GetErrorCount() uint64
	// GetDownloadFail 获取下载失败的总数
	GetDownloadFail() uint64
	// OutputStats 格式化统计指标
	OutputStats() map[string]uint64
	// Reset 重置指标统计组件,所有指标归零
	Reset() error
	// setCurrentSpider 设置当前正在运行的spider
	setCurrentSpider(spider string)
}
// Statistic 数据统计指标
type Statistic struct {
	// ItemScraped items总数
	ItemScraped  uint64 `json:"items"`
	// RequestSent 请求总数
	RequestSent  uint64 `json:"requets"`
	// DownloadFail 下载失败总数
	DownloadFail uint64 `json:"download_fail"`
	// ErrorCount 错误总数
	ErrorCount   uint64 `json:"errors"`
	// spider 当前正在运行的spider名
	spider       string `json:"-"`
}

// DistributeStatistic 分布式统计组件
type DistributeStatistic struct {
	// keyPrefix 缓存key前缀，默认tegenaria:v1:nodes
	keyPrefix     string
	// nodesKey 节点池的key
	nodesKey      string
	// rdb redis客户端实例
	rdb           redis.Cmdable
	// 调度该组件的wg
	wg            *sync.WaitGroup
	// afterResetTTL 重置数据之前缓存多久
	// 默认不缓存这些统计数据
	afterResetTTL time.Duration
	// spider 当前在运行的spider名
	spider        string
	// fields 所有参与统计的指标
	fields []StatsFieldType
}
// DistributeStatisticOption 分布式组件可选参数定义
type DistributeStatisticOption func(d *DistributeStatistic)
// NewStatistic 默认统计数据组件构造函数
func NewStatistic() *Statistic {
	return &Statistic{
		ItemScraped:  0,
		RequestSent:  0,
		DownloadFail: 0,
		ErrorCount:   0,
	}
}
// setCurrentSpider 设置当前的spider
func (s *Statistic) setCurrentSpider(spider string) {
	s.spider = spider
}
// IncrItemScraped 累加一条item
func (s *Statistic) IncrItemScraped() {
	atomic.AddUint64(&s.ItemScraped, 1)
}
// IncrRequestSent 累加一条request
func (s *Statistic) IncrRequestSent() {
	atomic.AddUint64(&s.RequestSent, 1)
}

// IncrDownloadFail 累加一条下载失败数据
func (s *Statistic) IncrDownloadFail() {
	atomic.AddUint64(&s.DownloadFail, 1)
}
// IncrErrorCount 累加捕获到的数量
func (s *Statistic) IncrErrorCount() {
	atomic.AddUint64(&s.ErrorCount, 1)
}

// GetItemScraped 获取拿到的item总数
func (s *Statistic) GetItemScraped() uint64 {
	return atomic.LoadUint64(&s.ItemScraped)
}

// GetRequestSent 获取发送的请求总数
func (s *Statistic) GetRequestSent() uint64 {
	return atomic.LoadUint64(&s.RequestSent)
}
// GetDownloadFail 获取下载失败的总数
func (s *Statistic) GetDownloadFail() uint64 {
	return atomic.LoadUint64(&s.DownloadFail)
}

// GetErrorCount 获取捕获到错误总数
func (s *Statistic) GetErrorCount() uint64 {
	return atomic.LoadUint64(&s.ErrorCount)
}

// OutputStats 格式化统计数据
func (s *Statistic) OutputStats() map[string]uint64 {
	result := map[string]uint64{}
	b, _ := json.Marshal(s)
	_ = json.Unmarshal(b, &result)
	return result
}
// Reset 重置统计数据
func (s *Statistic) Reset() error {
	atomic.StoreUint64(&s.DownloadFail, 0)
	atomic.StoreUint64(&s.ItemScraped, 0)
	atomic.StoreUint64(&s.RequestSent, 0)
	atomic.StoreUint64(&s.ErrorCount, 0)
	return nil
}
// setCurrentSpider 设置当前的spider名
func (s *DistributeStatistic) setCurrentSpider(spider string) {
	s.spider = spider
}
// NewDistributeStatistic 分布式数据统计组件构造函数
func NewDistributeStatistic(statsPrefixKey string, rdb redis.Cmdable, wg *sync.WaitGroup, opts ...DistributeStatisticOption) *DistributeStatistic {
	d := &DistributeStatistic{
		keyPrefix:     statsPrefixKey,
		nodesKey:      "tegenaria:v1:nodes",
		rdb:           rdb,
		wg:            wg,
		afterResetTTL: -1 * time.Second,
		fields: []StatsFieldType{ItemsStats,RequestStats,DownloadFailStats,ErrorStats},
	}
	for _, o := range opts {
		o(d)
	}
	return d
}
// IncrStats累加指定的统计指标
func (s *DistributeStatistic) IncrStats(field StatsFieldType) {
	f := func() error {
		return s.rdb.Incr(context.TODO(), fmt.Sprintf("%s:%s:%s", s.keyPrefix, s.spider, field)).Err()
	}
	funcs := []GoFunc{f}
	AddGo(s.wg, funcs...)
}

// IncrItemScraped 累加一条item
func (s *DistributeStatistic) IncrItemScraped() {
	s.IncrStats(ItemsStats)
}

// IncrRequestSent 累加一条request
func (s *DistributeStatistic) IncrRequestSent() {
	s.IncrStats(RequestStats)

}
// IncrErrorCount 累加一条错误
func (s *DistributeStatistic) IncrErrorCount() {
	s.IncrStats(ErrorStats)

}
// GetDownloadFail 累计获取下载失败的数量
func (s *DistributeStatistic) IncrDownloadFail() {
	s.IncrStats(DownloadFailStats)

}

// GetStatsField 获取指定的数据指标值
func (s *DistributeStatistic) GetStatsField(field StatsFieldType) uint64 {
	val, err := s.rdb.Get(context.TODO(), fmt.Sprintf("%s:%s:%s", s.keyPrefix, s.spider, field)).Int64()
	if err != nil {
		engineLog.Errorf("get %s stats error %s", field, err.Error())
		return 0
	}
	return uint64(val)
}
// GetItemScraped 获取items的值
func (s *DistributeStatistic) GetItemScraped() uint64 {
	return s.GetStatsField(ItemsStats)
}
// GetRequestSent 获取request 量
func (s *DistributeStatistic) GetRequestSent() uint64 {
	return s.GetStatsField(RequestStats)
}
// GetDownloadFail 获取下载失败数
func (s *DistributeStatistic) GetDownloadFail() uint64 {
	return s.GetStatsField(DownloadFailStats)

}
// GetErrorCount 获取错误数
func (s *DistributeStatistic) GetErrorCount() uint64 {
	return s.GetStatsField(ErrorStats)
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
// OutputStats 格式化输出所有的数据指标
func (s *DistributeStatistic) OutputStats() map[string]uint64 {

	// fields := []string{"items", "requests", "download_fail", "errors"}
	pipe := s.rdb.Pipeline()
	result := []*redis.StringCmd{}
	for _, field := range s.fields {
		key:=fmt.Sprintf("%s:%s:%s", s.keyPrefix, s.spider, field)
		result = append(result, pipe.Get(context.TODO(), key))
	}
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
