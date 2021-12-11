package spiders

import (
	"sync"

	"github.com/geebytes/Tegenaria/exceptions"
	"github.com/geebytes/Tegenaria/items"
	"github.com/geebytes/Tegenaria/net"
)

type SpiderInterface interface {
	StartRequest() error
	Parser(net.Response) (*items.ItemInterface, error)
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

var once sync.Once
var SpidersList *Spiders

func NewBaseSpider(name string, feedUrls []string) *BaseSpider {
	return &BaseSpider{
		Name:     name,
		FeedUrls: feedUrls,
	}
}
func (s *BaseSpider) StartRequest() error {
	return nil
}
func (s *BaseSpider) Parser(net.Response) (*items.ItemInterface, error) {
	return nil, nil
}
func (s *BaseSpider) ErrorHandler() {

}
func NewSpiders() *Spiders {
	once.Do(func() {
		SpidersList = &Spiders{
			SpidersModules: make(map[string]SpiderInterface),
		}
	})
	return SpidersList
}
func (s *Spiders) Register(spider SpiderInterface) error {
	if len(spider.GetName()) == 0 {
		return exceptions.ErrEmptySpiderName
	}
	if _, ok := s.SpidersModules[spider.GetName()]; ok {
		return exceptions.ErrDuplicateSpiderName
	} else {
		s.SpidersModules[spider.GetName()] = spider
		return nil
	}
}
func (s *Spiders) GetSpider(name string) (SpiderInterface, error) {
	if _, ok := s.SpidersModules[name]; !ok {
		return nil, exceptions.ErrSpiderNotExist
	} else {
		return s.SpidersModules[name], nil
	}
}
