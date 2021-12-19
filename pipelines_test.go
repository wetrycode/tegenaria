package tegenaria

import (
	"fmt"
	"sort"
	"testing"
)

type TestItem struct {
	Text   string
	Author string
	Tags   string
}
type TestItemPipeline struct {
	Priority int
}
type TestItemPipeline2 struct {
	Priority int
}
type TestItemPipeline3 struct {
	Priority int
}

func (p *TestItemPipeline) ProcessItem(spider SpiderInterface, item ItemInterface) error {
	fmt.Printf("Spider %s run TestItemPipeline priority is %d\n", spider.GetName(), p.GetPriority())

	return nil

}
func (p *TestItemPipeline) GetPriority() int {
	return p.Priority
}
func (p *TestItemPipeline2) ProcessItem(spider SpiderInterface, item ItemInterface) error {
	fmt.Printf("Spider %s run TestItemPipeline2 priority is %d\n", spider.GetName(), p.GetPriority())
	return nil
}
func (p *TestItemPipeline2) GetPriority() int {
	return p.Priority
}

func (p *TestItemPipeline3) ProcessItem(spider SpiderInterface, item ItemInterface) error {
	fmt.Printf("Spider %s run TestItemPipeline3 priority is %d\n", spider.GetName(), p.GetPriority())
	return nil

}
func (p *TestItemPipeline3) GetPriority() int {
	return p.Priority
}

func TestPipelines(t *testing.T) {
	pipelines := make(ItemPipelines, 0)
	pipe := &TestItemPipeline{Priority: 0}
	pipe1 := &TestItemPipeline{Priority: 1}
	pipe2 := &TestItemPipeline{Priority: 3}
	pipe3 := &TestItemPipeline{Priority: 2}

	pipelines = append(pipelines, pipe)
	pipelines = append(pipelines, pipe1)
	pipelines = append(pipelines, pipe2)
	pipelines = append(pipelines, pipe3)
	sort.Sort(pipelines)
	for index, pipeline := range pipelines{
		if pipeline.GetPriority() != index{
			t.Errorf("pipeline priority %d error", pipeline.GetPriority())
		}
	}

}
