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
	"errors"
	"fmt"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/alicebob/miniredis/v2"
	c "github.com/smartystreets/goconvey/convey" // 别名导入
	"github.com/wetrycode/tegenaria"
)

func watcher(t *testing.T, spider string) (tegenaria.DistributedWorkerInterface, *miniredis.Miniredis) {
	mockRedis := miniredis.RunT(t)
	config := NewDistributedWorkerConfig(NewRedisConfig(mockRedis.Addr(), "", "", 0), &InfluxdbConfig{})

	worker := NewDistributedWorker(config)
	return worker, mockRedis

}
func TestEventWatcher(t *testing.T) {
	c.Convey("Watch events", t, func() {

		woker, r := watcher(t, "eventWatcherSpider")
		defer r.Close()
		hooker := NewDistributedHooks(woker)
		f := func() error {
			ch := make(chan tegenaria.EventType, 5)
			defer close(ch)
			ch <- tegenaria.START
			ch <- tegenaria.PAUSE
			ch <- tegenaria.ERROR
			ch <- tegenaria.HEARTBEAT
			ch <- tegenaria.EXIT
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
			ch := make(chan tegenaria.EventType, 1)
			defer close(ch)

			ch <- tegenaria.START
			err := hooker.EventsWatcher(ch)
			if err != nil {
				panic(err)
			}
		}
		c.So(f, c.ShouldPanic)
		defer patch.Reset()
	})
	c.Convey("Watch pause event error", t, func() {
		patch := gomonkey.ApplyFunc((*DistributedHooks).Pause, func(_ *DistributedHooks, _ ...interface{}) error {
			return fmt.Errorf("stop error")
		})
		worker, r := watcher(t, "stopErrorSpider")
		defer r.Close()
		hooker := NewDistributedHooks(worker)

		f := func() {
			ch := make(chan tegenaria.EventType, 1)
			ch <- tegenaria.PAUSE
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
			ch := make(chan tegenaria.EventType, 1)
			ch <- tegenaria.ERROR
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
			ch := make(chan tegenaria.EventType, 1)
			ch <- tegenaria.HEARTBEAT
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
			ch := make(chan tegenaria.EventType, 1)
			ch <- tegenaria.EXIT
			err := hooker.EventsWatcher(ch)
			if err != nil {
				panic(err)
			}
		}
		c.So(f, c.ShouldNotPanic)
		defer patch.Reset()
	})
}
