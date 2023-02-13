package tegenaria

import (
	"bytes"
	"testing"
	"time"

	"github.com/smartystreets/goconvey/convey"
)

func TestCmdStart(t *testing.T) {
	convey.Convey("test cmd", t, func() {
		engine := newTestEngine("testCmdSpider")
		buf := new(bytes.Buffer)
		defer func() {
			rootCmd.ResetFlags()
			rootCmd.ResetCommands()
		}()
		rootCmd.SetOut(buf)
		rootCmd.SetErr(buf)
		rootCmd.SetArgs([]string{"crawl", "testCmdSpider"})
		convey.So(engine.Execute, convey.ShouldNotPanic)

	})
	convey.Convey("test cmd with interval", t, func() {
		defer rootCmd.ResetCommands()

		engine := newTestEngine("testCmdIntervalSpider", EngineWithInterval(4*time.Second), EngineWithUniqueReq(false))
		rootCmd.SetArgs([]string{"crawl", "testCmdIntervalSpider", "-i"})
		go engine.Execute()
		time.Sleep(5 * time.Second)
		convey.So(engine.statistic.GetRequestSent(), convey.ShouldAlmostEqual, 2)
		convey.So(engine.statistic.GetItemScraped(), convey.ShouldAlmostEqual, 2)
	})
}
