package tegenaria

import (
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/smartystreets/goconvey/convey"
)

func newTestStats(mockRedis *miniredis.Miniredis, t *testing.T, opts ...DistributeStatisticOption) *DistributeStatistic {

	rdb := redis.NewClient(&redis.Options{
		Addr: mockRedis.Addr(),
	})
	wg := &sync.WaitGroup{}
	stats := NewDistributeStatistic("tegenaria:v1:stats", rdb, wg, opts...)
	stats.setCurrentSpider("distributedStatsSpider")

	worker := NewDistributedWorker(mockRedis.Addr(), NewDistributedWorkerConfig("", "", 0))
	worker.setCurrentSpider("distributedStatsSpider")
	err := worker.AddNode()
	convey.So(err, convey.ShouldBeNil)
	stats.IncrRequestSent()
	stats.IncrItemScraped()
	stats.IncrErrorCount()
	stats.IncrDownloadFail()
	wg.Wait()
	return stats
}
func TestDistributedStats(t *testing.T) {
	convey.Convey("test distribute stats", t, func() {
		mockRedis := miniredis.RunT(t)

		defer mockRedis.Close()
		stats := newTestStats(mockRedis, t, DistributeStatisticAfterResetTTL(10*time.Second))
		result := stats.OutputStats()
		convey.So(len(result), convey.ShouldBeGreaterThan, 0)
		for _, r := range result {
			convey.So(r, convey.ShouldAlmostEqual, 1)
		}
		funcs := []func() uint64{}
		funcs = append(funcs, stats.GetDownloadFail)
		funcs = append(funcs, stats.GetErrorCount)
		funcs = append(funcs, stats.GetItemScraped)
		funcs = append(funcs, stats.GetRequestSent)
		for _, f := range funcs {
			val := f()
			convey.So(val, convey.ShouldAlmostEqual, 1)
		}
		err := stats.Reset()
		convey.So(err, convey.ShouldBeNil)
		mockRedis.FastForward(20 * time.Second)
		for _, f := range funcs {
			val := f()
			convey.So(val, convey.ShouldAlmostEqual, 0)
		}
	})
	convey.Convey("test stats no afterTTL", t, func() {
		mockRedis := miniredis.RunT(t)
		defer mockRedis.Close()
		stats := newTestStats(mockRedis, t)
		result := stats.OutputStats()
		convey.So(len(result), convey.ShouldBeGreaterThan, 0)

		for _, r := range result {
			convey.So(r, convey.ShouldAlmostEqual, 1)
		}
		err := stats.Reset()
		convey.So(err, convey.ShouldBeNil)
		funcs := []func() uint64{}
		funcs = append(funcs, stats.GetDownloadFail)
		funcs = append(funcs, stats.GetErrorCount)
		funcs = append(funcs, stats.GetItemScraped)
		funcs = append(funcs, stats.GetRequestSent)
		for _, f := range funcs {
			val := f()
			convey.So(val, convey.ShouldAlmostEqual, 0)
		}
	})
}
