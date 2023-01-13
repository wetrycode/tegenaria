package quotes

import "github.com/wetrycode/tegenaria"

// QuotesbotItemPipeline an example of tegenaria.PipelinesInterface interface
// You need to set priority of each piplines
// These pipeline will handle all item order by priority
// Priority is that the lower the number, the higher the priority
type QuotesbotItemPipeline struct {
	Priority int
}
type QuotesbotItemPipeline2 struct {
	Priority int
}
type QuotesbotItemPipeline3 struct {
	Priority int
}

// ProcessItem funcation of tegenaria.PipelinesInterface,it is used to handle item
func (p *QuotesbotItemPipeline) ProcessItem(spider tegenaria.SpiderInterface, item *tegenaria.ItemMeta) error {
	exampleLog.Infof("QuotesbotItemPipeline is running!")
	return nil

}

// GetPriority get priority of pipline
func (p *QuotesbotItemPipeline) GetPriority() int {
	return p.Priority
}
func (p *QuotesbotItemPipeline2) ProcessItem(spider tegenaria.SpiderInterface, item *tegenaria.ItemMeta) error {
	exampleLog.Infof("QuotesbotItemPipeline2 is running!")

	return nil
}
func (p *QuotesbotItemPipeline2) GetPriority() int {
	return p.Priority
}

func (p *QuotesbotItemPipeline3) ProcessItem(spider tegenaria.SpiderInterface, item *tegenaria.ItemMeta) error {
	exampleLog.Infof("QuotesbotItemPipeline3 is running!")

	return nil

}
func (p *QuotesbotItemPipeline3) GetPriority() int {
	return p.Priority
}

