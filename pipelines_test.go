package tegenaria

import (
	"sort"
	"testing"
)

type testItem struct {
	test      string
	pipelines []int
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

func (p *TestItemPipeline) ProcessItem(spider SpiderInterface, item *ItemMeta) error {
	i := item.Item.(*testItem)
	i.pipelines = append(i.pipelines, p.Priority)
	return nil

}
func (p *TestItemPipeline) GetPriority() int {
	return p.Priority
}
func (p *TestItemPipeline2) ProcessItem(spider SpiderInterface, item *ItemMeta) error {
	i := item.Item.(*testItem)
	i.pipelines = append(i.pipelines, p.Priority)
	return nil
}
func (p *TestItemPipeline2) GetPriority() int {
	return p.Priority
}

func (p *TestItemPipeline3) ProcessItem(spider SpiderInterface, item *ItemMeta) error {
	i := item.Item.(*testItem)
	i.pipelines = append(i.pipelines, p.Priority)
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
	for index, pipeline := range pipelines {
		if pipeline.GetPriority() != index {
			t.Errorf("pipeline priority %d error", pipeline.GetPriority())
		}
	}

}
