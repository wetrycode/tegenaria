package tegenaria

type PipelinesInterface interface {
	GetPriority() int
	ProcessItem(spider SpiderInterface, item ItemInterface) error
}
type PipelinesBase struct {
	Priority int
}

type ItemPipelines []PipelinesInterface

func (p ItemPipelines) Len() int           { return len(p) }
func (p ItemPipelines) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p ItemPipelines) Less(i, j int) bool { return p[i].GetPriority() < p[j].GetPriority() }
