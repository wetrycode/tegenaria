package tegenaria

import (
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

func TestLeakyBucketLimiterWithRdb(t *testing.T) {

	mockRedis, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer mockRedis.Close()
	rdb := redis.NewClient(&redis.Options{
		Addr: mockRedis.Addr(),
	})
	limit := NewLeakyBucketLimiterWithRdb(16, rdb, getLimiterDefaultKey)
	start := time.Now()
	wg:=&sync.WaitGroup{}
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
	if interval <= 2{
		t.Errorf("LeakyBucketLimiter is not woking")
	}
	t.Logf("task interval is %f", interval)
}

func TestDeafultLimit(t *testing.T){
	limit:=NewDefaultLimiter(16)
	start := time.Now()
	wg:=&sync.WaitGroup{}
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
	if interval <= 2{
		t.Errorf("LeakyBucketLimiter is not woking")
	}
	t.Logf("task interval is %f", interval)
}