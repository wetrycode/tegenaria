package tegenaria

import "sync"

type SpiderInterface interface {
	StartRequest(req chan<- *Request)
	Parser(resp *Response, item chan<- ItemInterface, req chan<- *Request)
	ErrorHandler()
	GetName() string
}
type BaseSpider struct {
	Name     string
	FeedUrls []string
}

type Spiders struct {
	SpidersModules map[string]SpiderInterface
}

var SpidersList *Spiders
var onceSpiders sync.Once

func NewBaseSpider(name string, feedUrls []string) *BaseSpider {
	return &BaseSpider{
		Name:     name,
		FeedUrls: feedUrls,
	}
}
func (s *BaseSpider) StartRequest(req chan<- *Request) {
}
func (s *BaseSpider) Parser(resp *Response, item chan<- *ItemInterface, req chan<- *Request) {
}
func (s *BaseSpider) ErrorHandler() {

}
func NewSpiders() *Spiders {
	onceSpiders.Do(func() {
		SpidersList = &Spiders{
			SpidersModules: make(map[string]SpiderInterface),
		}
	})
	return SpidersList
}
func (s *Spiders) Register(spider SpiderInterface) error {
	if len(spider.GetName()) == 0 {
		return ErrEmptySpiderName
	}
	if _, ok := s.SpidersModules[spider.GetName()]; ok {
		return ErrDuplicateSpiderName
	} else {
		s.SpidersModules[spider.GetName()] = spider
		return nil
	}
}
func (s *Spiders) GetSpider(name string) (SpiderInterface, error) {
	if _, ok := s.SpidersModules[name]; !ok {
		return nil, ErrSpiderNotExist
	} else {
		return s.SpidersModules[name], nil
	}
}
