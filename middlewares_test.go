package tegenaria

// import (
// 	"context"
// 	"testing"

// 	"github.com/geebytes/Tegenaria/items"
// 	"github.com/geebytes/Tegenaria/http"
// 	"github.com/geebytes/Tegenaria/spiders"
// )

// type TestSpiderMiddlerwares struct {
// 	Name     string
// 	Priority int
// }
// type TestSpider struct {
// 	Base     *spiders.BaseSpider
// 	Priority map[string]int
// }

// func (s *TestSpider) StartRequest() error {
// 	return nil
// }
// func (s *TestSpider) Parser(http.Response) (*items.ItemInterface, error) {
// 	return nil, nil
// }
// func (s *TestSpider) ErrorHandler() {

// }
// func (s *TestSpiderMiddlerwares) Do(cxt context.Context, spider *spiders.SpiderInterface, ch chan interface{}) error {
// 	// respResult := <-resp

// 	// testItem.Priority[spider.Name] = s.Priority
// 	// for {
// 	// 	select
// 	// }
// 	go func ()  {
// 		for {
// 			select{
// 			case msg :=<-ch:
// 				switch v:=msg.(type){
// 				case *http.Response:
// 					s.ProcessSpiderInput(spider, v)
// 				case *items.ItemInterface:
// 					s.
// 				}
// 			}
// 		}
// 	}()
// 	return nil
// }
// func (s *TestSpiderMiddlerwares)ProcessSpiderInput(spider *spiders.SpiderInterface, resp *http.Response){

// }
// func (s *TestSpiderMiddlerwares)ProcessSpiderOutput(spider *spiders.SpiderInterface, resp *http.Response, resutl chan items.ItemInterface){

// }

// func (s *TestSpiderMiddlerwares) GetPriority() int {
// 	return s.Priority
// }
// func (s *TestSpiderMiddlerwares) GetName() string {
// 	return s.Name
// }
// func TestSpiderMiddlewares(t *testing.T) {
// 	spiderMiddlerwares := NewMiddlewaresHandlers("SpiderMiddleware")
// 	m1 := &TestSpiderMiddlerwares{
// 		Name:     "SPM1",
// 		Priority: 1,
// 	}
// 	m2 := &TestSpiderMiddlerwares{
// 		Name:     "SPM2",
// 		Priority: 2,
// 	}
// 	m3 := &TestSpiderMiddlerwares{
// 		Name:     "SPM3",
// 		Priority: 3,
// 	}
// 	spiderMiddlerwares.Add(m1)
// 	spiderMiddlerwares.Add(m2)

// 	spiderMiddlerwares.Add(m3)

// 	item := &TestSpider{
// 		Base: spiders.NewBaseSpider("tesspider", []string{}),

// 		Priority: map[string]int{},
// 	}
// 	ctx := context.TODO()
// 	except := map[string]int{"SPM1": 1, "SPM2": 2, "SPM3": 3}
// 	spiderMiddlerwares.Do(ctx, item)
// 	spiderMiddlerwares.Do(ctx, item)
// 	if len(item.Priority) == 0 {
// 		t.Errorf("Spider Middlerware test err")
// 	}
// 	for key, value := range item.Priority {
// 		if val, ok := except[key]; ok {
// 			if val != value {
// 				t.Errorf("Spider Middlerware test err")
// 			}
// 		} else {
// 			t.Errorf("Spider Middlerware test err")
// 		}

// 	}
// }
