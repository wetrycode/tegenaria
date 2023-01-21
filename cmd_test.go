package tegenaria

import (
	"bytes"
	"testing"

	"github.com/smartystreets/goconvey/convey"
)

func TestCmdStart(t *testing.T) {
	convey.Convey("test cmd", t, func() {
		engine := newTestEngine("testCmdSpider")
		buf := new(bytes.Buffer)
		rootCmd.SetOut(buf)
		rootCmd.SetErr(buf)
		rootCmd.SetArgs([]string{"crawl", "testCmdSpider"})
		convey.So(engine.Execute, convey.ShouldNotPanic)
	})
}
