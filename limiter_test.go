package tegenaria

import (
	"sync"
	"testing"
	"time"

	"github.com/smartystreets/goconvey/convey"
)

func TestDeafultLimit(t *testing.T) {

	convey.Convey("test default limit", t, func() {
		limit := NewDefaultLimiter(16)
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
