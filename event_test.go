package tegenaria

import (
	"errors"
	"fmt"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/alicebob/miniredis/v2"
	c "github.com/smartystreets/goconvey/convey" // 别名导入
)

func watcher(t *testing.T, spider string) (DistributedWorkerInterface, *miniredis.Miniredis) {
	config := NewDistributedWorkerConfig("", "", 0)
	mockRedis := miniredis.RunT(t)
	worker := NewDistributedWorker(mockRedis.Addr(), config)
	// engine := newTestEngine(spider, EngineWithDistributedWorker(worker))
	worker.setCurrentSpider(spider)
	return worker, mockRedis

}
func TestEventWatcher(t *testing.T) {
	c.Convey("Watch events", t, func() {

		woker, r := watcher(t, "eventWatcherSpider")
		defer r.Close()
		hooker := NewDistributedHooks(woker)
		f := func() error {
			ch := make(chan EventType, 5)
			defer close(ch)
			ch <- START
			ch <- STOP
			ch <- ERROR
			ch <- HEARTBEAT
			ch <- EXIT
			return hooker.EventsWatcher(ch)

		}
		c.So(f(), c.ShouldBeNil)

	})
}

func TestEventWatcherWithError(t *testing.T) {

	c.Convey("Watch start event error", t, func() {
		worker, r := watcher(t, "startErrorSpider")
		defer r.Close()
		hooker := NewDistributedHooks(worker)
		patch := gomonkey.ApplyFunc((*DistributedHooks).Start, func(_ *DistributedHooks, params ...interface{}) error { return errors.New("start error") })
		f := func() {
			ch := make(chan EventType, 1)
			defer close(ch)

			ch <- START
			err := hooker.EventsWatcher(ch)
			if err != nil {
				panic(err)
			}
		}
		c.So(f, c.ShouldPanic)
		defer patch.Reset()
	})
	c.Convey("Watch stop event error", t, func() {
		patch := gomonkey.ApplyFunc((*DistributedHooks).Stop, func(_ *DistributedHooks, _ ...interface{}) error {
			return fmt.Errorf("stop error")
		})
		worker, r := watcher(t, "stopErrorSpider")
		defer r.Close()
		hooker := NewDistributedHooks(worker)

		f := func() {
			ch := make(chan EventType, 1)
			ch <- STOP
			err := hooker.EventsWatcher(ch)
			if err != nil {
				panic(err)
			}
		}
		c.So(f, c.ShouldPanic)
		defer patch.Reset()
	})
	c.Convey("Watch ERROR event error", t, func() {
		patch := gomonkey.ApplyFunc((*DistributedHooks).Error, func(_ *DistributedHooks, _ ...interface{}) error {
			return fmt.Errorf("handle error event error")
		})
		worker, r := watcher(t, "ErrorSpider")
		defer r.Close()
		hooker := NewDistributedHooks(worker)

		f := func() {
			ch := make(chan EventType, 1)
			ch <- ERROR
			err := hooker.EventsWatcher(ch)
			if err != nil {
				panic(err)
			}
		}
		c.So(f, c.ShouldNotPanic)
		defer patch.Reset()
	})
	c.Convey("Watch heartbeat event error", t, func() {
		patch := gomonkey.ApplyFunc((*DistributedHooks).Heartbeat, func(_ *DistributedHooks, _ ...interface{}) error {
			return fmt.Errorf("heartbeat error")
		})
		worker, r := watcher(t, "hearbeatErrorSpider")
		defer r.Close()
		hooker := NewDistributedHooks(worker)

		f := func() {
			ch := make(chan EventType, 1)
			ch <- HEARTBEAT
			err := hooker.EventsWatcher(ch)
			if err != nil {
				panic(err)
			}
		}
		c.So(f, c.ShouldPanic)
		defer patch.Reset()
	})

	c.Convey("Watch EXIT event error", t, func() {
		patch := gomonkey.ApplyFunc((*DistributedHooks).Heartbeat, func(_ *DistributedHooks, _ ...interface{}) error {
			return fmt.Errorf("EXIT error")
		})
		worker, r := watcher(t, "exitErrorSpider")
		defer r.Close()
		hooker := NewDistributedHooks(worker)

		f := func() {
			ch := make(chan EventType, 1)
			ch <- EXIT
			err := hooker.EventsWatcher(ch)
			if err != nil {
				panic(err)
			}
		}
		c.So(f, c.ShouldNotPanic)
		defer patch.Reset()
	})
}
