package distributed

import (
	"testing"

	"github.com/smartystreets/goconvey/convey"
	"github.com/wetrycode/tegenaria"
)

func TestInvaildParse(t *testing.T) {
	convey.Convey("test invalid parse funcation", t, func() {
		request := tegenaria.NewRequest("http://www.example.com", tegenaria.GET, func(resp *tegenaria.Context, req chan<- *tegenaria.Context) error { return nil })
		spider1 := &tegenaria.TestSpider{
			BaseSpider: tegenaria.NewBaseSpider("testspider", []string{"https://www.example.com"}),
		}
		spiderName := spider1.GetName()
		f := func() {
			_, err := newRdbCache(request, "xxxxxxx", spiderName)
			convey.So(err, convey.ShouldBeNil)
		}
		convey.So(f, convey.ShouldPanic)
	})

}
