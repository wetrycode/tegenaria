package tegenaria

// import (
// 	"context"

// 	"github.com/geebytes/Tegenaria/exceptions"
// 	logger "github.com/geebytes/Tegenaria/logging"
// 	"github.com/geebytes/Tegenaria/http"
// 	"github.com/geebytes/Tegenaria/spiders"
// 	"github.com/liyue201/gostl/ds/priorityqueue"
// 	"github.com/sirupsen/logrus"
// )

// type MiddlewaresInterface interface {
// 	Do(cxt context.Context, spider *spiders.SpiderInterface, item chan interface{}) error
// 	// Register(name string, middleware interface{}) error
// 	GetPriority() int
// 	GetName() string
// }
// type DownloadMiddlewaresInterface interface {
// 	MiddlewaresInterface
// 	ProcessResponse(spider spiders.SpiderInterface, resp chan http.Response, result chan interface{}) error
// 	ProcessRequest(spider spiders.SpiderInterface, req chan http.Request, result chan interface{}) error
// 	ProcessException(spider spiders.SpiderInterface, req chan http.Request, resp chan http.Response)
// }

// type MiddlerwaresHandlerInterface interface {
// 	Add(middleware MiddlewaresInterface)
// 	Do(ctx context.Context, spider *spiders.SpiderInterface, item chan interface{}) error
// }

// var middlerwareLog *logrus.Entry = logger.GetLogger("spidermiddlerware")

// func MiddlerwaresTypeComparator(a, b interface{}) int {
// 	aPriority := a.(MiddlewaresInterface).GetPriority()
// 	bPriority := b.(MiddlewaresInterface).GetPriority()
// 	if aPriority == bPriority {
// 		return 0
// 	}
// 	if aPriority < bPriority {
// 		return -1
// 	} else {
// 		return 1
// 	}

// }

// type MiddlewaresHandlers struct {
// 	Middlewares *priorityqueue.PriorityQueue
// 	Name        string
// }

// func (s *MiddlewaresHandlers) Add(middleware MiddlewaresInterface) {
// 	s.Middlewares.Push(middleware)
// }

// func (s *MiddlewaresHandlers) Do(ctx context.Context, spider *spiders.SpiderInterface, ch chan interface{}) error {
// 	doMiddlewares := new(priorityqueue.PriorityQueue)
// 	*doMiddlewares = *s.Middlewares
// 	for !doMiddlewares.Empty() {
// 		middleware := doMiddlewares.Pop().(MiddlewaresInterface)
// 		err := middleware.Do(ctx, spider, ch)
// 		if err != nil {
// 			middlerwareLog.Errorf("Do %s handler error %s", middleware.GetName(), err.Error())
// 			return exceptions.ErrSpiderMiddleware
// 		}
// 	}
// 	return nil
// }
// func NewMiddlewaresHandlers(name string) *MiddlewaresHandlers {
// 	return &MiddlewaresHandlers{
// 		Middlewares: priorityqueue.New(priorityqueue.WithComparator(MiddlerwaresTypeComparator), priorityqueue.WithGoroutineSafe()),
// 		Name:        name,
// 	}
// }
