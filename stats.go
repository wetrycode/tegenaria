package tegenaria

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/go-redis/redis/v8"
)

type StatisticInterface interface {
	IncrItemScraped()
	IncrRequestSent()
	IncrDownloadFail()
	IncrErrorCount()
	GetItemScraped() uint64
	GetRequestSent() uint64
	GetErrorCount() uint64
	GetDownloadFail() uint64
	OutputStats() map[string]uint64
	Reset()
	setCurrentSpider(spider string)
}
type Statistic struct {
	ItemScraped  uint64 `json:"items"`
	RequestSent  uint64 `json:"requets"`
	DownloadFail uint64 `json:"download_fail"`
	ErrorCount   uint64 `json:"errors"`
	spider string `json:"-"`
}

// type needLockIncr func()

type DistributeStatistic struct {
	keyPrefix string
	nodesKey string
	rdb       redis.Cmdable
	wg *sync.WaitGroup
	spider string
}

func NewStatistic() *Statistic {
	return &Statistic{
		ItemScraped:  0,
		RequestSent:  0,
		DownloadFail: 0,
		ErrorCount:   0,
	}
}
func (s *Statistic)	setCurrentSpider(spider string){
	s.spider = spider
}
func (s *Statistic) IncrItemScraped() {
	atomic.AddUint64(&s.ItemScraped, 1)
}

func (s *Statistic) IncrRequestSent() {
	atomic.AddUint64(&s.RequestSent, 1)
}
func (s *Statistic) IncrDownloadFail() {
	atomic.AddUint64(&s.DownloadFail, 1)
}

func (s *Statistic) IncrErrorCount() {
	atomic.AddUint64(&s.ErrorCount, 1)
}

func (s *Statistic) GetItemScraped() uint64 {
	return atomic.LoadUint64(&s.ItemScraped)
}

func (s *Statistic) GetRequestSent() uint64 {
	return atomic.LoadUint64(&s.RequestSent)
}
func (s *Statistic) GetDownloadFail() uint64 {
	return atomic.LoadUint64(&s.DownloadFail)
}

func (s *Statistic) GetErrorCount() uint64 {
	return atomic.LoadUint64(&s.ErrorCount)
}

func (s *Statistic) OutputStats() map[string]uint64 {
	result := map[string]uint64{}
	b, _ := json.Marshal(s)
	_ = json.Unmarshal(b, &result)
	return result
}
func(s *Statistic)Reset(){
	atomic.StoreUint64(&s.DownloadFail,0)
	atomic.StoreUint64(&s.ItemScraped,0)
	atomic.StoreUint64(&s.RequestSent,0)
	atomic.StoreUint64(&s.ErrorCount,0)

}
func (s *DistributeStatistic)	setCurrentSpider(spider string){
	s.spider = spider
}
func NewDistributeStatistic(statsPrefixKey string, rdb redis.Cmdable, wg *sync.WaitGroup) *DistributeStatistic{
	return &DistributeStatistic{
		keyPrefix: statsPrefixKey,
		nodesKey: "tegenaria:v1:nodes",
		rdb: rdb,
		wg:wg,
	}
}
func(s *DistributeStatistic)IncrStats(field string){
	f := func(){
		s.rdb.Incr(context.TODO(), fmt.Sprintf("%s:%s:%s", s.keyPrefix,s.spider, field))
	}
	funcs:=[]GoFunc{f}
	GoSyncWait(s.wg, funcs...)
}
func (s *DistributeStatistic) IncrItemScraped() {
	s.IncrStats("items")
}

func (s *DistributeStatistic) IncrRequestSent() {
	s.IncrStats("requests")

}
func (s *DistributeStatistic) IncrDownloadFail() {
	s.IncrStats("download_fail")

}

func (s *DistributeStatistic) IncrErrorCount() {
	s.IncrStats("errors")

}
func (s *DistributeStatistic) GetStatsField(field string) uint64 {
	val, err := s.rdb.Get(context.TODO(), fmt.Sprintf("%s:%s:%s", s.keyPrefix,s.spider, field)).Int64()
	if err != nil {
		engineLog.Errorf("get %s stats error %s", field, err.Error())
		return 0
	}
	return uint64(val)
}
func (s *DistributeStatistic) GetItemScraped() uint64 {
	return s.GetStatsField("items")
}

func (s *DistributeStatistic) GetRequestSent() uint64 {
	return s.GetStatsField("requests")
}
func (s *DistributeStatistic) GetDownloadFail() uint64 {
	return s.GetStatsField("download_fail")

}

func (s *DistributeStatistic) GetErrorCount() uint64 {
	return s.GetStatsField("errors")
}
func(s *DistributeStatistic)Reset(){
	members:=s.rdb.SCard(context.TODO(), fmt.Sprintf("%s:%s",s.nodesKey, s.spider)).Val()
	if members <=0{
		return
	}
	fields := []string{"items", "requests", "download_fail", "errors"}
	pipe := s.rdb.Pipeline()
	for _, field := range fields {
		pipe.Del(context.TODO(), fmt.Sprintf("%s:%s", s.keyPrefix, field))
	}
	pipe.Exec(context.TODO())
}
func (s *DistributeStatistic) OutputStats() map[string]uint64 {
	
	fields := []string{"items", "requests", "download_fail", "errors"}
	pipe := s.rdb.Pipeline()
	result := []*redis.StringCmd{}
	defer s.Reset()
	for _, field := range fields {
		result = append(result, pipe.Get(context.TODO(), fmt.Sprintf("%s:%s:%s", s.keyPrefix, s.spider, field)))
	}
	_, err := pipe.Exec(context.TODO())
	if err != nil {
		engineLog.Errorf("output stats error %s", err.Error())
		return map[string]uint64{}
	}
	stats := map[string]uint64{}
	for index, r := range result {
		val, err := r.Result()
		if err!=nil{
			engineLog.Errorf("get stats error %s", err.Error())
		}
		v, _ := strconv.ParseInt(val, 10, 64)
		stats[fields[index]] = uint64(v)
	}

	return stats
}
