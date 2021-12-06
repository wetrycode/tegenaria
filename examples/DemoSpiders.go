package examples

import "github.com/geebytes/go-scrapy/core"

type DemoSpider struct {
	Spider *core.Spider //
}

func NewDemoSpider(name string, urls []string) *DemoSpider {
	return &DemoSpider{
		Spider: core.NewSpider(name, urls),
	}
}

func (s *DemoSpider) StartRequest() error {
	return nil
}

func (s *DemoSpider) Parser() (*core.ItemProcesser, error) {
	return nil, nil
}

func (s *DemoSpider) ErrorHandler() {

}
