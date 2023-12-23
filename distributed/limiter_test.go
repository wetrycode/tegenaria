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
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/smartystreets/goconvey/convey"
)

func TestLeakyBucketLimiterWithRdb(t *testing.T) {
	convey.Convey("test leaky bucket limiter with rdb", t, func() {
		mockRedis := miniredis.RunT(t)

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
				err := limit.CheckAndWaitLimiterPass()
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
