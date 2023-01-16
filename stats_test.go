package tegenaria

import (
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/smartystreets/goconvey/convey"
)

func TestDistributedStats(t *testing.T) {
	convey.Convey("test distribute stats", t, func() {
		mockRedis, err := miniredis.Run()
		if err != nil {
			panic(err)
		}
		defer mockRedis.Close()
		rdb := redis.NewClient(&redis.Options{
			Addr: mockRedis.Addr(),
		})
		wg := &sync.WaitGroup{}
		stats := NewDistributeStatistic("tegenaria:v1:stats", rdb, wg, DistributeStatisticAfterResetTTL(10*time.Second))
		stats.setCurrentSpider("distributedStatsSpider")

		worker := NewDistributedWorker(mockRedis.Addr(), NewDistributedWorkerConfig("", "", 0))
		worker.setCurrentSpider("distributedStatsSpider")
		err = worker.AddNode()
		convey.So(err, convey.ShouldBeNil)
		stats.IncrRequestSent()
		stats.IncrItemScraped()
		stats.IncrErrorCount()
		stats.IncrDownloadFail()
		result := stats.OutputStats()
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
		err = stats.Reset()
		convey.So(err, convey.ShouldBeNil)
		mockRedis.FastForward(20 * time.Second)
		for _, f := range funcs {
			val := f()
			convey.So(val, convey.ShouldAlmostEqual, 0)
		}
	})
}
