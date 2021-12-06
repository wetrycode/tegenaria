package core

import "sync"

type SpiderEnginer struct {
	SpidersModules []*SpiderProcesser
}

var Enginer *SpiderEnginer
var once sync.Once

func NewSpiderEnginer() *SpiderEnginer {
	once.Do(func() {
		Enginer = &SpiderEnginer{}
	})
	return Enginer
}

func (s *SpiderEnginer) Register(spider *SpiderProcesser) {
	s.SpidersModules = append(s.SpidersModules, spider)
}
