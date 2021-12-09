package core

import (
	"sync"

	"github.com/geebytes/go-scrapy/middlewares"
	"github.com/geebytes/go-scrapy/pipelines"
)

type SpiderEnginer struct {
	SpidersModules map[string]*SpiderProcesser
	DownloaderMiddliers map[string] *middlewares.DownloadMiddlewares
	SpidersMiddliers map[string] *middlewares.SpidersMiddlewares
	Pipelines map[string] *pipelines.Pipelines

}

var Enginer *SpiderEnginer
var once sync.Once

func NewSpiderEnginer() *SpiderEnginer {
	once.Do(func() {
		Enginer = &SpiderEnginer{}
	})
	return Enginer
}

func (s *SpiderEnginer) Register(name string, spider *SpiderProcesser) {
	s.SpidersModules[name] = spider
}
func init(){
	Enginer = NewSpiderEnginer()
}
