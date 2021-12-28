package middlewares

import "github.com/geebytes/tegenaria"

type MiddlewaresInterface interface {
	GetPriority() int
	ProcessItem(spider tegenaria.SpiderInterface, item tegenaria.ItemInterface) error
}
type MiddlewaresBase struct {
	Priority int
}

type Middlewares []MiddlewaresInterface

func (p Middlewares) Len() int           { return len(p) }
func (p Middlewares) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p Middlewares) Less(i, j int) bool { return p[i].GetPriority() < p[j].GetPriority() }
