package tegenaria

import (
	"testing"

	"github.com/smartystreets/goconvey/convey"
)

func TestIncrNewMetric(t *testing.T) {
	convey.Convey("Test a new metric incr", t, func() {
		d := NewDefaultStatistic()
		d.Incr("403")
	})
}
