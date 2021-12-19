package tegenaria

import (
	"testing"
)

type TestSpider struct {
	*BaseSpider
}

func (s *TestSpider) StartRequest(req chan<- *Request) {
}
func (s *TestSpider) Parser(resp *Response, item chan<- ItemInterface, req chan<- *Request) {
}
func (s *TestSpider) ErrorHandler() {

}
func (s *TestSpider) GetName() string {
	return s.Name
}

func TestSpiders(t *testing.T) {
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
	spiders.Register(spider1)
	err := spiders.Register(spider2)
	if err == nil {
		t.Error("Register duplicate spider name")
	} else {
		if err.Error() != ErrDuplicateSpiderName.Error() {
			t.Errorf("Unexpected error register %s", err.Error())
		}
	}
	spiders.Register(spider3)
	spiders.Register(spider4)
	spiderNames := []string{"testspider", "testspider1", "testspider2"}
	for _, spider := range spiderNames {
		_, err := spiders.GetSpider(spider)
		if err != nil {
			t.Errorf("Get spider by name error %s", err.Error())
		}
	}
	_, err1 := spiders.GetSpider("spider4")
	if err1.Error() != ErrSpiderNotExist.Error() {
		t.Errorf("Get spider by name unexpected error %s", err.Error())

	}

}
