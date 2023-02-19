package tegenaria

import (
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/smartystreets/goconvey/convey"
)

func TestLeakyBucketLimiterWithRdb(t *testing.T) {
	convey.Convey("test leaky bucket limiter with rdb", t, func() {
		mockRedis, err := miniredis.Run()
		if err != nil {
			panic(err)
		}
		defer mockRedis.Close()
		rdb := redis.NewClient(&redis.Options{
			Addr: mockRedis.Addr(),
		})
		f := func() (string, time.Duration) {
			return "tegenaria:v1:limiter", 5 * time.Second
		}
		limit := NewLeakyBucketLimiterWithRdb(16, rdb, f)
		limit.setCurrrentSpider("testLeakyBucketLimiterSpider")
		start := time.Now()
		wg := &sync.WaitGroup{}
		for i := 0; i < 64; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := limit.checkAndWaitLimiterPass()
				if err != nil {
					t.Errorf("checkAndWaitLimiterPass error %s", err.Error())
				}
			}()
		}
		wg.Wait()
		interval := time.Since(start).Seconds()
		convey.So(interval, convey.ShouldBeGreaterThan, 2)
		t.Logf("task interval is %f", interval)

	})
}

func TestDeafultLimit(t *testing.T) {

	convey.Convey("test default limit", t, func() {
		limit := NewDefaultLimiter(16)
		start := time.Now()
		wg := &sync.WaitGroup{}
		for i := 0; i < 64; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := limit.checkAndWaitLimiterPass()
				if err != nil {
					t.Errorf("checkAndWaitLimiterPass error %s", err.Error())
				}
			}()
		}
		wg.Wait()
		interval := time.Since(start).Seconds()
		convey.So(interval, convey.ShouldBeGreaterThan, 2)
		t.Logf("task interval is %f", interval)
	})
}
