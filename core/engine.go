package core

import (
	"sync"

	logger "github.com/geebytes/Tegenaria/logging"

	"github.com/geebytes/Tegenaria/items"
	"github.com/geebytes/Tegenaria/middlewares"
	"github.com/geebytes/Tegenaria/net"
	"github.com/geebytes/Tegenaria/spiders"
	"github.com/sirupsen/logrus"
)

type SpiderEnginer struct {
	DownloaderMiddlerwaresHandlers *middlewares.MiddlewaresHandlers
	SpidersMiddlerwaresHandlers    *middlewares.MiddlewaresHandlers
	PipelinesHandlers              *middlewares.MiddlewaresHandlers
	Spiders                        *spiders.Spiders
	RequestsChan                   chan *net.Request
	SpidersChan                    chan *spiders.SpiderInterface
	ItemsChan                      chan *items.ItemInterface
	RequestConcurrent              int
	ItemConcurrent                 int
	SpidersConcurrent              int
}

var Enginer *SpiderEnginer
var once sync.Once
var enginerLog *logrus.Entry = logger.GetLogger("enginer")

type Option func(r *SpiderEnginer)

func (e *SpiderEnginer) Start() {
	defer func() {
		e.Close()
		if p := recover(); p != nil {
			enginerLog.Errorf("Close engier fail")
		}
	}()
	for {
		select {
		case spider := <-e.SpidersChan:
			go e.DoSpidersMiddlerwaresHandlers(spider)
		case req := <-e.RequestsChan:
			go e.DoDownloaderMiddlerwaresHandlers(req)
		case item := <-e.ItemsChan:
			go e.DoPipelinesHandlers(item)
		default:
			enginerLog.Info("Enginer is waiting task schedule")

		}
	}
}
func (e *SpiderEnginer) Close() {
	defer func() {
		if p := recover(); p != nil {
			enginerLog.Errorf("Close engier fail")
			panic("Close engier fail")
		}
	}()
	once.Do(func() {
		close(e.RequestsChan)
		close(e.SpidersChan)
		close(e.ItemsChan)
	})
}
func (e *SpiderEnginer) DoSpidersMiddlerwaresHandlers(spider *spiders.SpiderInterface) {

}
func (e *SpiderEnginer) DoDownloaderMiddlerwaresHandlers(req *net.Request) {

}
func (e *SpiderEnginer) DoPipelinesHandlers(item *items.ItemInterface) {

}
func WithRequestConcurrent(concurrent int) Option {
	return func(r *SpiderEnginer) {
		r.RequestConcurrent = concurrent
		r.RequestsChan = make(chan *net.Request, concurrent)
	}
}
func WithItemConcurrent(concurrent int) Option {
	return func(r *SpiderEnginer) {
		r.ItemConcurrent = concurrent
		r.ItemsChan = make(chan *items.ItemInterface, concurrent)
	}
}
func WithSpidersConcurrent(concurrent int) Option {
	return func(r *SpiderEnginer) {
		r.SpidersConcurrent = concurrent
		r.ItemsChan = make(chan *items.ItemInterface, concurrent)
	}
}
func NewSpiderEnginer(opts ...Option) *SpiderEnginer {
	once.Do(func() {
		Enginer = &SpiderEnginer{
			DownloaderMiddlerwaresHandlers: middlewares.NewMiddlewaresHandlers("downloader"),
			SpidersMiddlerwaresHandlers:    middlewares.NewMiddlewaresHandlers("spiders"),
			PipelinesHandlers:              middlewares.NewMiddlewaresHandlers("pipelines"),
			Spiders:                        spiders.NewSpiders(),
			RequestsChan:                   make(chan *net.Request, 256),
			SpidersChan:                    make(chan *spiders.SpiderInterface, 256),
			ItemsChan:                      make(chan *items.ItemInterface, 256),
		}
		for _, o := range opts {
			o(Enginer)
		}
	})
	return Enginer
}

func init() {
	Enginer = NewSpiderEnginer()

}
