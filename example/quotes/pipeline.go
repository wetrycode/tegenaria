package quotes

import "github.com/wetrycode/tegenaria"

// QuotesbotItemPipeline tegenaria.PipelinesInterface 接口示例
// 用于item处理的pipeline
type QuotesbotItemPipeline struct {
	Priority int
}
type QuotesbotItemPipeline2 struct {
	Priority int
}
type QuotesbotItemPipeline3 struct {
	Priority int
}

// ProcessItem item处理函数
func (p *QuotesbotItemPipeline) ProcessItem(spider tegenaria.SpiderInterface, item *tegenaria.ItemMeta) error {
	i := item.Item.(*QuotesbotItem)
	exampleLog.Infof("%s 抓取到数据:%s", item.CtxId, i.Text)
	return nil

}

// GetPriority 获取该pipeline的优先级
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
