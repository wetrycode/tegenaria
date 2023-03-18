package tegenaria

import (
	"sort"
	"testing"
)

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
