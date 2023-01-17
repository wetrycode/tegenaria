package tegenaria

import (
	"testing"

	"github.com/smartystreets/goconvey/convey"
)

type TestSpider struct {
	*BaseSpider
}

func (s *TestSpider) StartRequest(req chan<- *Context) {
	
	for _, url := range s.FeedUrls {
		request := NewRequest(url, GET, s.Parser)
		ctx := NewContext(request, s)
		req <- ctx
	}
}
func (s *TestSpider) Parser(resp *Context, req chan<- *Context) error {
	return testParser(resp, req)
}
func (s *TestSpider) ErrorHandler(err *Context, req chan<- *Context){

}
func (s *TestSpider) GetName() string {
	return s.Name
}

func TestSpiders(t *testing.T) {
	convey.Convey("test spiders",t,func(){
		spiders := NewSpiders()
		spider1 := &TestSpider{
			NewBaseSpider("testspider", []string{"https://www.baidu.com"}),
		}
		spider2 := &TestSpider{
			NewBaseSpider("testspider", []string{"https://www.baidu.com"}),
		}
		spider3 := &TestSpider{
			NewBaseSpider("testspider1", []string{"https://www.baidu.com"}),
		}
		spider4 := &TestSpider{
			NewBaseSpider("testspider2", []string{"https://www.baidu.com"}),
		}
		spider5 := &TestSpider{
			NewBaseSpider("", []string{"https://www.baidu.com"}),
		}
		err:=spiders.Register(spider1)
		convey.So(err, convey.ShouldBeNil)
	
		err = spiders.Register(spider2)
		convey.So(err, convey.ShouldBeError,ErrDuplicateSpiderName)

		err = spiders.Register(spider3)
		convey.So(err, convey.ShouldBeNil)
		err =spiders.Register(spider4)
		convey.So(err, convey.ShouldBeNil)
		spiderNames := []string{"testspider", "testspider1", "testspider2"}
		for _, spider := range spiderNames {
			_, err := spiders.GetSpider(spider)
			convey.So(err, convey.ShouldBeNil)
		}
		_, err1 := spiders.GetSpider("spider4")
		convey.So(err1, convey.ShouldBeError, ErrSpiderNotExist)
		err = spiders.Register(spider5)
		convey.So(err, convey.ShouldBeError, ErrEmptySpiderName)

	})

}
