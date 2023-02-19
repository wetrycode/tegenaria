package tegenaria

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/smartystreets/goconvey/convey"
	"github.com/sourcegraph/conc"
)

func newTestStats(mockRedis *miniredis.Miniredis, t *testing.T, opts ...DistributeStatisticOption) *DistributeStatistic {

	rdb := redis.NewClient(&redis.Options{
		Addr: mockRedis.Addr(),
	})
	wg := &conc.WaitGroup{}
	stats := NewDistributeStatistic("tegenaria:v1:stats", rdb, wg, opts...)
	stats.setCurrentSpider("distributedStatsSpider")

	worker := NewDistributedWorker(mockRedis.Addr(), NewDistributedWorkerConfig("", "", 0))
	worker.setCurrentSpider("distributedStatsSpider")
	err := worker.AddNode()
	convey.So(err, convey.ShouldBeNil)
	stats.Incr(RequestStats)
	stats.Incr(ItemsStats)
	stats.Incr(ErrorStats)
	stats.Incr(DownloadFailStats)
	wg.Wait()
	return stats
}
func TestDistributedStats(t *testing.T) {
	convey.Convey("test distribute stats", t, func() {
		mockRedis := miniredis.RunT(t)

		defer mockRedis.Close()
		stats := newTestStats(mockRedis, t, DistributeStatisticAfterResetTTL(10*time.Second))
		result := stats.GetAllStats()
		convey.So(len(result), convey.ShouldBeGreaterThan, 0)
		for _, r := range result {
			convey.So(r, convey.ShouldAlmostEqual, 1)
		}
		values := []uint64{}
		values = append(values, stats.Get(DownloadFailStats))
		values = append(values, stats.Get(ErrorStats))
		values = append(values, stats.Get(ItemsStats))
		values = append(values, stats.Get(RequestStats))
		for _, v := range values {
			convey.So(v, convey.ShouldAlmostEqual, 1)
		}
		err := stats.Reset()
		convey.So(err, convey.ShouldBeNil)
		mockRedis.FastForward(20 * time.Second)
		metrics := []string{DownloadFailStats, ErrorStats, ItemsStats, RequestStats}

		for _, f := range metrics {
			val := stats.Get(f)
			convey.So(val, convey.ShouldAlmostEqual, 0)
		}
	})
	convey.Convey("test stats no afterTTL", t, func() {
		mockRedis := miniredis.RunT(t)
		defer mockRedis.Close()
		stats := newTestStats(mockRedis, t)
		result := stats.GetAllStats()
		convey.So(len(result), convey.ShouldBeGreaterThan, 0)

		for _, r := range result {
			convey.So(r, convey.ShouldAlmostEqual, 1)
		}
		err := stats.Reset()
		convey.So(err, convey.ShouldBeNil)
		metrics := []string{DownloadFailStats, ErrorStats, ItemsStats, RequestStats}

		for _, f := range metrics {
			val := stats.Get(f)
			convey.So(val, convey.ShouldAlmostEqual, 0)
		}
	})
}
