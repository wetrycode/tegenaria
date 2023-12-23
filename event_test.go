package tegenaria

import (
	"errors"
	"fmt"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	c "github.com/smartystreets/goconvey/convey" // 别名导入
)

func TestEventWatcher(t *testing.T) {
	c.Convey("Watch events", t, func() {
		hooker := NewDefaultHooks()
		f := func() error {
			ch := make(chan EventType, 5)
			defer close(ch)
			ch <- START
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
		hooker := NewDefaultHooks()

		patch := gomonkey.ApplyFunc((*DefaultHooks).Start, func(_ *DefaultHooks, params ...interface{}) error { return errors.New("start error") })
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
	c.Convey("Watch pause event error", t, func() {
		patch := gomonkey.ApplyFunc((*DefaultHooks).Pause, func(_ *DefaultHooks, _ ...interface{}) error {
			return fmt.Errorf("pause error")
		})
		hooker := NewDefaultHooks()

		f := func() {
			ch := make(chan EventType, 1)
			ch <- PAUSE
			err := hooker.EventsWatcher(ch)
			if err != nil {
				panic(err)
			}
		}
		c.So(f, c.ShouldPanic)
		defer patch.Reset()
	})
	c.Convey("Watch ERROR event error", t, func() {
		patch := gomonkey.ApplyFunc((*DefaultHooks).Error, func(_ *DefaultHooks, _ ...interface{}) error {
			return fmt.Errorf("handle error event error")
		})
		hooker := NewDefaultHooks()

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
		patch := gomonkey.ApplyFunc((*DefaultHooks).Heartbeat, func(_ *DefaultHooks, _ ...interface{}) error {
			return fmt.Errorf("heartbeat error")
		})
		hooker := NewDefaultHooks()

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
		patch := gomonkey.ApplyFunc((*DefaultHooks).Exit, func(_ *DefaultHooks, _ ...interface{}) error {
			return fmt.Errorf("EXIT error")
		})
		hooker := NewDefaultHooks()

		f := func() {
			ch := make(chan EventType, 1)
			ch <- EXIT
			err := hooker.EventsWatcher(ch)
			if err != nil {
				panic(err)
			}
		}
		c.So(f, c.ShouldPanic)
		defer patch.Reset()
	})
}
